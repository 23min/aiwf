package main

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

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/pluginstate"
	"github.com/23min/aiwf/internal/skills"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/version"
)

// newDoctorCmd builds `aiwf doctor`: version check, materialized-skill
// drift check, id-collision check. With --self-check, instead drives
// every mutating verb against a throwaway repo to prove the binary
// works end-to-end.
func newDoctorCmd() *cobra.Command {
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
			return cliutil.WrapExitCode(runDoctorCmd(root, selfCheck, checkLatest))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "consumer repo root")
	cmd.Flags().BoolVar(&selfCheck, "self-check", false, "run every verb against a temp repo and report pass/fail")
	cmd.Flags().BoolVar(&checkLatest, "check-latest", false, "look up the latest published aiwf version on the Go module proxy (one HTTP call; honors GOPROXY=off)")
	return cmd
}

func runDoctorCmd(root string, selfCheck, checkLatest bool) int {
	if selfCheck {
		return runSelfCheck()
	}

	rootDir, err := resolveRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aiwf doctor: %v\n", err)
		return cliutil.ExitUsage
	}

	report, problems := doctorReport(rootDir, doctorOptions{CheckLatest: checkLatest})
	for _, line := range report {
		fmt.Println(line)
	}
	if problems > 0 {
		return cliutil.ExitFindings
	}
	return cliutil.ExitOK
}

// doctorOptions carries flag-derived knobs into doctorReport. Kept
// separate from runDoctor's flag.FlagSet so doctorReport stays
// flag-package-free and unit-testable. Add fields here when new
// doctor flags arrive.
type doctorOptions struct {
	// CheckLatest, when true, performs a Go module proxy lookup for
	// the latest published aiwf version and adds a `latest:` row to
	// the report. Default false (offline).
	CheckLatest bool
}

