// Package doctor implements the `aiwf doctor` verb. The bare verb
// runs a local health check (config presence, identity resolution,
// materialized-skill drift, id collisions, validator availability,
// hook drift, recommended-plugin presence). With --self-check it
// drives every aiwf verb against a throwaway repo to prove the
// binary works end-to-end; the --self-check entry point uses the
// in-process Dispatcher seam wired by cmd/aiwf/main.go.
package doctor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/skills"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/version"
)

// NewCmd builds `aiwf doctor`: version check, materialized-skill
// drift check, id-collision check. With --self-check, instead drives
// every mutating verb against a throwaway repo to prove the binary
// works end-to-end.
func NewCmd() *cobra.Command {
	var (
		root         string
		selfCheck    bool
		checkLatest  bool
		writeHealth  bool
		checkRituals bool
	)
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Drift / version / id-collision health check",
		Example: `  # Local health check on the current consumer repo
  aiwf doctor

  # Drive every verb against a throwaway repo (CI smoke test)
  aiwf doctor --self-check

  # Compare to the latest published release on the Go module proxy
  aiwf doctor --check-latest

  # Terse, exit-code-meaningful ritual-materialization check (for
  # automation, e.g. the worktree-materialization SessionStart hook)
  aiwf doctor --check-rituals --root <worktree-path>`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			if checkRituals {
				return cliutil.WrapExitCode(RunCheckRituals(root))
			}
			if writeHealth {
				return cliutil.WrapExitCode(runWriteHealth(root))
			}
			return cliutil.WrapExitCode(Run(root, selfCheck, checkLatest))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().BoolVar(&selfCheck, "self-check", false, "run every verb against a temp repo and report pass/fail")
	cmd.Flags().BoolVar(&checkLatest, "check-latest", false, "look up the latest published aiwf version on the Go module proxy (one HTTP call; honors GOPROXY=off)")
	cmd.Flags().BoolVar(&writeHealth, "write-health", false, "write .claude/health.aiwf.json from doctor's warnings and errors (consumed by the statusline health stoplight)")
	cmd.Flags().BoolVar(&checkRituals, "check-rituals", false, "check only whether ritual artifacts (skills/agents/templates) are materialized; silent with exit 0 if so, or a single actionable stderr line with exit 1 otherwise")
	return cmd
}

