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

// DoctorReport collects every doctor finding into a slice of human
// strings and returns the count of problems. Pure for testability.
func DoctorReport(rootDir string, opts DoctorOptions) (lines []string, problems int) {
	current := version.Current()
	lines = append(lines, fmt.Sprintf("binary:    %s", renderBinaryVersion(current)))

	if opts.CheckLatest {
		lines = append(lines, "latest:    "+renderLatestPublished(current))
	}

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
			lines = append(lines,
				fmt.Sprintf("           note: aiwf.yaml carries a deprecated `actor: %s` key — identity is now runtime-derived from git config user.email; the field is ignored. Run `aiwf update` to remove.", cfg.LegacyActor))
		}
		if cfg.LegacyAiwfVersion != "" {
			lines = append(lines,
				fmt.Sprintf("           note: aiwf.yaml carries a deprecated `aiwf_version: %s` key — version state is now derived from the binary (`aiwf version`); the field is ignored. Run `aiwf update` to remove.", cfg.LegacyAiwfVersion))
		}
	}

	if actor, source, actorErr := cliutil.ResolveActorWithSource("", rootDir); actorErr != nil {
		lines = append(lines, "actor:     "+actorErr.Error())
		problems++
	} else {
		lines = append(lines, fmt.Sprintf("actor:     %s (from %s)", actor, source))
	}

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

	lines, problems = appendValidatorReport(lines, problems, rootDir)

	lines = append(lines, fmt.Sprintf("filesystem: %s (%s)", filesystemCaseLabel(rootDir), rootDir))

	lines, problems = appendHookReport(lines, problems, rootDir)
	lines, problems = appendPreCommitHookReport(lines, problems, rootDir)
	lines, problems = appendPostCommitHookReport(lines, problems, rootDir)
	lines, problems = appendRenderReport(lines, problems, rootDir)
	lines = appendRecommendedPluginsReport(lines, cfg, rootDir)

	return lines, problems
}