// doctorReport collects every doctor finding into a slice of human
// strings and returns the count of problems. Pure for testability.
func doctorReport(rootDir string, opts doctorOptions) (lines []string, problems int) {
	// 1. Binary version (advisory). Always shown; reads from
	//    runtime/debug.ReadBuildInfo via version.Current().
	current := version.Current()
	lines = append(lines, fmt.Sprintf("binary:    %s", renderBinaryVersion(current)))

	// 1b. Latest published (advisory, opt-in). One HTTP call to the
	//     Go module proxy. Skipped unless --check-latest is set so
	//     `aiwf doctor` stays fast and offline by default.
	if opts.CheckLatest {
		lines = append(lines, "latest:    "+renderLatestPublished(current))
	}

	// 2. aiwf.yaml presence (advisory). Load-error states increment
	//    problems — those are real config faults. Two legacy fields
	//    surface as one-line deprecation hints so the user knows
	//    they're dead weight and `aiwf update` will strip them.
	cfg, err := config.Load(rootDir)
	switch {
	case errors.Is(err, config.ErrNotFound):
		lines = append(lines, "config:    aiwf.yaml not found (run `aiwf init`)")
		problems++
	case err != nil:
		lines = append(lines, "config:    "+err.Error())
		problems++
	default:
		lines = append(lines, "config:    ok")
		if cfg.LegacyActor != "" {
			// Pre-I2.5 `actor:` field. Identity is now runtime-derived
			// (per provenance-model.md); the file's value is ignored.
			lines = append(lines,
				fmt.Sprintf("           note: aiwf.yaml carries a deprecated `actor: %s` key — identity is now runtime-derived from git config user.email; the field is ignored. Run `aiwf update` to remove.", cfg.LegacyActor))
		}
		if cfg.LegacyAiwfVersion != "" {
			// Pre-G47 `aiwf_version:` pin. Set once at init, never
			// auto-maintained, produced chronic doctor noise. Now
			// dead — `aiwf version` reports the binary; the stored
			// pin is redundant.
			lines = append(lines,
				fmt.Sprintf("           note: aiwf.yaml carries a deprecated `aiwf_version: %s` key — version state is now derived from the binary (`aiwf version`); the field is ignored. Run `aiwf update` to remove.", cfg.LegacyAiwfVersion))
		}
	}

	// 1b. Runtime-identity resolution. Echoes what the next mutating
	//     verb's aiwf-actor: trailer would say, plus the source the
	//     value came from (--actor flag is absent here, so the source
	//     is git config user.email).
	if actor, source, actorErr := cliutil.ResolveActorWithSource("", rootDir); actorErr != nil {
		lines = append(lines, "actor:     "+actorErr.Error())
		problems++
	} else {
		lines = append(lines, fmt.Sprintf("actor:     %s (from %s)", actor, source))
	}

	// 2. Materialized-skill drift.
	embedded, err := skills.List()
	if err != nil {
		lines = append(lines, "skills:    "+err.Error())
		problems++
	} else {
		drift, missing := skillDrift(rootDir, embedded)
		switch {
		case len(missing) > 0:
			lines = append(lines, fmt.Sprintf("skills:    %d missing — run `aiwf init` or `aiwf update`", len(missing)))
			for _, m := range missing {
				lines = append(lines, "             - "+m)
			}
			problems++
		case len(drift) > 0:
			lines = append(lines, fmt.Sprintf("skills:    %d drifted — run `aiwf update` to refresh", len(drift)))
			for _, d := range drift {
				lines = append(lines, "             - "+d)
			}
			problems++
		default:
			lines = append(lines, fmt.Sprintf("skills:    ok (%d skills, byte-equal to embed)", len(embedded)))
		}
	}

	// 3. id-collision check (only ids-unique findings; all other
	// errors are reported by `aiwf check`).
	tr, loadErrs, err := tree.Load(context.Background(), rootDir)
	if err != nil {
		lines = append(lines, "ids:       "+err.Error())
		problems++
	} else {
		findings := check.Run(tr, loadErrs)
		collisions := 0
		for i := range findings {
			f := &findings[i]
			if f.Code == "ids-unique" {
				collisions++
				lines = append(lines, fmt.Sprintf("ids:       collision %s @ %s", f.EntityID, f.Path))
			}
		}
		if collisions == 0 {
			lines = append(lines, "ids:       ok (no collisions)")
		} else {
			problems++
		}
	}

	// 4. Configured contract validators: list each one and whether
	//    the binary is on PATH. A missing validator is reported but
	//    does not increment problems unless `strict_validators: true`
	//    is set — matches the contract verify rendering.
	lines, problems = appendValidatorReport(lines, problems, rootDir)

	// 5. Filesystem case-sensitivity. Informational; case-insensitive
	//    is the default on macOS APFS and Windows NTFS, and on those
	//    volumes E-01-foo and E-01-Foo collapse to the same dir.
	//    Users should know which they're on before they hit the
	//    footgun. The check.casePaths validator catches actual
	//    collisions; this line just surfaces the platform fact.
	lines = append(lines, fmt.Sprintf("filesystem: %s (%s)", filesystemCaseLabel(rootDir), rootDir))

	// 6. Pre-push hook: present, marker-tagged, and pointing at a
	//    binary that still exists. Catches the G12 drift case where
	//    `aiwf init` baked in an absolute path that's since moved.
	lines, problems = appendHookReport(lines, problems, rootDir)

	// 6b. Pre-commit hook: same drift detection. Per G-0112 the
	//     pre-commit hook is gate-only (the STATUS.md regen moved to
	//     the post-commit hook); a stale regen step is flagged for
	//     `aiwf update` to refresh.
	lines, problems = appendPreCommitHookReport(lines, problems, rootDir)

	// 6c. Post-commit hook (G-0112): present when
	//     status_md.auto_update is true, absent when false. A
	//     mismatch is flagged for `aiwf update` to reconcile.
	lines, problems = appendPostCommitHookReport(lines, problems, rootDir)

	// 4b. Render config: surface the configured out_dir and
	//     commit_output flag, plus the misconfiguration where
	//     commit_output: true but the gitignore still holds the
	//     out_dir line (recoverable by `aiwf update`).
	lines, problems = appendRenderReport(lines, problems, rootDir)

	// 5. Recommended-plugin presence (M-070). Reads
	//    `doctor.recommended_plugins` from aiwf.yaml and warns once per
	//    declared plugin not installed for this repo's project scope.
	//    Empty/absent list → zero observations; the kernel makes no
	//    assumption about which plugins a consumer "should" have.
	//    Severity: warning (problems unchanged) — refusing on absence
	//    is too strong for an advisory surface. Replaces the pre-M-070
	//    hardcoded `aiwf-extensions` heuristic that grepped
	//    `.claude/settings*.json`; see commit history for the prior
	//    behavior.
	lines = appendRecommendedPluginsReport(lines, cfg, rootDir)

	return lines, problems
}