// Run is the exported entry point for `aiwf doctor`. Dispatches to
// the report-only path or --self-check based on the flag.
func Run(root string, selfCheck, checkLatest bool) int {
	if selfCheck {
		return runSelfCheck()
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil { //coverage:ignore ResolveRoot(--root) resolves via filepath.Abs and cannot fail here; defensive parity with runWriteHealth
		cliutil.Errorf("aiwf doctor: %v\n", err)
		return cliutil.ExitUsage
	}

	report, problems := DoctorReport(rootDir, DoctorOptions{CheckLatest: checkLatest})
	for _, line := range report {
		cliutil.Println(line)
	}
	// Exit findings iff at least one problem is error-severity;
	// warnings alone keep exit 0 (doctor exits 0 on advisories, same
	// as `aiwf check`).
	for i := range problems {
		if problems[i].Severity == SeverityError {
			return cliutil.ExitFindings
		}
	}
	return cliutil.ExitOK
}

// runWriteHealth writes .claude/health.aiwf.json for the statusline
// health stoplight. It stamps generated_at at the CLI edge (keeping
// WriteHealth wall-clock-free) and stays quiet on success so callers
// like `aiwf update` can invoke it without report noise.
func runWriteHealth(root string) int {
	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil { //coverage:ignore ResolveRoot(--root) resolves via filepath.Abs and cannot fail here; defensive parity with Run
		cliutil.Errorf("aiwf doctor: %v\n", err)
		return cliutil.ExitUsage
	}
	if err := WriteHealth(context.Background(), rootDir, time.Now().UTC().Format(time.RFC3339), DoctorOptions{}); err != nil {
		cliutil.Errorf("aiwf doctor --write-health: %v\n", err)
		return cliutil.ExitInternal
	}
	return cliutil.ExitOK
}

// DoctorOptions carries flag-derived knobs into DoctorReport. Kept
// separate so DoctorReport stays flag-package-free and unit-testable.
type DoctorOptions struct {
	// CheckLatest, when true, performs a Go module proxy lookup for
	// the latest published aiwf version and adds a `latest:` row to
	// the report. Default false (offline).
	CheckLatest bool
}

// labelWidth is the column where doctor values start. The widest
// label today is `plugin-mount:` (13 chars); +1 trailing space puts
// the value column at 14. label() pads to this width so every line
// aligns.
const labelWidth = 14

// label pads a doctor-output prefix to labelWidth columns so values
// line up. Labels at or beyond labelWidth get exactly one trailing
// space (graceful degradation; the column shifts on that one line
// only). The sub-line continuation indent is the same width as a
// blank label — just spell `subIndent` instead of remembering the
// number.
func label(s string) string {
	if len(s) >= labelWidth {
		return s + " "
	}
	return s + strings.Repeat(" ", labelWidth-len(s))
}

// subIndent is the continuation indent for multi-line doctor entries
// (e.g., `note:` lines under config:, or per-skill bullets under
// skills:). Exactly labelWidth spaces so the continuation aligns
// with the value column.
var subIndent = strings.Repeat(" ", labelWidth)

// DoctorReport collects every doctor finding into a slice of human
// strings and returns the warnings and errors as []Problem — the
// report's problem states without the ok/info context. Pure for
// testability. The error count (what doctor's exit code weighs) is the
// number of SeverityError entries; warnings are advisory.
func DoctorReport(rootDir string, opts DoctorOptions) (lines []string, problems []Problem) {
	current := version.Current()
	binaryRow := label("binary:") + renderBinaryVersion(current)
	binaryRow += binaryStaleness(context.Background(), rootDir, current, version.ModulePath())
	lines = append(lines, binaryRow)

	if opts.CheckLatest {
		lines = append(lines, label("latest:")+renderLatestPublished(current))
	}

	// env: line — informational, never increments problems. M-0135/AC-1.
	inContainer, envLabel := InContainer()
	lines = append(lines, label("env:")+envLabel)

	// plugin-mount: + plugin-paths: lines — gated on in-container,
	// never increment problems. M-0135/AC-2; plugin-paths: closes
	// G-0174.
	if inContainer {
		if home, homeErr := os.UserHomeDir(); homeErr != nil {
			lines = append(lines, renderMountLine(mountStateError, 0, homeErr.Error()))
		} else {
			state, count, mountErr := shadowMountStatus(home)
			if mountErr != nil {
				lines = append(lines, renderMountLine(mountStateError, 0, mountErr.Error()))
			} else {
				lines = append(lines, renderMountLine(state, count, ""))
			}
			// plugin-paths: advisory hint — the plugin-mount probe
			// above checks the target's presence, not the OS-correctness
			// of the paths inside the index. This catches the
			// claude-code#31388 leak the presence check reports as `ok`.
			if sample, found := foreignPluginPaths(home, foreignHomePrefix()); found {
				lines = append(lines, renderPluginPathHintLine(sample))
			}
		}
	}

	cfg, err := config.Load(rootDir)
	switch {
	case errors.Is(err, config.ErrNotFound):
		val := "aiwf.yaml not found (run `aiwf init`)"
		lines = append(lines, label("config:")+val)
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
	case err != nil:
		lines = append(lines, label("config:")+err.Error())
		problems = append(problems, Problem{Severity: SeverityError, Message: err.Error()})
	default:
		lines = append(lines, label("config:")+"ok")
		if cfg.LegacyActor != "" {
			lines = append(lines,
				fmt.Sprintf("%snote: aiwf.yaml carries a deprecated `actor: %s` key — identity is now runtime-derived from git config user.email; the field is ignored. Run `aiwf update` to remove.", subIndent, cfg.LegacyActor))
		}
		if cfg.LegacyAiwfVersion != "" {
			lines = append(lines,
				fmt.Sprintf("%snote: aiwf.yaml carries a deprecated `aiwf_version: %s` key — version state is now derived from the binary (`aiwf version`); the field is ignored. Run `aiwf update` to remove.", subIndent, cfg.LegacyAiwfVersion))
		}
	}

	// worktree-dir: line (M-0189) — the resolved ritual-worktree
	// placement directory the start rituals (M-0190) read to honor
	// aiwf.yaml worktree.dir. Informational; never increments problems.
	// Annotated (configured) when the consumer's knob is honored,
	// (default) when falling back to the kernel default. cfg may be nil
	// here (config load failed above); WorktreeDir() is nil-tolerant.
	worktreeDir := cfg.WorktreeDir()
	worktreeAnnot := "default"
	if worktreeDir != config.DefaultWorktreeDir {
		worktreeAnnot = "configured"
	}
	lines = append(lines, fmt.Sprintf("%s%s (%s)", label("worktree-dir:"), worktreeDir, worktreeAnnot))

	lines, problems = appendLoggingReport(lines, problems, cfg)

	if actor, source, actorErr := cliutil.ResolveActorWithSource("", rootDir); actorErr != nil {
		lines = append(lines, label("actor:")+actorErr.Error())
		problems = append(problems, Problem{Severity: SeverityError, Message: actorErr.Error()})
	} else {
		lines = append(lines, fmt.Sprintf("%s%s (from %s)", label("actor:"), actor, source))
	}

	// M-0161/AC-7 / G-0207: detached-HEAD advisory. Detached HEAD
	// is a state operators reach via `git checkout <sha>`,
	// `git worktree add --detach`, or during a `git rebase`. The
	// `aiwf authorize` preflight refuses on detached HEAD (per
	// AC-7's preflight refinement); surfacing the state at doctor
	// time lets operators discover it proactively rather than via
	// a verb refusal. Advisory severity — does NOT increment
	// problems; lines starting with `head: detached-head ...` are
	// the canonical substring marker for AC-7 E2E discrimination
	// (per AC-7 body line 498's documented substring-against-stderr
	// exception, paralleled here as substring-against-stdout for
	// the doctor surface).
	if currentBranch(rootDir) == "" && headIsDetached(rootDir) {
		val := "detached-head: advisory — no symbolic HEAD; checkout a ritual branch (epic/E-NNNN-<slug> / milestone/M-NNNN-<slug>) before running `aiwf authorize`, or use `--force --reason \"...\"` to override the preflight refusal."
		lines = append(lines, label("head:")+val)
		problems = append(problems, Problem{Severity: SeverityWarn, Message: val})
	}

	embedded, err := skills.List()
	if err != nil { //coverage:ignore skills.List reads the compiled-in embed FS; it cannot fail at runtime, so tempdir tests cannot reach this arm
		lines = append(lines, label("skills:")+err.Error())
		problems = append(problems, Problem{Severity: SeverityError, Message: err.Error()})
	} else {
		drift, missing := skillDrift(rootDir, embedded)
		switch {
		case len(missing) > 0:
			val := fmt.Sprintf("%d missing — run `aiwf init` or `aiwf update`", len(missing))
			lines = append(lines, label("skills:")+val)
			for _, m := range missing {
				lines = append(lines, subIndent+"- "+m)
			}
			problems = append(problems, Problem{Severity: SeverityError, Message: val})
		case len(drift) > 0:
			val := fmt.Sprintf("%d drifted — run `aiwf update` to refresh", len(drift))
			lines = append(lines, label("skills:")+val)
			for _, d := range drift {
				lines = append(lines, subIndent+"- "+d)
			}
			problems = append(problems, Problem{Severity: SeverityError, Message: val})
		default:
			lines = append(lines, fmt.Sprintf("%sok (%d skills, byte-equal to embed)", label("skills:"), len(embedded)))
		}
	}

	tr, loadErrs, err := tree.Load(context.Background(), rootDir)
	if err != nil {
		lines = append(lines, label("ids:")+err.Error())
		problems = append(problems, Problem{Severity: SeverityError, Message: err.Error()})
	} else {
		findings := check.Run(tr, loadErrs)
		collisions := 0
		for i := range findings {
			f := &findings[i]
			if f.Code == check.CodeIDsUnique {
				collisions++
				val := fmt.Sprintf("collision %s @ %s", f.EntityID, f.Path)
				lines = append(lines, label("ids:")+val)
				problems = append(problems, Problem{Severity: SeverityError, Message: val})
			}
		}
		if collisions == 0 {
			lines = append(lines, label("ids:")+"ok (no collisions)")
		}
	}

	lines, problems = appendValidatorReport(lines, problems, rootDir)

	lines = append(lines, fmt.Sprintf("%s%s (%s)", label("filesystem:"), filesystemCaseLabel(rootDir), rootDir))

	lines, problems = appendHookReport(lines, problems, rootDir)
	lines, problems = appendPreCommitHookReport(lines, problems, rootDir)
	lines, problems = appendCommitMsgHookReport(lines, problems, rootDir)
	lines, problems = appendPostCommitHookReport(lines, problems, rootDir)
	lines, problems = appendRenderReport(lines, problems, rootDir)
	lines, problems = appendMaterializedRitualsReport(lines, problems, rootDir)
	lines, problems = appendHookMaterializationReport(lines, problems, rootDir, skills.ShippedHooks)
	lines, problems = appendStatuslineReport(lines, problems, rootDir)
	lines, problems = appendGuidanceImportReport(lines, problems, rootDir)

	return lines, problems
}

// appendMaterializedRitualsReport verifies the embedded ritual
// artifacts (skills, agents, templates) are materialized under the
// consumer's `.claude/` tree (ADR-0014 §5 — doctor verifies the
// materialized artifacts instead of recommending a marketplace plugin).
// A `rituals:` ok line confirms presence; a soft warning naming the
// missing artifacts points at `aiwf update`. Rituals are advisory
// artifacts, so a miss never increments the problem count.
func appendMaterializedRitualsReport(in []string, problemsIn []Problem, rootDir string) (lines []string, problems []Problem) {
	problems = problemsIn
	present, missing, err := skills.MaterializedRituals(rootDir, skills.ClaudeTarget)
	if err != nil { //coverage:ignore MaterializedRituals errors only when the compiled-in embed FS walk fails; unreachable at runtime, so tempdir tests cannot reach this arm
		return append(in, label("rituals:")+err.Error()), problems
	}
	if len(missing) > 0 {
		out := in
		val := fmt.Sprintf("%d of %d ritual artifacts not materialized — run `aiwf update`", len(missing), len(present)+len(missing))
		out = append(out, label("rituals:")+val)
		for _, m := range missing {
			out = append(out, subIndent+"- "+m)
		}
		problems = append(problems, Problem{Severity: SeverityWarn, Message: val})
		return out, problems
	}
	return append(in,
		fmt.Sprintf("%sok (%d artifacts materialized)", label("rituals:"), len(present)),
		subIndent+"managed by aiwf (skills aiwf-*/aiwfx-*/wf-*, agents, templates); `aiwf update` refreshes — do not hand-edit (see .claude/skills/README.md)",
	), problems
}

// appendHookMaterializationReport surfaces ADR-0032's three
// doctor-visible hook-registry drift classes: a shipped hook still
// undecided in the consumer's aiwf.yaml, one materialized but not
// wired into the shared settings file, and one wired despite its
// decision no longer authorizing it (or its script gone missing) —
// "wired but stale". hooks is passed explicitly (rather than always
// reading skills.ShippedHooks) so tests can exercise the reporting
// logic against a synthetic registry ahead of any concrete hook
// landing (M-0236). An empty registry — today's state — reports a
// quiet ok line rather than silently omitting the row.
func appendHookMaterializationReport(in []string, problemsIn []Problem, rootDir string, hooks []skills.HookDef) (lines []string, problems []Problem) {
	lines = in
	problems = problemsIn
	if len(hooks) == 0 {
		return append(lines, label("hooks:")+"ok (no hooks registered yet)"), problems
	}

	configPath := filepath.Join(rootDir, config.FileName)
	doc, _, readErr := aiwfyaml.Read(configPath)
	var decisions map[string]bool
	switch {
	case readErr == nil:
		var hooksErr error
		decisions, hooksErr = doc.Hooks()
		if hooksErr != nil {
			val := hooksErr.Error()
			lines = append(lines, label("hooks:")+val)
			problems = append(problems, Problem{Severity: SeverityError, Message: val})
			return lines, problems
		}
	case errors.Is(readErr, os.ErrNotExist):
		// No aiwf.yaml yet — every registry hook is undecided, not an
		// error (mirrors appendRenderReport's config.Load-missing
		// default-and-continue handling).
		decisions = map[string]bool{}
	default:
		val := readErr.Error()
		lines = append(lines, label("hooks:")+val)
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
		return lines, problems
	}

	settingsPath := filepath.Join(rootDir, skills.SharedSettingsRelPath)
	report, err := skills.HookDrift(rootDir, skills.ClaudeTarget, hooks, decisions, settingsPath)
	if err != nil {
		val := err.Error()
		lines = append(lines, label("hooks:")+val)
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
		return lines, problems
	}

	if len(report.Undecided) == 0 && len(report.MaterializedNotWired) == 0 && len(report.WiredButStale) == 0 {
		return append(lines, fmt.Sprintf("%sok (%d hooks synced)", label("hooks:"), len(hooks))), problems
	}

	val := fmt.Sprintf("drift: %d undecided, %d materialized-not-wired, %d wired-but-stale — run `aiwf update` to reconcile",
		len(report.Undecided), len(report.MaterializedNotWired), len(report.WiredButStale))
	lines = append(lines, label("hooks:")+val)
	for _, name := range report.Undecided {
		lines = append(lines, subIndent+"- undecided: "+name)
	}
	for _, name := range report.MaterializedNotWired {
		lines = append(lines, subIndent+"- materialized-not-wired: "+name)
	}
	for _, name := range report.WiredButStale {
		lines = append(lines, subIndent+"- wired-but-stale: "+name)
	}
	problems = append(problems, Problem{Severity: SeverityWarn, Message: val})
	return lines, problems
}

// appendRenderReport surfaces the consumer's HTML render
// configuration plus a check for the false→true commit_output
// misconfiguration: when the consumer flips commit_output to true
// without re-running `aiwf update`, the gitignore still carries the
// stale `<out_dir>/` line and the rendered files are invisible to
// git.
func appendRenderReport(in []string, problemsIn []Problem, rootDir string) (lines []string, problems []Problem) {
	lines = in
	problems = problemsIn

	cfg, err := config.Load(rootDir)
	outDir := config.DefaultHTMLOutDir
	commitOutput := false
	if err == nil && cfg != nil {
		outDir = cfg.HTMLOutDir()
		commitOutput = cfg.HTML.CommitOutput
	}
	commitLabel := "false (output gitignored)"
	if commitOutput {
		commitLabel = "true (output committed)"
	}
	lines = append(lines, fmt.Sprintf("%sout_dir=%s commit_output=%s", label("render:"), outDir, commitLabel))

	if commitOutput {
		gitignorePath := filepath.Join(rootDir, ".gitignore")
		if raw, readErr := os.ReadFile(gitignorePath); readErr == nil {
			needle := strings.TrimRight(outDir, "/") + "/"
			for _, line := range strings.Split(string(raw), "\n") {
				if strings.TrimSpace(line) == needle {
					val := fmt.Sprintf("drift: commit_output is true but .gitignore still holds %q; run `aiwf update` to reconcile", needle)
					lines = append(lines, subIndent+val)
					problems = append(problems, Problem{Severity: SeverityError, Message: val})
					break
				}
			}
		}
	}
	return lines, problems
}

// appendHookReport inspects the pre-push hook at the consumer's
// effective hooks directory and reports its state.
func appendHookReport(in []string, problemsIn []Problem, rootDir string) (lines []string, problems []Problem) {
	lines = in
	problems = problemsIn

	hooksDir := resolveHooksDir(rootDir)
	hookPath := filepath.Join(hooksDir, "pre-push")
	raw, err := os.ReadFile(hookPath)
	if errors.Is(err, os.ErrNotExist) {
		val := "missing — pre-push validation not installed; run `aiwf init` to install"
		lines = append(lines, label("hook:")+val)
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
		return lines, problems
	}
	if err != nil {
		lines = append(lines, label("hook:")+err.Error())
		problems = append(problems, Problem{Severity: SeverityError, Message: err.Error()})
		return lines, problems
	}
	if !strings.Contains(string(raw), initrepo.HookMarker()) {
		val := "present but not aiwf-managed (no `# aiwf:pre-push` marker); aiwf check is not running pre-push"
		lines = append(lines, label("hook:")+val)
		problems = append(problems, Problem{Severity: SeverityWarn, Message: val})
		return lines, problems
	}

	// Post-G-0135 / M-0133 / AC-1: hooks resolve aiwf via PATH lookup
	// at hook-fire time. Validate the binary is reachable on PATH via
	// exec.LookPath rather than stat'ing a baked path.
	if strings.Contains(string(raw), "command -v aiwf") {
		found, lookErr := exec.LookPath("aiwf")
		if lookErr != nil {
			val := "aiwf binary not found on PATH (hook would fail at push time); install via `go install ./cmd/aiwf` and ensure $GOPATH/bin is on PATH"
			lines = append(lines, label("hook:")+val)
			problems = append(problems, Problem{Severity: SeverityError, Message: val})
			return lines, problems
		}
		chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "pre-push")
		val := fmt.Sprintf("ok (resolves to %s)%s", found, chainSuffix)
		lines = append(lines, label("hook:")+val)
		if chainProblem {
			problems = append(problems, Problem{Severity: SeverityError, Message: val})
		}
		return lines, problems
	}

	// Pre-G-0135 shape: absolute path baked at install time. Detect
	// the baked path; if it no longer exists, report stale and
	// recommend `aiwf update` (which refreshes to the PATH-lookup
	// shape).
	embedded := extractHookExecPath(string(raw))
	if embedded == "" {
		val := "aiwf-managed but malformed (no exec line found); run `aiwf init` to refresh"
		lines = append(lines, label("hook:")+val)
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
		return lines, problems
	}
	if _, statErr := os.Stat(embedded); statErr != nil {
		val := fmt.Sprintf("stale path %s — binary moved or removed; run `aiwf update` to refresh (post-G-0135 hooks resolve aiwf via PATH)", embedded)
		lines = append(lines, label("hook:")+val)
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
		return lines, problems
	}
	chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "pre-push")
	val := fmt.Sprintf("ok (%s; pre-G-0135 shape, run `aiwf update` to switch to PATH lookup)%s", embedded, chainSuffix)
	lines = append(lines, label("hook:")+val)
	if chainProblem {
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
	}
	return lines, problems
}

