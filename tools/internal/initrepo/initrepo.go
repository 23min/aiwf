// Package initrepo implements `aiwf init`: idempotent first-time
// setup for a consumer repo. See docs/poc-plan.md Session 3 for the
// full contract.
//
// The package never produces a git commit — it writes/scaffolds and
// reports back; the user commits when ready. It is also safe to re-run:
// pre-existing files (aiwf.yaml, CLAUDE.md, custom .gitignore content)
// are preserved verbatim; skills are always wiped-and-rewritten per the
// cache contract.
package initrepo

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/23min/ai-workflow-v2/tools/internal/config"
	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
	"github.com/23min/ai-workflow-v2/tools/internal/skills"
)

// preHookMarker is the exact comment line `aiwf init` writes inside its
// managed pre-push hook. Re-running init detects this marker to know
// whether overwriting is safe (we own the hook) vs. refusing (a
// pre-existing user hook has its own logic).
const preHookMarker = "# aiwf:pre-push"

// shellQuoteSingle wraps s in single quotes for safe /bin/sh
// interpolation. Single quotes prevent every shell expansion; to
// embed a literal single quote we close the quote, write a
// backslash-escaped quote, then reopen — see POSIX shell quoting.
func shellQuoteSingle(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// CLAUDETemplate is the boilerplate written to CLAUDE.md when no file
// exists. Short by design — consumers customize it freely.
const CLAUDETemplate = `# CLAUDE.md

This repository uses [aiwf](https://github.com/23min/ai-workflow-v2) to track planning state.

## Quick reference

- ` + "`aiwf check`" + ` — validate the planning tree.
- ` + "`aiwf add <kind> --title \"...\"`" + ` — create an entity (epic, milestone, adr, gap, decision, contract).
- ` + "`aiwf promote <id> <status>`" + ` — advance status.
- ` + "`aiwf history <id>`" + ` — show what happened to an entity.

The pre-push hook runs ` + "`aiwf check`" + ` automatically; broken state cannot be pushed.

Skills under ` + "`.claude/skills/wf-*/`" + ` are gitignored and regenerated on ` + "`aiwf update`" + `.
`

// preHookScript renders the body of the pre-push hook installed by
// init, with the absolute path of the binary baked in. Hardcoding
// the path is more robust than relying on `$PATH` at push time —
// `git push` runs hooks under whatever shell git chose, and the
// user's interactive PATH may not match. Re-running `aiwf init`
// after a binary upgrade refreshes the path (idempotent because the
// marker tells us we own the hook).
func preHookScript(execPath string) string {
	return `#!/bin/sh
` + preHookMarker + `
# Installed by aiwf init. To customize, replace this hook with one
# managed by husky/lefthook (etc.) and call ` + "`aiwf check`" + ` from there.
exec ` + shellQuoteSingle(execPath) + ` check
`
}

// ErrPreHookConflict is returned when a pre-push hook exists without
// the aiwf marker. Init refuses to clobber user-authored hooks; the
// caller surfaces a remediation instruction.
var ErrPreHookConflict = errors.New("pre-push hook exists without aiwf marker")

// Action classifies what init did for a single step. The CLI uses this
// to render a friendly summary.
type Action string

// Action values reported per step.
const (
	ActionCreated   Action = "created"
	ActionPreserved Action = "preserved"
	ActionUpdated   Action = "updated"
)

// StepResult is one line of init's per-step ledger.
type StepResult struct {
	What   string
	Action Action
	Detail string
}

// Result is the per-step ledger init returns. Order matches the order
// of operations.
type Result struct {
	Steps []StepResult
}

// Options carries init-time inputs that override or supplement the
// defaults. ActorOverride bypasses git-config derivation when set.
// AiwfVersion stamps aiwf.yaml's `aiwf_version`; the CLI passes the
// binary's Version constant.
type Options struct {
	ActorOverride string
	AiwfVersion   string
}

// Init runs the documented setup steps in order. Returns a Result that
// describes what was created vs preserved vs updated. Errors abort
// early — a partially-applied init is rare in practice (init only
// touches config / scaffolding / skills) and the user can re-run.
func Init(ctx context.Context, root string, opts Options) (*Result, error) {
	if opts.AiwfVersion == "" {
		return nil, errors.New("AiwfVersion is required")
	}

	res := &Result{}

	// 1. aiwf.yaml — write only if missing.
	cfgStep, err := ensureConfig(root, opts)
	if err != nil {
		return nil, err
	}
	res.Steps = append(res.Steps, cfgStep)

	// 2. Scaffold entity directories.
	scaffoldSteps, err := scaffoldDirs(root)
	if err != nil {
		return nil, err
	}
	res.Steps = append(res.Steps, scaffoldSteps...)

	// 3. Materialize skills (wipe-and-rewrite per cache contract).
	if mErr := skills.Materialize(root); mErr != nil {
		return nil, fmt.Errorf("materializing skills: %w", mErr)
	}
	res.Steps = append(res.Steps, StepResult{
		What:   ".claude/skills/wf-*",
		Action: ActionUpdated,
		Detail: "materialized from embedded skills",
	})

	// 4. Append skill paths to .gitignore.
	gitignoreStep, err := ensureGitignore(root)
	if err != nil {
		return nil, err
	}
	res.Steps = append(res.Steps, gitignoreStep)

	// 5. CLAUDE.md template — write only if missing.
	claudeStep, err := ensureClaudeMd(root)
	if err != nil {
		return nil, err
	}
	res.Steps = append(res.Steps, claudeStep)

	// 6. Pre-push hook — install or overwrite-if-marker-present.
	hookStep, err := ensurePreHook(ctx, root)
	if err != nil {
		return nil, err
	}
	res.Steps = append(res.Steps, hookStep)

	return res, nil
}

func ensureConfig(root string, opts Options) (StepResult, error) {
	path := filepath.Join(root, config.FileName)
	if _, err := os.Stat(path); err == nil {
		return StepResult{What: config.FileName, Action: ActionPreserved}, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return StepResult{}, fmt.Errorf("statting %s: %w", config.FileName, err)
	}

	actor, err := deriveActor(opts.ActorOverride, root)
	if err != nil {
		return StepResult{}, err
	}

	cfg := &config.Config{
		AiwfVersion: opts.AiwfVersion,
		Actor:       actor,
	}
	if err := config.Write(root, cfg); err != nil {
		return StepResult{}, err
	}
	return StepResult{
		What:   config.FileName,
		Action: ActionCreated,
		Detail: "actor=" + actor,
	}, nil
}

// deriveActor follows the documented precedence: explicit > git
// config user.email derivation. The git lookup runs inside root so
// the consumer repo's local config wins over the host's global.
// Errors if neither yields a valid actor (so init fails loudly
// rather than writing aiwf.yaml without an actor field).
func deriveActor(override, root string) (string, error) {
	if override != "" {
		if !config.ActorPattern.MatchString(override) {
			return "", fmt.Errorf("--actor %q must match <role>/<identifier> (single '/', no whitespace)", override)
		}
		return override, nil
	}
	cmd := exec.Command("git", "config", "user.email")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", errors.New("no actor: pass --actor <role>/<identifier> or set git config user.email")
	}
	email := strings.TrimSpace(string(out))
	at := strings.IndexByte(email, '@')
	if at <= 0 {
		return "", fmt.Errorf("git config user.email %q has no local part; pass --actor <role>/<identifier>", email)
	}
	candidate := "human/" + email[:at]
	if !config.ActorPattern.MatchString(candidate) {
		return "", fmt.Errorf("derived actor %q is not in <role>/<identifier> form; pass --actor explicitly", candidate)
	}
	return candidate, nil
}

