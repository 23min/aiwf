// Package initrepo implements `aiwf init`: idempotent first-time
// setup for a consumer repo. See docs/pocv3/plans/poc-plan.md Session 3 for the
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

// preCommitHookMarker is the sibling marker for the pre-commit hook
// that regenerates `STATUS.md`. Same protective contract as
// preHookMarker: re-running init/update overwrites only when the
// existing hook carries this marker; an alien pre-commit hook is
// left untouched.
const preCommitHookMarker = "# aiwf:pre-commit"

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

Skills under ` + "`.claude/skills/aiwf-*/`" + ` are gitignored and regenerated on ` + "`aiwf update`" + `.
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

// preCommitHookScript renders the body of the pre-commit hook that
// regenerates `STATUS.md` on every commit. The aiwf binary's absolute
// path is baked in (same rationale as preHookScript: hooks should
// not depend on the user's interactive `$PATH`). The script is
// tolerant by design — any failure path silently exits 0, so a
// missing/moved/broken binary, a transient `aiwf status` error, or
// a tree the engine refuses to read does not block commits. Drift
// between the installed body and this template is detected by
// `aiwf doctor` and remediated by `aiwf update`.
func preCommitHookScript(execPath string) string {
	return `#!/bin/sh
` + preCommitHookMarker + `
# Installed by aiwf init/update. Regenerates STATUS.md so the
# committed snapshot stays in sync with the entity tree. Tolerant —
# any failure silently no-ops so contributors are never blocked.
# Opt out: set status_md.auto_update: false in aiwf.yaml and run
# 'aiwf update' to remove this hook.
set -e
repo_root="$(git rev-parse --show-toplevel)"
tmp="$repo_root/STATUS.md.tmp"
if ` + shellQuoteSingle(execPath) + ` status --root "$repo_root" --format=md >"$tmp" 2>/dev/null; then
    mv "$tmp" "$repo_root/STATUS.md"
    git add "$repo_root/STATUS.md"
else
    rm -f "$tmp"
fi
exit 0
`
}

// Action classifies what init did for a single step. The CLI uses this
// to render a friendly summary.
type Action string

// Action values reported per step.
const (
	ActionCreated   Action = "created"
	ActionPreserved Action = "preserved"
	ActionUpdated   Action = "updated"
	// ActionSkipped marks a step that init declined to perform because
	// doing so would clobber user-managed state. The Detail field on
	// the StepResult explains why and what the user should do next.
	ActionSkipped Action = "skipped"
	// ActionRemoved marks a step that uninstalled a previously-managed
	// artifact because the consumer opted out. Currently used only by
	// the pre-commit hook step when status_md.auto_update flips false.
	ActionRemoved Action = "removed"
)

// StepResult is one line of init's per-step ledger.
type StepResult struct {
	What   string
	Action Action
	Detail string
}

// Result is the per-step ledger init returns. Order matches the order
// of operations. HookConflict is set when init declined to install
// the pre-push hook because a non-aiwf hook was already in place;
// callers should surface remediation guidance to the user. DryRun
// echoes Options.DryRun so callers can format output appropriately
// (a dry-run ledger looks identical but no writes occurred).
type Result struct {
	Steps        []StepResult
	HookConflict bool
	DryRun       bool
}

// Options carries init-time inputs that override or supplement the
// defaults. ActorOverride bypasses git-config derivation when set.
// AiwfVersion stamps aiwf.yaml's `aiwf_version`; the CLI passes the
// binary's Version constant.
//
// DryRun computes the would-be ledger without performing any
// filesystem mutations. SkipHook omits pre-push hook installation
// entirely (still reported in the ledger as a skipped step).
type Options struct {
	ActorOverride string
	AiwfVersion   string
	DryRun        bool
	SkipHook      bool
}