// appendRecommendedPluginsReport emits one warning per recommended
// plugin not installed for the consumer's project scope. The check
// is opt-in via aiwf.yaml's `doctor.recommended_plugins` list (an
// empty/absent list is the kernel-neutral default — no observations).
//
// Warnings are soft: the function does not return a problem count
// because the M-070 spec ("Plugins are advisory; refusing on absence
// is too strong") forbids them from contributing to the doctor's
// non-zero exit code. Compare to other append* helpers that do return
// a problem count for hard findings.
//
// On read errors (corrupted installed_plugins.json, permission denied
// on the home dir) the function emits a single advisory line and skips
// the per-plugin checks; the absence case (file missing entirely) is
// deliberately treated as "no plugins installed" so every recommended
// plugin warns — see pluginstate.Load.
func appendRecommendedPluginsReport(in []string, cfg *config.Config, rootDir string) []string {
	if cfg == nil || len(cfg.Doctor.RecommendedPlugins) == 0 {
		return in
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return append(in, "plugins:   "+err.Error())
	}
	idx, err := pluginstate.Load(home)
	if err != nil {
		return append(in, "plugins:   "+err.Error())
	}
	out := in
	for _, plugin := range cfg.Doctor.RecommendedPlugins {
		ok, _ := idx.HasProjectScope(plugin, rootDir)
		if ok {
			continue
		}
		out = append(out,
			fmt.Sprintf("plugins:   recommended-plugin-not-installed: %s", plugin),
			fmt.Sprintf("             install: claude /plugin install %s", plugin),
		)
	}
	return out
}

// appendRenderReport surfaces the consumer's HTML render
// configuration plus a check for the false→true commit_output
// misconfiguration: when the consumer flips commit_output to true
// without re-running `aiwf update`, the gitignore still carries the
// stale `<out_dir>/` line and the rendered files are invisible to
// git. This block names the path, the flag, and the fix.
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
	lines = append(lines, fmt.Sprintf("render:    out_dir=%s commit_output=%s", outDir, commitLabel))

	// Misconfiguration check: commit_output: true but a gitignore
	// line for the out_dir still exists.
	if commitOutput {
		gitignorePath := filepath.Join(rootDir, ".gitignore")
		if raw, readErr := os.ReadFile(gitignorePath); readErr == nil {
			needle := strings.TrimRight(outDir, "/") + "/"
			for _, line := range strings.Split(string(raw), "\n") {
				if strings.TrimSpace(line) == needle {
					lines = append(lines,
						fmt.Sprintf("           drift: commit_output is true but .gitignore still holds %q; run `aiwf update` to reconcile", needle))
					problems++
					break
				}
			}
		}
	}
	return lines, problems
}