// resolveHooksDir returns the effective hooks directory for the
// repo at rootDir, falling back to `<rootDir>/.git/hooks` if the
// gitops query fails.
func resolveHooksDir(rootDir string) string {
	if dir, err := gitops.HooksDir(context.Background(), rootDir); err == nil {
		return dir
	}
	return filepath.Join(rootDir, ".git", "hooks")
}

// hookDisplayPath returns a hook file path for use in doctor's
// human-readable lines, expressed relative to rootDir when the
// hook lives under it.
func hookDisplayPath(rootDir, hookPath string) string {
	canonical := rootDir
	if resolved, err := filepath.EvalSymlinks(rootDir); err == nil {
		canonical = resolved
	}
	rel, err := filepath.Rel(canonical, hookPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return hookPath
	}
	return rel
}

// localChainSuffix returns the suffix to append to the hook line
// describing the `.local` sibling state plus a bool indicating
// whether the state is a problem.
func localChainSuffix(rootDir, hooksDir, hookName string) (suffix string, problem bool) {
	localPath := filepath.Join(hooksDir, hookName+".local")
	report := hookDisplayPath(rootDir, localPath)
	info, err := os.Stat(localPath)
	if errors.Is(err, os.ErrNotExist) {
		return "", false
	}
	if err != nil {
		return "; " + report + ": " + err.Error(), true
	}
	if info.Mode()&0o111 == 0 {
		return "; " + report + " exists but is not executable — chmod +x to enable, or remove the file", true
	}
	return "; chains to " + report, false
}

