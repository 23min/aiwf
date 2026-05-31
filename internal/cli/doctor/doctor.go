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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
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
		root        string
		selfCheck   bool
		checkLatest bool
	)
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Drift / version / id-collision health check",
		Example: `  # Local health check on the current consumer repo
  aiwf doctor

  # Drive every verb against a throwaway repo (CI smoke test)
  aiwf doctor --self-check

  # Compare to the latest published release on the Go module proxy
  aiwf doctor --check-latest`,
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return cliutil.WrapExitCode(Run(root, selfCheck, checkLatest))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().BoolVar(&selfCheck, "self-check", false, "run every verb against a temp repo and report pass/fail")
	cmd.Flags().BoolVar(&checkLatest, "check-latest", false, "look up the latest published aiwf version on the Go module proxy (one HTTP call; honors GOPROXY=off)")
	return cmd
}

// Run is the exported entry point for `aiwf doctor`. Dispatches to
// the report-only path or --self-check based on the flag.
func Run(root string, selfCheck, checkLatest bool) int {
	if selfCheck {
		return runSelfCheck()
	}

	rootDir, err := cliutil.ResolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf doctor: %v\n", err)
		return cliutil.ExitUsage
	}

	report, problems := DoctorReport(rootDir, DoctorOptions{CheckLatest: checkLatest})
	for _, line := range report {
		fmt.Println(line)
	}
	if problems > 0 {
		return cliutil.ExitFindings
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
// strings and returns the count of problems. Pure for testability.
func DoctorReport(rootDir string, opts DoctorOptions) (lines []string, problems int) {
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
		lines = append(lines, label("config:")+"aiwf.yaml not found (run `aiwf init`)")
		problems++
	case err != nil:
		lines = append(lines, label("config:")+err.Error())
		problems++
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

	if actor, source, actorErr := cliutil.ResolveActorWithSource("", rootDir); actorErr != nil {
		lines = append(lines, label("actor:")+actorErr.Error())
		problems++
	} else {
		lines = append(lines, fmt.Sprintf("%s%s (from %s)", label("actor:"), actor, source))
	}

	embedded, err := skills.List()
	if err != nil {
		lines = append(lines, label("skills:")+err.Error())
		problems++
	} else {
		drift, missing := skillDrift(rootDir, embedded)
		switch {
		case len(missing) > 0:
			lines = append(lines, fmt.Sprintf("%s%d missing — run `aiwf init` or `aiwf update`", label("skills:"), len(missing)))
			for _, m := range missing {
				lines = append(lines, subIndent+"- "+m)
			}
			problems++
		case len(drift) > 0:
			lines = append(lines, fmt.Sprintf("%s%d drifted — run `aiwf update` to refresh", label("skills:"), len(drift)))
			for _, d := range drift {
				lines = append(lines, subIndent+"- "+d)
			}
			problems++
		default:
			lines = append(lines, fmt.Sprintf("%sok (%d skills, byte-equal to embed)", label("skills:"), len(embedded)))
		}
	}

	tr, loadErrs, err := tree.Load(context.Background(), rootDir)
	if err != nil {
		lines = append(lines, label("ids:")+err.Error())
		problems++
	} else {
		findings := check.Run(tr, loadErrs)
		collisions := 0
		for i := range findings {
			f := &findings[i]
			if f.Code == check.CodeIDsUnique {
				collisions++
				lines = append(lines, fmt.Sprintf("%scollision %s @ %s", label("ids:"), f.EntityID, f.Path))
			}
		}
		if collisions == 0 {
			lines = append(lines, label("ids:")+"ok (no collisions)")
		} else {
			problems++
		}
	}

	lines, problems = appendValidatorReport(lines, problems, rootDir)

	lines = append(lines, fmt.Sprintf("%s%s (%s)", label("filesystem:"), filesystemCaseLabel(rootDir), rootDir))

	lines, problems = appendHookReport(lines, problems, rootDir)
	lines, problems = appendPreCommitHookReport(lines, problems, rootDir)
	lines, problems = appendPostCommitHookReport(lines, problems, rootDir)
	lines, problems = appendRenderReport(lines, problems, rootDir)
	lines = appendMaterializedRitualsReport(lines, rootDir)
	lines = appendMarketplaceOverlapReport(lines, rootDir)
	lines = appendStatuslineReport(lines, rootDir)

	return lines, problems
}

// ritualsMarketplaceSuffix is the `@<marketplace>` suffix of a rituals
// plugin id as it appears in `.claude/settings.json` enabledPlugins
// (e.g. `aiwf-extensions@ai-workflow-rituals`). Used by the de-dupe
// guard to recognize an enabled marketplace plugin that overlaps with
// the materialized rituals.
const ritualsMarketplaceSuffix = "@ai-workflow-rituals"

// appendMaterializedRitualsReport verifies the embedded ritual
// artifacts (skills, agents, templates) are materialized under the
// consumer's `.claude/` tree (ADR-0014 §5 — doctor verifies the
// materialized artifacts instead of recommending a marketplace plugin).
// A `rituals:` ok line confirms presence; a soft warning naming the
// missing artifacts points at `aiwf update`. Rituals are advisory
// artifacts, so a miss never increments the problem count.
func appendMaterializedRitualsReport(in []string, rootDir string) []string {
	present, missing, err := skills.MaterializedRituals(rootDir, skills.ClaudeTarget)
	if err != nil {
		return append(in, label("rituals:")+err.Error())
	}
	if len(missing) > 0 {
		out := in
		out = append(out, fmt.Sprintf("%s%d of %d ritual artifacts not materialized — run `aiwf update`", label("rituals:"), len(missing), len(present)+len(missing)))
		for _, m := range missing {
			out = append(out, subIndent+"- "+m)
		}
		return out
	}
	return append(in,
		fmt.Sprintf("%sok (%d artifacts materialized)", label("rituals:"), len(present)),
		subIndent+"managed by aiwf (skills aiwf-*/aiwfx-*/wf-*, agents, templates); `aiwf update` refreshes — do not hand-edit (see .claude/skills/README.md)",
	)
}

// appendMarketplaceOverlapReport is the de-dupe guard (ADR-0014 §5):
// when the consumer has a rituals marketplace plugin enabled in
// `.claude/settings.json` AND the rituals are materialized under
// `.claude/`, the same skill `name:` is exposed twice. The guard
// detects the overlap and instructs the operator to disable the plugin —
// the marketplace-plugin scenario is not consent-eligible (quiet
// mutation of user settings is more invasive than the marker-managed
// posture allows). The narrow exception aiwf does take on is the
// statusline opt-in (`--wire-settings` / TTY `[y/N]`), gated by explicit
// per-invocation consent per ADR-0015; this de-dupe guard does not
// participate in that flow. Soft (advisory): it does not increment the
// problem count.
func appendMarketplaceOverlapReport(in []string, rootDir string) []string {
	enabled, err := loadEnabledPlugins(rootDir)
	if err != nil {
		return append(in, label("plugins:")+err.Error())
	}
	var enabledRituals []string
	for id, on := range enabled {
		if on && strings.HasSuffix(id, ritualsMarketplaceSuffix) {
			enabledRituals = append(enabledRituals, id)
		}
	}
	if len(enabledRituals) == 0 {
		return in
	}
	present, _, mErr := skills.MaterializedRituals(rootDir, skills.ClaudeTarget)
	if mErr != nil || len(present) == 0 {
		// No materialized rituals → no duplication hazard (or the
		// materialized-rituals report above already surfaced mErr).
		return in
	}
	sort.Strings(enabledRituals)
	out := in
	out = append(out, fmt.Sprintf("%smarketplace-rituals-overlap: rituals materialized AND %d marketplace plugin(s) enabled — disable the plugin(s) to avoid duplicate skills", label("plugins:"), len(enabledRituals)))
	for _, id := range enabledRituals {
		out = append(out, fmt.Sprintf("%s- %s (disable via the `/plugin` menu; aiwf does not edit your settings.json without explicit per-invocation consent — see ADR-0015 — and the marketplace-overlap scenario is not consent-eligible)", subIndent, id))
	}
	return out
}

// loadEnabledPlugins reads the project's `.claude/settings.json` and
// returns its `enabledPlugins` map. The map key is `name@marketplace`;
// the value is true when the project declares the plugin enabled.
//
// Missing file returns an empty map (no plugins declared) without
// error; malformed JSON returns a wrapped error so doctor can surface
// it as a configuration issue rather than silently treating it as
// "no plugins enabled."
func loadEnabledPlugins(rootDir string) (map[string]bool, error) {
	path := filepath.Join(rootDir, ".claude", "settings.json")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading .claude/settings.json: %w", err)
	}
	var doc struct {
		EnabledPlugins map[string]bool `json:"enabledPlugins"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parsing .claude/settings.json: %w", err)
	}
	return doc.EnabledPlugins, nil
}

// appendRenderReport surfaces the consumer's HTML render
// configuration plus a check for the false→true commit_output
// misconfiguration: when the consumer flips commit_output to true
// without re-running `aiwf update`, the gitignore still carries the
// stale `<out_dir>/` line and the rendered files are invisible to
// git.
func appendRenderReport(in []string, problemsIn int, rootDir string) (lines []string, problems int) {
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
					lines = append(lines,
						fmt.Sprintf("%sdrift: commit_output is true but .gitignore still holds %q; run `aiwf update` to reconcile", subIndent, needle))
					problems++
					break
				}
			}
		}
	}
	return lines, problems
}

// appendHookReport inspects the pre-push hook at the consumer's
// effective hooks directory and reports its state.
func appendHookReport(in []string, problemsIn int, rootDir string) (lines []string, problems int) {
	lines = in
	problems = problemsIn

	hooksDir := resolveHooksDir(rootDir)
	hookPath := filepath.Join(hooksDir, "pre-push")
	raw, err := os.ReadFile(hookPath)
	if errors.Is(err, os.ErrNotExist) {
		lines = append(lines, label("hook:")+"missing — pre-push validation not installed; run `aiwf init` to install")
		problems++
		return lines, problems
	}
	if err != nil {
		lines = append(lines, label("hook:")+err.Error())
		problems++
		return lines, problems
	}
	if !strings.Contains(string(raw), "# aiwf:pre-push") {
		lines = append(lines, label("hook:")+"present but not aiwf-managed (no `# aiwf:pre-push` marker); aiwf check is not running pre-push")
		return lines, problems
	}

	// Post-G-0135 / M-0133 / AC-1: hooks resolve aiwf via PATH lookup
	// at hook-fire time. Validate the binary is reachable on PATH via
	// exec.LookPath rather than stat'ing a baked path.
	if strings.Contains(string(raw), "command -v aiwf") {
		found, lookErr := exec.LookPath("aiwf")
		if lookErr != nil {
			lines = append(lines, label("hook:")+"aiwf binary not found on PATH (hook would fail at push time); install via `go install ./cmd/aiwf` and ensure $GOPATH/bin is on PATH")
			problems++
			return lines, problems
		}
		chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "pre-push")
		if chainProblem {
			problems++
		}
		lines = append(lines, fmt.Sprintf("%sok (resolves to %s)%s", label("hook:"), found, chainSuffix))
		return lines, problems
	}

	// Pre-G-0135 shape: absolute path baked at install time. Detect
	// the baked path; if it no longer exists, report stale and
	// recommend `aiwf update` (which refreshes to the PATH-lookup
	// shape).
	embedded := extractHookExecPath(string(raw))
	if embedded == "" {
		lines = append(lines, label("hook:")+"aiwf-managed but malformed (no exec line found); run `aiwf init` to refresh")
		problems++
		return lines, problems
	}
	if _, statErr := os.Stat(embedded); statErr != nil {
		lines = append(lines, fmt.Sprintf("%sstale path %s — binary moved or removed; run `aiwf update` to refresh (post-G-0135 hooks resolve aiwf via PATH)", label("hook:"), embedded))
		problems++
		return lines, problems
	}
	chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "pre-push")
	if chainProblem {
		problems++
	}
	lines = append(lines, fmt.Sprintf("%sok (%s; pre-G-0135 shape, run `aiwf update` to switch to PATH lookup)%s", label("hook:"), embedded, chainSuffix))
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
func appendPreCommitHookReport(in []string, problemsIn int, rootDir string) (lines []string, problems int) {
	lines = in
	problems = problemsIn

	hooksDir := resolveHooksDir(rootDir)
	hookPath := filepath.Join(hooksDir, "pre-commit")
	raw, err := os.ReadFile(hookPath)
	if errors.Is(err, os.ErrNotExist) {
		lines = append(lines, label("pre-commit:")+"missing — tree-discipline gate not installed; run `aiwf update`")
		problems++
		return lines, problems
	}
	if err != nil {
		lines = append(lines, label("pre-commit:")+err.Error())
		problems++
		return lines, problems
	}
	if !strings.Contains(string(raw), "# aiwf:pre-commit") {
		lines = append(lines, label("pre-commit:")+"present but not aiwf-managed (no `# aiwf:pre-commit` marker); tree-discipline gate is not enforced")
		return lines, problems
	}

	// Post-G-0135 / M-0133 / AC-1: hook resolves aiwf via PATH at
	// hook-fire time. Validate via exec.LookPath.
	if strings.Contains(string(raw), "command -v aiwf") {
		found, lookErr := exec.LookPath("aiwf")
		if lookErr != nil {
			lines = append(lines, label("pre-commit:")+"aiwf binary not found on PATH (hook would fail at commit time); install via `go install ./cmd/aiwf` and ensure $GOPATH/bin is on PATH")
			problems++
			return lines, problems
		}
		// G-0112 drift check (regen step in pre-commit is a regression).
		if strings.Contains(string(raw), "status --root") {
			lines = append(lines, label("pre-commit:")+"present with stale STATUS.md regen step (G-0112: regen moved to post-commit); run `aiwf update` to refresh")
			problems++
			return lines, problems
		}
		chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "pre-commit")
		if chainProblem {
			problems++
		}
		lines = append(lines, fmt.Sprintf("%sok (resolves to %s)%s", label("pre-commit:"), found, chainSuffix))
		return lines, problems
	}

	// Pre-G-0135: absolute path baked at install time. Stale-path
	// check takes precedence over the G-0112 drift check because a
	// stale path means the hook can't run at all.
	embedded := extractPreCommitExecPath(string(raw))
	if embedded == "" {
		lines = append(lines, label("pre-commit:")+"aiwf-managed but malformed (no aiwf invocation found); run `aiwf update` to refresh")
		problems++
		return lines, problems
	}
	if _, statErr := os.Stat(embedded); statErr != nil {
		lines = append(lines, fmt.Sprintf("%sstale path %s — binary moved or removed; run `aiwf update` to refresh (post-G-0135 hooks resolve aiwf via PATH)", label("pre-commit:"), embedded))
		problems++
		return lines, problems
	}
	chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "pre-commit")
	if chainProblem {
		problems++
	}
	if strings.Contains(string(raw), "status --root") {
		lines = append(lines, label("pre-commit:")+"present with stale STATUS.md regen step (G-0112: regen moved to post-commit); run `aiwf update` to refresh")
		problems++
		return lines, problems
	}
	lines = append(lines, fmt.Sprintf("%sok (%s; pre-G-0135 shape, run `aiwf update` to switch to PATH lookup)%s", label("pre-commit:"), embedded, chainSuffix))
	return lines, problems
}