// appendHookReport inspects the pre-push hook at the consumer's
// effective hooks directory (default `<gitDir>/hooks/` or whatever
// `core.hooksPath` resolves to) and reports its state: missing,
// present-but-not-aiwf-managed, stale (the embedded absolute binary
// path no longer exists), or ok. A stale or missing-from-managed
// hook is a problem; a non-aiwf hook is a warning surfaced as
// informational text.
//
// Per G48 the path is resolved via gitops.HooksDir so a consumer
// with `core.hooksPath` set sees the correct hook reported.
func appendHookReport(in []string, problemsIn int, rootDir string) (lines []string, problems int) {
	lines = in
	problems = problemsIn

	hooksDir := resolveHooksDir(rootDir)
	hookPath := filepath.Join(hooksDir, "pre-push")
	raw, err := os.ReadFile(hookPath)
	if errors.Is(err, os.ErrNotExist) {
		lines = append(lines, "hook:      missing — pre-push validation not installed; run `aiwf init` to install")
		problems++
		return lines, problems
	}
	if err != nil {
		lines = append(lines, "hook:      "+err.Error())
		problems++
		return lines, problems
	}
	if !strings.Contains(string(raw), "# aiwf:pre-push") {
		lines = append(lines, "hook:      present but not aiwf-managed (no `# aiwf:pre-push` marker); aiwf check is not running pre-push")
		return lines, problems
	}
	// Extract the absolute path from `exec '<path>' check`.
	embedded := extractHookExecPath(string(raw))
	if embedded == "" {
		lines = append(lines, "hook:      aiwf-managed but malformed (no exec line found); run `aiwf init` to refresh")
		problems++
		return lines, problems
	}
	if _, statErr := os.Stat(embedded); statErr != nil {
		lines = append(lines, fmt.Sprintf("hook:      stale path %s — binary moved or removed; run `aiwf init` to refresh", embedded))
		problems++
		return lines, problems
	}
	chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "pre-push")
	if chainProblem {
		problems++
	}
	lines = append(lines, fmt.Sprintf("hook:      ok (%s)%s", embedded, chainSuffix))
	return lines, problems
}

// resolveHooksDir returns the effective hooks directory for the
// repo at rootDir, falling back to `<rootDir>/.git/hooks` if the
// gitops query fails. The fallback keeps doctor useful even when
// git's machinery is partially broken (e.g. a stripped repo with
// no `.git/config`); the report just describes the default layout
// instead of refusing to render.
func resolveHooksDir(rootDir string) string {
	if dir, err := gitops.HooksDir(context.Background(), rootDir); err == nil {
		return dir
	}
	return filepath.Join(rootDir, ".git", "hooks")
}