// appendPreCommitHookReport inspects .git/hooks/pre-commit and
// reports its state.
func appendPreCommitHookReport(in []string, problemsIn []Problem, rootDir string) (lines []string, problems []Problem) {
	lines = in
	problems = problemsIn

	hooksDir := resolveHooksDir(rootDir)
	hookPath := filepath.Join(hooksDir, "pre-commit")
	raw, err := os.ReadFile(hookPath)
	if errors.Is(err, os.ErrNotExist) {
		val := "missing — tree-discipline gate not installed; run `aiwf update`"
		lines = append(lines, label("pre-commit:")+val)
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
		return lines, problems
	}
	if err != nil {
		lines = append(lines, label("pre-commit:")+err.Error())
		problems = append(problems, Problem{Severity: SeverityError, Message: err.Error()})
		return lines, problems
	}
	if !strings.Contains(string(raw), initrepo.PreCommitHookMarker()) {
		val := "present but not aiwf-managed (no `# aiwf:pre-commit` marker); tree-discipline gate is not enforced"
		lines = append(lines, label("pre-commit:")+val)
		problems = append(problems, Problem{Severity: SeverityWarn, Message: val})
		return lines, problems
	}

	// Post-G-0135 / M-0133 / AC-1: hook resolves aiwf via PATH at
	// hook-fire time. Validate via exec.LookPath.
	if strings.Contains(string(raw), "command -v aiwf") {
		found, lookErr := exec.LookPath("aiwf")
		if lookErr != nil {
			val := "aiwf binary not found on PATH (hook would fail at commit time); install via `go install ./cmd/aiwf` and ensure $GOPATH/bin is on PATH"
			lines = append(lines, label("pre-commit:")+val)
			problems = append(problems, Problem{Severity: SeverityError, Message: val})
			return lines, problems
		}
		// G-0112 drift check (regen step in pre-commit is a regression).
		if strings.Contains(string(raw), "status --root") {
			val := "present with stale STATUS.md regen step (G-0112: regen moved to post-commit); run `aiwf update` to refresh"
			lines = append(lines, label("pre-commit:")+val)
			problems = append(problems, Problem{Severity: SeverityError, Message: val})
			return lines, problems
		}
		chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "pre-commit")
		val := fmt.Sprintf("ok (resolves to %s)%s", found, chainSuffix)
		lines = append(lines, label("pre-commit:")+val)
		if chainProblem {
			problems = append(problems, Problem{Severity: SeverityError, Message: val})
		}
		return lines, problems
	}

	// Pre-G-0135: absolute path baked at install time. Stale-path
	// check takes precedence over the G-0112 drift check because a
	// stale path means the hook can't run at all.
	embedded := extractPreCommitExecPath(string(raw))
	if embedded == "" {
		val := "aiwf-managed but malformed (no aiwf invocation found); run `aiwf update` to refresh"
		lines = append(lines, label("pre-commit:")+val)
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
		return lines, problems
	}
	if _, statErr := os.Stat(embedded); statErr != nil {
		val := fmt.Sprintf("stale path %s — binary moved or removed; run `aiwf update` to refresh (post-G-0135 hooks resolve aiwf via PATH)", embedded)
		lines = append(lines, label("pre-commit:")+val)
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
		return lines, problems
	}
	chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "pre-commit")
	if chainProblem {
		problems = append(problems, Problem{Severity: SeverityError, Message: strings.TrimPrefix(chainSuffix, "; ")})
	}
	if strings.Contains(string(raw), "status --root") {
		val := "present with stale STATUS.md regen step (G-0112: regen moved to post-commit); run `aiwf update` to refresh"
		lines = append(lines, label("pre-commit:")+val)
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
		return lines, problems
	}
	lines = append(lines, fmt.Sprintf("%sok (%s; pre-G-0135 shape, run `aiwf update` to switch to PATH lookup)%s", label("pre-commit:"), embedded, chainSuffix))
	return lines, problems
}