// appendPostCommitHookReport inspects .git/hooks/post-commit and
// reports its state (G-0112).
func appendPostCommitHookReport(in []string, problemsIn int, rootDir string) (lines []string, problems int) {
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
			lines = append(lines, label("post-commit:")+"missing — STATUS.md will not regenerate (status_md.auto_update: true); run `aiwf update`")
			problems++
			return lines, problems
		}
		lines = append(lines, label("post-commit:")+"not installed (status_md.auto_update: false; nothing to install)")
		return lines, problems
	}
	if err != nil {
		lines = append(lines, label("post-commit:")+err.Error())
		problems++
		return lines, problems
	}
	hasOurMarker := strings.Contains(string(raw), "# aiwf:post-commit")
	if !hasOurMarker {
		lines = append(lines, label("post-commit:")+"present but not aiwf-managed (no `# aiwf:post-commit` marker); STATUS.md regen will not run")
		return lines, problems
	}
	if !autoUpdate {
		lines = append(lines, label("post-commit:")+"present (aiwf-managed) but config says off (status_md.auto_update: false); run `aiwf update` to remove")
		problems++
		return lines, problems
	}
	// Post-G-0135 / M-0133 / AC-1: hook resolves aiwf via PATH at
	// hook-fire time. Validate via exec.LookPath.
	if strings.Contains(string(raw), "command -v aiwf") {
		found, lookErr := exec.LookPath("aiwf")
		if lookErr != nil {
			lines = append(lines, label("post-commit:")+"aiwf binary not found on PATH (STATUS.md regen will skip silently); install via `go install ./cmd/aiwf` and ensure $GOPATH/bin is on PATH")
			problems++
			return lines, problems
		}
		chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "post-commit")
		if chainProblem {
			problems++
		}
		lines = append(lines, fmt.Sprintf("%sok (resolves to %s)%s", label("post-commit:"), found, chainSuffix))
		return lines, problems
	}

	// Pre-G-0135: absolute path baked at install time.
	embedded := extractPreCommitExecPath(string(raw))
	if embedded == "" {
		lines = append(lines, label("post-commit:")+"aiwf-managed but malformed (no aiwf invocation found); run `aiwf update` to refresh")
		problems++
		return lines, problems
	}
	if _, statErr := os.Stat(embedded); statErr != nil {
		lines = append(lines, fmt.Sprintf("%sstale path %s — binary moved or removed; run `aiwf update` to refresh (post-G-0135 hooks resolve aiwf via PATH)", label("post-commit:"), embedded))
		problems++
		return lines, problems
	}
	chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "post-commit")
	if chainProblem {
		problems++
	}
	lines = append(lines, fmt.Sprintf("%sok (%s; pre-G-0135 shape, run `aiwf update` to switch to PATH lookup)%s", label("post-commit:"), embedded, chainSuffix))
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
func appendValidatorReport(in []string, problemsIn int, rootDir string) (lines []string, problems int) {
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

	missing := 0
	for _, n := range names {
		v := contracts.Validators[n]
		if _, lpErr := exec.LookPath(v.Command); lpErr == nil {
			lines = append(lines, fmt.Sprintf("%s%s ok (command=%s)", label("validator:"), n, v.Command))
		} else {
			lines = append(lines, fmt.Sprintf("%s%s missing (command=%s)", label("validator:"), n, v.Command))
			missing++
		}
	}
	if missing > 0 && contracts.StrictValidators {
		lines = append(lines, fmt.Sprintf("%s%d missing validator(s) and strict_validators=true; pre-push will fail", subIndent, missing))
		problems += missing
	} else if missing > 0 {
		lines = append(lines,
			subIndent+"missing binaries are warnings (strict_validators=false); pushes are not blocked",
			subIndent+"install the binary or set strict_validators=true to enforce on every machine",
		)
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