// hookDisplayPath returns a hook file path for use in doctor's
// human-readable lines, expressed relative to rootDir when the
// hook lives under it. Symlink-resolves rootDir to match the
// canonicalization gitops.HooksDir does. Falls back to the
// absolute path if the relative form would up-traverse outside
// the repo (e.g. `core.hooksPath` set to an absolute path
// elsewhere on disk).
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
// whether the state is a problem (non-executable). Mirrors the G45
// chain semantics in the installed hook script.
//
// hooksDir is the effective hooks directory (default `<gitDir>/hooks`
// or whatever `core.hooksPath` resolves to per G48). rootDir is the
// repo root, used to render the .local path in a friendlier form
// (relative to rootDir when the hook lives under it).
//
// States:
//   - sibling absent → "" (no suffix); not a problem.
//   - sibling present and executable → "; chains to <display>"; not a problem.
//   - sibling present, not executable → problem with chmod hint.
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
// reports its state. Per G42 the hook's sole responsibility is the
// G41 tree-discipline gate; per G-0112 the STATUS.md regen step
// moved to the post-commit hook, so pre-commit no longer toggles
// with `status_md.auto_update`. The hook always installs.
//
// States:
//   - hook missing → drift (the gate must always be present); problem.
//   - hook present, marker-managed, executable path resolves, regen
//     step absent → ok.
//   - hook present but carries a `status --root` regen step → stale
//     pre-G-0112 body; problem; remediated by `aiwf update`.
//   - hook present, no aiwf marker → user-written; we report
//     informationally but do not flag a problem.
func appendPreCommitHookReport(in []string, problemsIn int, rootDir string) (lines []string, problems int) {
	lines = in
	problems = problemsIn

	hooksDir := resolveHooksDir(rootDir)
	hookPath := filepath.Join(hooksDir, "pre-commit")
	raw, err := os.ReadFile(hookPath)
	if errors.Is(err, os.ErrNotExist) {
		// Per G42 the gate must always be installed when aiwf is
		// adopted in the repo.
		lines = append(lines, "pre-commit: missing — tree-discipline gate not installed; run `aiwf update`")
		problems++
		return lines, problems
	}
	if err != nil {
		lines = append(lines, "pre-commit: "+err.Error())
		problems++
		return lines, problems
	}
	if !strings.Contains(string(raw), "# aiwf:pre-commit") {
		lines = append(lines, "pre-commit: present but not aiwf-managed (no `# aiwf:pre-commit` marker); tree-discipline gate is not enforced")
		return lines, problems
	}
	embedded := extractPreCommitExecPath(string(raw))
	if embedded == "" {
		lines = append(lines, "pre-commit: aiwf-managed but malformed (no aiwf invocation found); run `aiwf update` to refresh")
		problems++
		return lines, problems
	}
	if _, statErr := os.Stat(embedded); statErr != nil {
		lines = append(lines, fmt.Sprintf("pre-commit: stale path %s — binary moved or removed; run `aiwf update` to refresh", embedded))
		problems++
		return lines, problems
	}
	chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "pre-commit")
	if chainProblem {
		problems++
	}
	// G-0112: a marker-managed pre-commit body must never carry the
	// STATUS.md regen step. The presence of one identifies a stale
	// pre-fix body that `aiwf update` will rewrite.
	if strings.Contains(string(raw), "status --root") {
		lines = append(lines, "pre-commit: present with stale STATUS.md regen step (G-0112: regen moved to post-commit); run `aiwf update` to refresh")
		problems++
		return lines, problems
	}
	lines = append(lines, fmt.Sprintf("pre-commit: ok (%s)%s", embedded, chainSuffix))
	return lines, problems
}

// appendPostCommitHookReport inspects .git/hooks/post-commit and
// reports its state (G-0112). The hook is opt-out via
// `status_md.auto_update`: with the default true, it must be
// installed and resolve to a valid binary; with auto_update: false,
// the desired state is "no marker-managed hook on disk".
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
			lines = append(lines, "post-commit: missing — STATUS.md will not regenerate (status_md.auto_update: true); run `aiwf update`")
			problems++
			return lines, problems
		}
		lines = append(lines, "post-commit: not installed (status_md.auto_update: false; nothing to install)")
		return lines, problems
	}
	if err != nil {
		lines = append(lines, "post-commit: "+err.Error())
		problems++
		return lines, problems
	}
	hasOurMarker := strings.Contains(string(raw), "# aiwf:post-commit")
	if !hasOurMarker {
		// User-written hook. Report informationally; don't flag a
		// problem because the user opted into managing it themselves.
		lines = append(lines, "post-commit: present but not aiwf-managed (no `# aiwf:post-commit` marker); STATUS.md regen will not run")
		return lines, problems
	}
	if !autoUpdate {
		lines = append(lines, "post-commit: present (aiwf-managed) but config says off (status_md.auto_update: false); run `aiwf update` to remove")
		problems++
		return lines, problems
	}
	embedded := extractPreCommitExecPath(string(raw))
	if embedded == "" {
		lines = append(lines, "post-commit: aiwf-managed but malformed (no aiwf invocation found); run `aiwf update` to refresh")
		problems++
		return lines, problems
	}
	if _, statErr := os.Stat(embedded); statErr != nil {
		lines = append(lines, fmt.Sprintf("post-commit: stale path %s — binary moved or removed; run `aiwf update` to refresh", embedded))
		problems++
		return lines, problems
	}
	chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "post-commit")
	if chainProblem {
		problems++
	}
	lines = append(lines, fmt.Sprintf("post-commit: ok (%s)%s", embedded, chainSuffix))
	return lines, problems
}