// appendCommitMsgHookReport inspects .git/hooks/commit-msg and
// reports its state (G-0218). The hook was born post-G-0135 (PATH
// lookup from day one), so there is no pre-G-0135 absolute-path
// shape to check for.
func appendCommitMsgHookReport(in []string, problemsIn []Problem, rootDir string) (lines []string, problems []Problem) {
	lines = in
	problems = problemsIn

	hooksDir := resolveHooksDir(rootDir)
	hookPath := filepath.Join(hooksDir, "commit-msg")
	raw, err := os.ReadFile(hookPath)
	if errors.Is(err, os.ErrNotExist) {
		val := "missing — G-0218 fabricated-trailer chokepoint not installed; run `aiwf update`"
		lines = append(lines, label("commit-msg:")+val)
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
		return lines, problems
	}
	if err != nil {
		lines = append(lines, label("commit-msg:")+err.Error())
		problems = append(problems, Problem{Severity: SeverityError, Message: err.Error()})
		return lines, problems
	}
	if !strings.Contains(string(raw), initrepo.CommitMsgHookMarker()) {
		val := "present but not aiwf-managed (no `# aiwf:commit-msg` marker); G-0218 fabricated-trailer chokepoint is not enforced"
		lines = append(lines, label("commit-msg:")+val)
		problems = append(problems, Problem{Severity: SeverityWarn, Message: val})
		return lines, problems
	}
	found, lookErr := exec.LookPath("aiwf")
	if lookErr != nil {
		val := "aiwf binary not found on PATH (hook would fail at commit time); install via `go install ./cmd/aiwf` and ensure $GOPATH/bin is on PATH"
		lines = append(lines, label("commit-msg:")+val)
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
		return lines, problems
	}
	chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "commit-msg")
	val := fmt.Sprintf("ok (resolves to %s)%s", found, chainSuffix)
	lines = append(lines, label("commit-msg:")+val)
	if chainProblem {
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
	}
	return lines, problems
}