// appendRecommendedPluginsReport emits one warning per recommended
// plugin not installed for the consumer's project scope. The check
// is opt-in via aiwf.yaml's `doctor.recommended_plugins` list.
//
// Warnings are soft: the function does not return a problem count
// because the M-070 spec forbids them from contributing to the
// doctor's non-zero exit code.
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
	lines = append(lines, fmt.Sprintf("render:    out_dir=%s commit_output=%s", outDir, commitLabel))

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
// effective hooks directory and reports its state.
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

	// Post-G-0135 / M-0133 / AC-1: hooks resolve aiwf via PATH lookup
	// at hook-fire time. Validate the binary is reachable on PATH via
	// exec.LookPath rather than stat'ing a baked path.
	if strings.Contains(string(raw), "command -v aiwf") {
		found, lookErr := exec.LookPath("aiwf")
		if lookErr != nil {
			lines = append(lines, "hook:      aiwf binary not found on PATH (hook would fail at push time); install via `go install ./cmd/aiwf` and ensure $GOPATH/bin is on PATH")
			problems++
			return lines, problems
		}
		chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "pre-push")
		if chainProblem {
			problems++
		}
		lines = append(lines, fmt.Sprintf("hook:      ok (resolves to %s)%s", found, chainSuffix))
		return lines, problems
	}

	// Pre-G-0135 shape: absolute path baked at install time. Detect
	// the baked path; if it no longer exists, report stale and
	// recommend `aiwf update` (which refreshes to the PATH-lookup
	// shape).
	embedded := extractHookExecPath(string(raw))
	if embedded == "" {
		lines = append(lines, "hook:      aiwf-managed but malformed (no exec line found); run `aiwf init` to refresh")
		problems++
		return lines, problems
	}
	if _, statErr := os.Stat(embedded); statErr != nil {
		lines = append(lines, fmt.Sprintf("hook:      stale path %s — binary moved or removed; run `aiwf update` to refresh (post-G-0135 hooks resolve aiwf via PATH)", embedded))
		problems++
		return lines, problems
	}
	chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "pre-push")
	if chainProblem {
		problems++
	}
	lines = append(lines, fmt.Sprintf("hook:      ok (%s; pre-G-0135 shape, run `aiwf update` to switch to PATH lookup)%s", embedded, chainSuffix))
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

	// Post-G-0135 / M-0133 / AC-1: hook resolves aiwf via PATH at
	// hook-fire time. Validate via exec.LookPath.
	if strings.Contains(string(raw), "command -v aiwf") {
		found, lookErr := exec.LookPath("aiwf")
		if lookErr != nil {
			lines = append(lines, "pre-commit: aiwf binary not found on PATH (hook would fail at commit time); install via `go install ./cmd/aiwf` and ensure $GOPATH/bin is on PATH")
			problems++
			return lines, problems
		}
		// G-0112 drift check (regen step in pre-commit is a regression).
		if strings.Contains(string(raw), "status --root") {
			lines = append(lines, "pre-commit: present with stale STATUS.md regen step (G-0112: regen moved to post-commit); run `aiwf update` to refresh")
			problems++
			return lines, problems
		}
		chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "pre-commit")
		if chainProblem {
			problems++
		}
		lines = append(lines, fmt.Sprintf("pre-commit: ok (resolves to %s)%s", found, chainSuffix))
		return lines, problems
	}

	// Pre-G-0135: absolute path baked at install time. Stale-path
	// check takes precedence over the G-0112 drift check because a
	// stale path means the hook can't run at all.
	embedded := extractPreCommitExecPath(string(raw))
	if embedded == "" {
		lines = append(lines, "pre-commit: aiwf-managed but malformed (no aiwf invocation found); run `aiwf update` to refresh")
		problems++
		return lines, problems
	}
	if _, statErr := os.Stat(embedded); statErr != nil {
		lines = append(lines, fmt.Sprintf("pre-commit: stale path %s — binary moved or removed; run `aiwf update` to refresh (post-G-0135 hooks resolve aiwf via PATH)", embedded))
		problems++
		return lines, problems
	}
	chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "pre-commit")
	if chainProblem {
		problems++
	}
	if strings.Contains(string(raw), "status --root") {
		lines = append(lines, "pre-commit: present with stale STATUS.md regen step (G-0112: regen moved to post-commit); run `aiwf update` to refresh")
		problems++
		return lines, problems
	}
	lines = append(lines, fmt.Sprintf("pre-commit: ok (%s; pre-G-0135 shape, run `aiwf update` to switch to PATH lookup)%s", embedded, chainSuffix))
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
		lines = append(lines, "post-commit: present but not aiwf-managed (no `# aiwf:post-commit` marker); STATUS.md regen will not run")
		return lines, problems
	}
	if !autoUpdate {
		lines = append(lines, "post-commit: present (aiwf-managed) but config says off (status_md.auto_update: false); run `aiwf update` to remove")
		problems++
		return lines, problems
	}
	// Post-G-0135 / M-0133 / AC-1: hook resolves aiwf via PATH at
	// hook-fire time. Validate via exec.LookPath.
	if strings.Contains(string(raw), "command -v aiwf") {
		found, lookErr := exec.LookPath("aiwf")
		if lookErr != nil {
			lines = append(lines, "post-commit: aiwf binary not found on PATH (STATUS.md regen will skip silently); install via `go install ./cmd/aiwf` and ensure $GOPATH/bin is on PATH")
			problems++
			return lines, problems
		}
		chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "post-commit")
		if chainProblem {
			problems++
		}
		lines = append(lines, fmt.Sprintf("post-commit: ok (resolves to %s)%s", found, chainSuffix))
		return lines, problems
	}

	// Pre-G-0135: absolute path baked at install time.
	embedded := extractPreCommitExecPath(string(raw))
	if embedded == "" {
		lines = append(lines, "post-commit: aiwf-managed but malformed (no aiwf invocation found); run `aiwf update` to refresh")
		problems++
		return lines, problems
	}
	if _, statErr := os.Stat(embedded); statErr != nil {
		lines = append(lines, fmt.Sprintf("post-commit: stale path %s — binary moved or removed; run `aiwf update` to refresh (post-G-0135 hooks resolve aiwf via PATH)", embedded))
		problems++
		return lines, problems
	}
	chainSuffix, chainProblem := localChainSuffix(rootDir, hooksDir, "post-commit")
	if chainProblem {
		problems++
	}
	lines = append(lines, fmt.Sprintf("post-commit: ok (%s; pre-G-0135 shape, run `aiwf update` to switch to PATH lookup)%s", embedded, chainSuffix))
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