// Init runs the documented setup steps in order. Returns a Result that
// describes what was created vs preserved vs updated. Errors abort
// early — a partially-applied init is rare in practice (init only
// touches config / scaffolding / skills) and the user can re-run.
func Init(ctx context.Context, root string, opts Options) (*Result, error) {
	if opts.AiwfVersion == "" {
		return nil, errors.New("AiwfVersion is required")
	}

	res := &Result{DryRun: opts.DryRun}

	// 1. aiwf.yaml — write only if missing.
	cfgStep, err := ensureConfig(root, opts)
	if err != nil {
		return nil, err
	}
	res.Steps = append(res.Steps, cfgStep)

	// 2. Scaffold entity directories.
	scaffoldSteps, err := scaffoldDirs(root, opts.DryRun)
	if err != nil {
		return nil, err
	}
	res.Steps = append(res.Steps, scaffoldSteps...)

	// 3. Materialize skills (wipe-and-rewrite per cache contract).
	skillsStep, err := ensureSkills(root, opts.DryRun)
	if err != nil {
		return nil, err
	}
	res.Steps = append(res.Steps, skillsStep)

	// 4. Append skill paths to .gitignore.
	gitignoreStep, err := ensureGitignore(root, opts.DryRun)
	if err != nil {
		return nil, err
	}
	res.Steps = append(res.Steps, gitignoreStep)

	// 5. CLAUDE.md template — write only if missing.
	claudeStep, err := ensureClaudeMd(root, opts.DryRun)
	if err != nil {
		return nil, err
	}
	res.Steps = append(res.Steps, claudeStep)

	// 6. Pre-push hook — install, overwrite-if-marker-present, or
	// skip-with-remediation when a non-aiwf hook is already in place.
	// SkipHook bypasses the install entirely (reported as skipped); a
	// skipped hook is never a fatal error.
	if opts.SkipHook {
		res.Steps = append(res.Steps, StepResult{
			What:   ".git/hooks/pre-push",
			Action: ActionSkipped,
			Detail: "--skip-hook flag set",
		})
	} else {
		hookStep, conflict, hErr := ensurePreHook(ctx, root, opts.DryRun)
		if hErr != nil {
			return nil, hErr
		}
		res.Steps = append(res.Steps, hookStep)
		res.HookConflict = conflict
	}

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

	if opts.DryRun {
		return StepResult{
			What:   config.FileName,
			Action: ActionCreated,
			Detail: "actor=" + actor,
		}, nil
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

func scaffoldDirs(root string, dryRun bool) ([]StepResult, error) {
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
		if !dryRun {
			if err := os.MkdirAll(full, 0o755); err != nil {
				return nil, fmt.Errorf("creating %s: %w", d, err)
			}
		}
		out = append(out, StepResult{What: d, Action: ActionCreated})
	}
	return out, nil
}

// ensureSkills materializes skill files (wipe-and-rewrite per cache
// contract). In dry-run mode, returns the would-be ledger entry
// without touching disk.
func ensureSkills(root string, dryRun bool) (StepResult, error) {
	if dryRun {
		embedded, err := skills.List()
		if err != nil {
			return StepResult{}, err
		}
		return StepResult{
			What:   ".claude/skills/aiwf-*",
			Action: ActionUpdated,
			Detail: fmt.Sprintf("would materialize %d skills from embedded", len(embedded)),
		}, nil
	}
	if err := skills.Materialize(root); err != nil {
		return StepResult{}, fmt.Errorf("materializing skills: %w", err)
	}
	return StepResult{
		What:   ".claude/skills/aiwf-*",
		Action: ActionUpdated,
		Detail: "materialized from embedded skills",
	}, nil
}

func ensureGitignore(root string, dryRun bool) (StepResult, error) {
	paths := skills.GitignorePatterns()

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
	action := ActionUpdated
	if readErr != nil {
		action = ActionCreated
	}
	if !dryRun {
		if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
			return StepResult{}, fmt.Errorf("writing .gitignore: %w", err)
		}
	}
	return StepResult{
		What:   ".gitignore",
		Action: action,
		Detail: fmt.Sprintf("appended %d skill path(s)", len(missing)),
	}, nil
}

func ensureClaudeMd(root string, dryRun bool) (StepResult, error) {
	path := filepath.Join(root, "CLAUDE.md")
	if _, err := os.Stat(path); err == nil {
		return StepResult{What: "CLAUDE.md", Action: ActionPreserved}, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return StepResult{}, fmt.Errorf("statting CLAUDE.md: %w", err)
	}
	if !dryRun {
		if err := os.WriteFile(path, []byte(CLAUDETemplate), 0o644); err != nil {
			return StepResult{}, fmt.Errorf("writing CLAUDE.md: %w", err)
		}
	}
	return StepResult{What: "CLAUDE.md", Action: ActionCreated}, nil
}