// appendPostCommitHookReport inspects .git/hooks/post-commit and
// reports its state (G-0112).
func appendPostCommitHookReport(in []string, problemsIn []Problem, rootDir string) (lines []string, problems []Problem) {
	lines = in
	problems = problemsIn

	autoUpdate := true
	if cfg, err := config.Load(rootDir); err == nil {
		autoUpdate = cfg.StatusMdAutoUpdate()
	}

	hooksDir := resolveHooksDir(rootDir)
	hookPath := filepath.Join(hooksDir, "post-commit")
	raw, err := os.ReadFile(hookPath)
	if errors.Is(err, os.ErrNotExist) {
		if autoUpdate {
			val := "missing — STATUS.md will not regenerate (status_md.auto_update: true); run `aiwf update`"
			lines = append(lines, label("post-commit:")+val)
			problems = append(problems, Problem{Severity: SeverityError, Message: val})
			return lines, problems
		}
		lines = append(lines, label("post-commit:")+"not installed (status_md.auto_update: false; nothing to install)")
		return lines, problems
	}
	if err != nil {
		lines = append(lines, label("post-commit:")+err.Error())
		problems = append(problems, Problem{Severity: SeverityError, Message: err.Error()})
		return lines, problems
	}
	hasOurMarker := strings.Contains(string(raw), initrepo.PostCommitHookMarker())
	if !hasOurMarker {
		val := "present but not aiwf-managed (no `# aiwf:post-commit` marker); STATUS.md regen will not run"
		lines = append(lines, label("post-commit:")+val)
		problems = append(problems, Problem{Severity: SeverityWarn, Message: val})
		return lines, problems
	}
	if !autoUpdate {
		val := "present (aiwf-managed) but config says off (status_md.auto_update: false); run `aiwf update` to remove"
		lines = append(lines, label("post-commit:")+val)
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
		return lines, problems
	}
	// Post-G-0135 / M-0133 / AC-1: hook resolves aiwf via PATH at
	// hook-fire time. Validate via exec.LookPath.
	if strings.Contains(string(raw), "command -v aiwf") {
		found, lookErr := exec.LookPath("aiwf")
		if lookErr != nil {
			val := "aiwf binary not found on PATH (STATUS.md regen will skip silently); install via `go install ./cmd/aiwf` and ensure $GOPATH/bin is on PATH"
			lines = append(lines, label("post-commit:")+val)
			problems = append(problems, Problem{Severity: SeverityError, Message: val})
			return lines, problems
		}
		chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "post-commit")
		val := fmt.Sprintf("ok (resolves to %s)%s", found, chainSuffix)
		lines = append(lines, label("post-commit:")+val)
		if chainProblem {
			problems = append(problems, Problem{Severity: SeverityError, Message: val})
		}
		return lines, problems
	}

	// Pre-G-0135: absolute path baked at install time.
	embedded := extractPreCommitExecPath(string(raw))
	if embedded == "" {
		val := "aiwf-managed but malformed (no aiwf invocation found); run `aiwf update` to refresh"
		lines = append(lines, label("post-commit:")+val)
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
		return lines, problems
	}
	if _, statErr := os.Stat(embedded); statErr != nil {
		val := fmt.Sprintf("stale path %s — binary moved or removed; run `aiwf update` to refresh (post-G-0135 hooks resolve aiwf via PATH)", embedded)
		lines = append(lines, label("post-commit:")+val)
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
		return lines, problems
	}
	chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "post-commit")
	val := fmt.Sprintf("ok (%s; pre-G-0135 shape, run `aiwf update` to switch to PATH lookup)%s", embedded, chainSuffix)
	lines = append(lines, label("post-commit:")+val)
	if chainProblem {
		problems = append(problems, Problem{Severity: SeverityError, Message: val})
	}
	return lines, problems
}