// extractPreCommitExecPath pulls the binary path out of the
// pre-commit hook's `if 'path' status …` line. Returns empty when
// the line cannot be located (malformed hook).
func extractPreCommitExecPath(script string) string {
	for _, line := range strings.Split(script, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "if ") {
			continue
		}
		rest := strings.TrimPrefix(line, "if ")
		// G42 introduced a `! ` negation form for the tree-discipline
		// gate (`if ! 'path' check --shape-only ...`). Strip the
		// negation if present so the quote scan finds the binary path.
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

// extractHookExecPath pulls the binary path out of the hook script's
// `exec '<path>' check` line. Returns empty when no such line is
// found (malformed hook).
func extractHookExecPath(script string) string {
	for _, line := range strings.Split(script, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "exec ") {
			continue
		}
		// `exec '/path/to/aiwf' check` — pull the single-quoted segment.
		rest := strings.TrimPrefix(line, "exec ")
		if !strings.HasPrefix(rest, "'") {
			// Bare exec word; take the first token before space.
			if idx := strings.IndexByte(rest, ' '); idx > 0 {
				return rest[:idx]
			}
			return rest
		}
		// Find the closing quote.
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
// based on a probe inside dir: write a temp file, stat its name in
// uppercase, and check whether the filesystem returned the same
// inode. If the probe fails (permissions, no temp space), returns
// "unknown" so the report stays informational rather than blocking.
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
// reports each configured validator's binary availability. A
// missing binary is a problem only when strict_validators is set;
// otherwise it's a soft note matching the runtime warning.
func appendValidatorReport(in []string, problemsIn int, rootDir string) (lines []string, problems int) {
	lines = in
	problems = problemsIn
	yamlPath := filepath.Join(rootDir, "aiwf.yaml")
	_, contracts, err := aiwfyaml.Read(yamlPath)
	if err != nil || contracts == nil || len(contracts.Validators) == 0 {
		// No contracts block (or unreadable yaml — already reported
		// by step 1 above). Skip the section silently.
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
			lines = append(lines, fmt.Sprintf("validator: %s ok (command=%s)", n, v.Command))
		} else {
			lines = append(lines, fmt.Sprintf("validator: %s missing (command=%s)", n, v.Command))
			missing++
		}
	}
	if missing > 0 && contracts.StrictValidators {
		lines = append(lines, fmt.Sprintf("             %d missing validator(s) and strict_validators=true; pre-push will fail", missing))
		problems += missing
	} else if missing > 0 {
		lines = append(lines,
			"             missing binaries are warnings (strict_validators=false); pushes are not blocked",
			"             install the binary or set strict_validators=true to enforce on every machine",
		)
	}
	return lines, problems
}

// skillDrift compares each embedded skill against its on-disk copy
// and reports two sets: drifted (file exists but differs) and missing
// (file absent).
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

// renderLatestPublished formats the doctor latest: row. Calls
// version.Latest with a fresh context, classifying the result:
//
//	v0.2.0 (up to date)
//	v0.2.1 (binary at v0.2.0; run `aiwf upgrade`)
//	v0.1.0 (binary newer at v0.2.0; rolled back?)
//	unavailable (proxy disabled — set GOPROXY to https://proxy.golang.org or override)
//	unavailable (timeout / network error)
//	skew unknown (latest is a pseudo-version; module has no tags yet)
//
// Network errors and proxy-disabled never increment the doctor
// problem count: the row is informational, and absent connectivity
// is not a fault of the running aiwf install.
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
		// Either side non-tagged. Most common case in the early-PoC
		// world: the module has no semver tags yet, so the proxy
		// returns a pseudo-version. Surface it honestly.
		return fmt.Sprintf("%s (binary at %s; skew unknown — devel or pseudo-version on either side)",
			latest.Version, current.Version)
	}
}

// renderBinaryVersion formats a version.Info for the doctor binary
// row: the version string plus a parenthetical state ("tagged",
// "working-tree build", "pseudo-version"). Mirrors the upgrade
// verb's renderVersionLabel; kept separate to avoid an admin →
// upgrade dependency.
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