// ensurePreHook installs (or refreshes) the marker-protected pre-push
// hook. The bool return is "skipped due to conflict": when a hook
// without aiwf's marker already exists, ensurePreHook returns a
// skipped StepResult and `true`, leaving the user's hook untouched.
// Skipping is not a fatal error — the caller surfaces remediation.
//
// In dry-run mode, conflict detection still runs (read-only) but no
// directory or file is created.
func ensurePreHook(ctx context.Context, root string, dryRun bool) (StepResult, bool, error) {
	gitDir, err := gitops.GitDir(ctx, root)
	if err != nil {
		return StepResult{}, false, fmt.Errorf("locating git dir: %w", err)
	}
	hooksDir := filepath.Join(gitDir, "hooks")
	if !dryRun {
		if mkErr := os.MkdirAll(hooksDir, 0o755); mkErr != nil {
			return StepResult{}, false, fmt.Errorf("creating hooks dir: %w", mkErr)
		}
	}
	hookPath := filepath.Join(hooksDir, "pre-push")

	existing, readErr := os.ReadFile(hookPath)
	switch {
	case errors.Is(readErr, fs.ErrNotExist):
		// no existing hook: create
	case readErr != nil:
		return StepResult{}, false, fmt.Errorf("reading pre-push hook: %w", readErr)
	case strings.Contains(string(existing), preHookMarker):
		// our own hook: overwrite is safe
	default:
		// non-aiwf hook in place: skip without clobbering.
		return StepResult{
			What:   ".git/hooks/pre-push",
			Action: ActionSkipped,
			Detail: "existing hook has no aiwf marker — left untouched (see remediation below)",
		}, true, nil
	}

	exePath, err := resolveExecutable()
	if err != nil {
		return StepResult{}, false, fmt.Errorf("resolving aiwf binary path: %w", err)
	}
	action := ActionCreated
	if !errors.Is(readErr, fs.ErrNotExist) {
		action = ActionUpdated
	}
	if !dryRun {
		if err := os.WriteFile(hookPath, []byte(preHookScript(exePath)), 0o755); err != nil {
			return StepResult{}, false, fmt.Errorf("writing pre-push hook: %w", err)
		}
	}
	return StepResult{
		What:   ".git/hooks/pre-push",
		Action: action,
		Detail: "exec " + exePath,
	}, false, nil
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

// PreCommitHookMarker exposes the pre-commit hook's marker line for
// tests and for `aiwf doctor` to identify a marker-managed hook
// versus a user-written one.
func PreCommitHookMarker() string { return preCommitHookMarker }

// ensurePreCommitHook installs (or refreshes, or removes) the
// marker-protected pre-commit hook that regenerates `STATUS.md`. The
// install argument carries the consumer's opt-in state from
// `aiwf.yaml.status_md.auto_update`:
//
//   - install=true: write or refresh the hook (Created or Updated).
//     If a non-marker hook is already in place, return ActionSkipped
//     and conflict=true so the caller can surface remediation, same
//     contract as ensurePreHook.
//
//   - install=false: remove a previously-installed marker-managed
//     hook (ActionRemoved). A non-marker hook is left alone (the
//     consumer's content is theirs to manage). Absence reports
//     ActionPreserved with a "disabled by config" detail so the
//     ledger explains why the step did nothing.
//
// In dry-run mode no filesystem mutation occurs; the StepResult still
// reflects what would have happened so the user can preview.
func ensurePreCommitHook(ctx context.Context, root string, install, dryRun bool) (StepResult, bool, error) {
	gitDir, err := gitops.GitDir(ctx, root)
	if err != nil {
		return StepResult{}, false, fmt.Errorf("locating git dir: %w", err)
	}
	hooksDir := filepath.Join(gitDir, "hooks")
	hookPath := filepath.Join(hooksDir, "pre-commit")
	what := ".git/hooks/pre-commit"

	existing, readErr := os.ReadFile(hookPath)
	hasOurMarker := readErr == nil && strings.Contains(string(existing), preCommitHookMarker)
	hasAlienHook := readErr == nil && !hasOurMarker

	if !install {
		switch {
		case hasAlienHook:
			return StepResult{
				What:   what,
				Action: ActionSkipped,
				Detail: "existing hook has no aiwf marker — left untouched",
			}, true, nil
		case hasOurMarker:
			if !dryRun {
				if rmErr := os.Remove(hookPath); rmErr != nil {
					return StepResult{}, false, fmt.Errorf("removing pre-commit hook: %w", rmErr)
				}
			}
			return StepResult{
				What:   what,
				Action: ActionRemoved,
				Detail: "status_md.auto_update: false",
			}, false, nil
		default:
			return StepResult{
				What:   what,
				Action: ActionPreserved,
				Detail: "disabled by config (status_md.auto_update: false)",
			}, false, nil
		}
	}

	if hasAlienHook {
		return StepResult{
			What:   what,
			Action: ActionSkipped,
			Detail: "existing hook has no aiwf marker — left untouched (see remediation below)",
		}, true, nil
	}

	if !dryRun {
		if mkErr := os.MkdirAll(hooksDir, 0o755); mkErr != nil {
			return StepResult{}, false, fmt.Errorf("creating hooks dir: %w", mkErr)
		}
	}

	exePath, err := resolveExecutable()
	if err != nil {
		return StepResult{}, false, fmt.Errorf("resolving aiwf binary path: %w", err)
	}

	action := ActionCreated
	if hasOurMarker {
		action = ActionUpdated
	}
	if !dryRun {
		if err := os.WriteFile(hookPath, []byte(preCommitHookScript(exePath)), 0o755); err != nil {
			return StepResult{}, false, fmt.Errorf("writing pre-commit hook: %w", err)
		}
	}
	return StepResult{
		What:   what,
		Action: action,
		Detail: "exec " + exePath + " status --format=md",
	}, false, nil
}