// extractPreCommitExecPath pulls the binary path out of the
// pre-commit hook's `if 'path' status …` line.
func extractPreCommitExecPath(script string) string {
	for _, line := range strings.Split(script, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "if ") {
			continue
		}
		rest := strings.TrimPrefix(line, "if ")
		rest = strings.TrimPrefix(rest, "! ")
		if !strings.HasPrefix(rest, "'") {
			continue
		}
		rest = rest[1:]
		end := strings.IndexByte(rest, '\'')
		if end < 0 {
			return ""
		}
		return rest[:end]
	}
	return ""
}

// extractHookExecPath pulls the binary path out of the hook
// script's `exec '<path>' check` line.
func extractHookExecPath(script string) string {
	for _, line := range strings.Split(script, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "exec ") {
			continue
		}
		rest := strings.TrimPrefix(line, "exec ")
		if !strings.HasPrefix(rest, "'") {
			if idx := strings.IndexByte(rest, ' '); idx > 0 {
				return rest[:idx]
			}
			return rest
		}
		rest = rest[1:]
		end := strings.IndexByte(rest, '\'')
		if end < 0 {
			return ""
		}
		return rest[:end]
	}
	return ""
}

// filesystemCaseLabel returns "case-sensitive" or "case-insensitive"
// based on a probe inside dir.
func filesystemCaseLabel(dir string) string {
	probe, err := os.CreateTemp(dir, ".aiwf-case-probe-")
	if err != nil {
		return "unknown"
	}
	name := probe.Name()
	_ = probe.Close()
	defer func() { _ = os.Remove(name) }()
	upper := filepath.Join(filepath.Dir(name), strings.ToUpper(filepath.Base(name)))
	if _, err := os.Stat(upper); err == nil {
		return "case-insensitive"
	}
	return "case-sensitive"
}