func scaffoldDirs(root string) ([]StepResult, error) {
	dirs := []string{
		filepath.Join("work", "epics"),
		filepath.Join("work", "gaps"),
		filepath.Join("work", "decisions"),
		filepath.Join("work", "contracts"),
		filepath.Join("docs", "adr"),
	}
	out := make([]StepResult, 0, len(dirs))
	for _, d := range dirs {
		full := filepath.Join(root, d)
		if _, err := os.Stat(full); err == nil {
			out = append(out, StepResult{What: d, Action: ActionPreserved})
			continue
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("statting %s: %w", d, err)
		}
		if err := os.MkdirAll(full, 0o755); err != nil {
			return nil, fmt.Errorf("creating %s: %w", d, err)
		}
		out = append(out, StepResult{What: d, Action: ActionCreated})
	}
	return out, nil
}

func ensureGitignore(root string) (StepResult, error) {
	paths, err := skills.MaterializedPaths()
	if err != nil {
		return StepResult{}, err
	}

	path := filepath.Join(root, ".gitignore")
	existing, readErr := os.ReadFile(path)
	if readErr != nil && !errors.Is(readErr, fs.ErrNotExist) {
		return StepResult{}, fmt.Errorf("reading .gitignore: %w", readErr)
	}

	have := make(map[string]bool)
	for _, line := range strings.Split(string(existing), "\n") {
		have[strings.TrimSpace(line)] = true
	}

	var missing []string
	for _, p := range paths {
		if !have[p] {
			missing = append(missing, p)
		}
	}
	if len(missing) == 0 {
		return StepResult{What: ".gitignore", Action: ActionPreserved}, nil
	}
	sort.Strings(missing)

	var b strings.Builder
	if len(existing) > 0 {
		b.Write(existing)
		if !strings.HasSuffix(string(existing), "\n") {
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("# aiwf: materialized skill adapters (regenerated by aiwf update)\n")
	for _, p := range missing {
		b.WriteString(p)
		b.WriteString("\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return StepResult{}, fmt.Errorf("writing .gitignore: %w", err)
	}
	action := ActionUpdated
	if readErr != nil {
		action = ActionCreated
	}
	return StepResult{
		What:   ".gitignore",
		Action: action,
		Detail: fmt.Sprintf("appended %d skill path(s)", len(missing)),
	}, nil
}

func ensureClaudeMd(root string) (StepResult, error) {
	path := filepath.Join(root, "CLAUDE.md")
	if _, err := os.Stat(path); err == nil {
		return StepResult{What: "CLAUDE.md", Action: ActionPreserved}, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return StepResult{}, fmt.Errorf("statting CLAUDE.md: %w", err)
	}
	if err := os.WriteFile(path, []byte(CLAUDETemplate), 0o644); err != nil {
		return StepResult{}, fmt.Errorf("writing CLAUDE.md: %w", err)
	}
	return StepResult{What: "CLAUDE.md", Action: ActionCreated}, nil
}

func ensurePreHook(ctx context.Context, root string) (StepResult, error) {
	gitDir, err := gitops.GitDir(ctx, root)
	if err != nil {
		return StepResult{}, fmt.Errorf("locating git dir: %w", err)
	}
	hooksDir := filepath.Join(gitDir, "hooks")
	if mkErr := os.MkdirAll(hooksDir, 0o755); mkErr != nil {
		return StepResult{}, fmt.Errorf("creating hooks dir: %w", mkErr)
	}
	hookPath := filepath.Join(hooksDir, "pre-push")

	existing, readErr := os.ReadFile(hookPath)
	switch {
	case errors.Is(readErr, fs.ErrNotExist):
		// no existing hook: create
	case readErr != nil:
		return StepResult{}, fmt.Errorf("reading pre-push hook: %w", readErr)
	case strings.Contains(string(existing), preHookMarker):
		// our own hook: overwrite is safe
	default:
		return StepResult{}, fmt.Errorf("%w: leave it in place and call `aiwf check` from inside it, or use a hook manager (husky/lefthook)", ErrPreHookConflict)
	}

	exePath, err := resolveExecutable()
	if err != nil {
		return StepResult{}, fmt.Errorf("resolving aiwf binary path: %w", err)
	}
	if err := os.WriteFile(hookPath, []byte(preHookScript(exePath)), 0o755); err != nil {
		return StepResult{}, fmt.Errorf("writing pre-push hook: %w", err)
	}
	action := ActionCreated
	if !errors.Is(readErr, fs.ErrNotExist) {
		action = ActionUpdated
	}
	return StepResult{
		What:   ".git/hooks/pre-push",
		Action: action,
		Detail: "exec " + exePath,
	}, nil
}

// resolveExecutable returns the absolute, symlink-resolved path of
// the running binary. The hook bakes this in so push-time hook
// execution never depends on `$PATH`.
func resolveExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		// Symlink resolution can fail on some platforms or for
		// unusual paths; fall back to the unresolved exe rather than
		// failing init outright. Re-running init after a `mv` will
		// fix it.
		return exe, nil
	}
	return resolved, nil
}

// HookMarker exposes the marker line for tests that assert the hook
// was installed by aiwf vs. someone else.
func HookMarker() string { return preHookMarker }