// appendValidatorReport reads aiwf.yaml's contracts block and
// reports each configured validator's binary availability.
func appendValidatorReport(in []string, problemsIn []Problem, rootDir string) (lines []string, problems []Problem) {
	lines = in
	problems = problemsIn
	yamlPath := filepath.Join(rootDir, "aiwf.yaml")
	_, contracts, err := aiwfyaml.Read(yamlPath)
	if err != nil || contracts == nil || len(contracts.Validators) == 0 {
		return lines, problems
	}
	names := make([]string, 0, len(contracts.Validators))
	for n := range contracts.Validators {
		names = append(names, n)
	}
	sort.Strings(names)

	// missingVals holds the value text of each missing-validator line so
	// the strict branch can raise one error per missing binary (matching
	// the pre-[]Problem `problems += missing` count).
	var missingVals []string
	for _, n := range names {
		v := contracts.Validators[n]
		if _, lpErr := exec.LookPath(v.Command); lpErr == nil {
			lines = append(lines, fmt.Sprintf("%s%s ok (command=%s)", label("validator:"), n, v.Command))
		} else {
			val := fmt.Sprintf("%s missing (command=%s)", n, v.Command)
			lines = append(lines, label("validator:")+val)
			missingVals = append(missingVals, val)
		}
	}
	if len(missingVals) > 0 && contracts.StrictValidators {
		lines = append(lines, fmt.Sprintf("%s%d missing validator(s) and strict_validators=true; pre-push will fail", subIndent, len(missingVals)))
		for _, val := range missingVals {
			problems = append(problems, Problem{Severity: SeverityError, Message: val})
		}
	} else if len(missingVals) > 0 {
		warn := "missing binaries are warnings (strict_validators=false); pushes are not blocked"
		lines = append(lines,
			subIndent+warn,
			subIndent+"install the binary or set strict_validators=true to enforce on every machine",
		)
		problems = append(problems, Problem{Severity: SeverityWarn, Message: warn})
	}
	return lines, problems
}

// skillDrift compares each embedded skill against its on-disk copy
// and reports two sets: drifted and missing.
func skillDrift(rootDir string, embedded []skills.Skill) (drifted, missing []string) {
	for _, s := range embedded {
		on := filepath.Join(rootDir, skills.SkillsDir, s.Name, "SKILL.md")
		got, err := os.ReadFile(on)
		switch {
		case errors.Is(err, os.ErrNotExist):
			missing = append(missing, s.Name)
		case err != nil:
			drifted = append(drifted, s.Name+": "+err.Error())
		case !bytes.Equal(got, s.Content):
			drifted = append(drifted, s.Name)
		}
	}
	return drifted, missing
}

// renderLatestPublished formats the doctor latest: row.
func renderLatestPublished(current version.Info) string {
	latest, err := version.Latest(context.Background())
	switch {
	case errors.Is(err, version.ErrProxyDisabled):
		return "unavailable (proxy disabled — set GOPROXY to https://proxy.golang.org or remove `off` from the chain)"
	case err != nil:
		return fmt.Sprintf("unavailable (%v)", err)
	}
	switch version.Compare(current, latest) {
	case version.SkewEqual:
		return latest.Version + " (up to date)"
	case version.SkewBehind:
		return fmt.Sprintf("%s (binary at %s; run `aiwf upgrade`)", latest.Version, current.Version)
	case version.SkewAhead:
		return fmt.Sprintf("%s (binary newer at %s; rolled back?)", latest.Version, current.Version)
	default:
		return fmt.Sprintf("%s (binary at %s; skew unknown — devel or pseudo-version on either side)",
			latest.Version, current.Version)
	}
}

// currentBranch returns the short name of HEAD's symbolic ref
// in rootDir, or "" when HEAD is detached or git fails. M-0161/
// AC-7 uses the empty return as the detection signal.
func currentBranch(rootDir string) string {
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = rootDir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// headIsDetached double-checks the detached-state by verifying
// `git rev-parse HEAD` succeeds (so HEAD points at an object,
// just not via a symbolic ref). Distinguishes detached HEAD
// from "no commits yet" or other git failures, both of which
// currentBranch reports as "".
func headIsDetached(rootDir string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = rootDir
	return cmd.Run() == nil
}

// renderBinaryVersion formats a version.Info for the doctor binary
// row: the version string plus a parenthetical state.
func renderBinaryVersion(info version.Info) string {
	switch {
	case info.Version == version.DevelVersion:
		return info.Version + " (working-tree build)"
	case strings.HasSuffix(info.Version, "+dirty"):
		return info.Version + " (working-tree build)"
	case info.Tagged:
		return info.Version + " (tagged)"
	default:
		return info.Version + " (pseudo-version)"
	}
}
