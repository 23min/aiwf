package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/23min/aiwf/internal/skills"
	"github.com/23min/aiwf/internal/version"
)

// appendStatuslineReport adds the statusline advisory block to the
// doctor output. Production entry point — resolves home via
// os.UserHomeDir(). Tests use appendStatuslineReportWithHome.
//
// M-0157.
func appendStatuslineReport(in []string, problemsIn []Problem, rootDir string) (lines []string, problems []Problem) {
	home, _ := os.UserHomeDir()
	inContainer, _ := InContainer()
	return appendStatuslineReportWithHome(in, problemsIn, rootDir, home, inContainer)
}

// appendStatuslineReportWithHome is the testable core. Emitted only
// when `.claude/statusline.sh` exists in the repo (project scope) or
// the user's home (user scope). Advisories surface as SeverityWarn
// problems (never SeverityError) — they inform but do not gate exit.
//
// Reports: dep availability (jq, gh), wiring state, embedded-vs-on-disk
// drift, and a container + project-scope nudge toward --scope user.
func appendStatuslineReportWithHome(in []string, problemsIn []Problem, rootDir, home string, inContainer bool) (lines []string, problems []Problem) {
	problems = problemsIn
	projectPath := filepath.Join(rootDir, ".claude", "statusline.sh")
	userPath := ""
	if home != "" {
		userPath = filepath.Join(home, ".claude", "statusline.sh")
	}

	installedPath, scope := resolveInstalledStatusline(projectPath, userPath)
	if installedPath == "" {
		return in, problems
	}

	out := in
	out = append(out, fmt.Sprintf("%sinstalled (%s scope: %s)", label("statusline:"), scope, installedPath))
	// The `installed` header is a status line, not an advisory; every
	// sub-line the checks below emit is an actionable warning.
	advisoryStart := len(out)

	out = appendDepCheck(out, "jq", jqInstallHint())
	out = appendDepCheck(out, "gh", ghInstallHint())

	out = appendWiringCheck(out, rootDir, home, scope)
	out = appendPrecedenceCheck(out, rootDir, home)
	out = appendProjectCommandCheck(out, rootDir)
	out = appendDriftCheck(out, installedPath)

	if inContainer && scope == "project" {
		out = append(out, subIndent+"nudge: running in a container with project scope — consider `aiwf update --statusline --scope user` so the statusline works across all repos in this container")
	}

	// Surface each advisory sub-line as a SeverityWarn without disturbing
	// the byte-for-byte report lines the checks already produced.
	for _, ln := range out[advisoryStart:] {
		problems = append(problems, Problem{Severity: SeverityWarn, Message: strings.TrimSpace(ln)})
	}

	return out, problems
}

// resolveInstalledStatusline returns the path and scope label of the
// installed statusline, preferring project scope over user scope.
// Returns ("", "") when neither exists.
func resolveInstalledStatusline(projectPath, userPath string) (path, scope string) {
	if _, err := os.Stat(projectPath); err == nil {
		return projectPath, "project"
	}
	if userPath != "" {
		if _, err := os.Stat(userPath); err == nil {
			return userPath, "user"
		}
	}
	return "", ""
}

// appendDepCheck adds a sub-line for a missing dependency binary.
func appendDepCheck(in []string, name, hint string) []string {
	if _, err := exec.LookPath(name); err != nil {
		return append(in, fmt.Sprintf("%sdep: %s not found — %s", subIndent, name, hint))
	}
	return in
}

// jqInstallHint returns a platform-branched install hint for jq.
func jqInstallHint() string {
	return installHintFor("jq", runtime.GOOS)
}

// ghInstallHint returns a platform-branched install hint for gh.
func ghInstallHint() string {
	return installHintFor("gh", runtime.GOOS)
}

// installHintFor returns a platform-branched install hint. Exposed as
// a testable function so both platforms are exercised on any host.
func installHintFor(tool, goos string) string {
	switch goos {
	case "darwin":
		return fmt.Sprintf("`brew install %s`", tool)
	default:
		return fmt.Sprintf("`sudo apt-get install %s` (or your distro's package manager)", tool)
	}
}

// appendWiringCheck adds a sub-line when the statusline is installed
// but no settings file contains a statusLine key.
func appendWiringCheck(in []string, rootDir, home, scope string) []string {
	wired := false

	for _, name := range []string{"settings.local.json", "settings.json"} {
		path := filepath.Join(rootDir, ".claude", name)
		if hasStatusLineKey(path) {
			wired = true
			break
		}
	}

	if !wired && home != "" {
		if hasStatusLineKey(filepath.Join(home, ".claude", "settings.json")) {
			wired = true
		}
	}

	if !wired {
		cmdPath := statuslineCmdPathForScope(scope, rootDir)
		return append(in,
			subIndent+"wiring: statusLine key not found in any settings file — the script is installed but inactive",
			subIndent+fmt.Sprintf("run `aiwf update --statusline --wire-settings` or add to your settings: %s", skills.FormatStatuslineSnippet(cmdPath)),
		)
	}
	return in
}

// statuslineCmdPathForScope returns the `statusLine.command` value the
// remediation hint should show, reusing the skills single-source helpers
// so the hint matches what wiring actually writes (G-0337).
func statuslineCmdPathForScope(scope, rootDir string) string {
	if scope == "project" {
		return skills.ProjectStatuslineCommand(rootDir)
	}
	return skills.UserStatuslineCommand()
}

// appendPrecedenceCheck warns when a statusLine is wired in BOTH a
// project settings file and the user settings file. Claude Code's
// project settings take precedence, so the project key silently wins and
// shadows the user one — the trap that let G-0337 hide (a correct
// user-scope wiring rendered nothing because a stale project key
// overrode it).
func appendPrecedenceCheck(in []string, rootDir, home string) []string {
	projWired := hasStatusLineKey(filepath.Join(rootDir, ".claude", "settings.local.json")) ||
		hasStatusLineKey(filepath.Join(rootDir, ".claude", "settings.json"))
	userWired := home != "" && hasStatusLineKey(filepath.Join(home, ".claude", "settings.json"))
	if projWired && userWired {
		return append(in, subIndent+"precedence: a statusLine is wired in BOTH project and user settings — the project key wins and shadows the user one; remove the project statusLine to use the user-scope one")
	}
	return in
}

// appendProjectCommandCheck warns when a project-scope statusLine command
// cannot resolve from a git worktree: a bare cwd-relative path (breaks
// the moment the session cwd is a worktree), or a
// `${CLAUDE_PROJECT_DIR:-<fallback>}` whose fallback path no longer
// exists (stale after a move/remount, as in the G-0337 report). Reads the
// project settings only — user-scope `$HOME` commands are inherently
// resolvable.
func appendProjectCommandCheck(in []string, rootDir string) []string {
	cmd := ""
	for _, name := range []string{"settings.local.json", "settings.json"} {
		if c := statusLineCommand(filepath.Join(rootDir, ".claude", name)); c != "" {
			cmd = c
			break
		}
	}
	if cmd == "" {
		return in
	}
	if !strings.HasPrefix(cmd, "/") && !strings.HasPrefix(cmd, "$") && !strings.HasPrefix(cmd, "~") {
		return append(in, subIndent+fmt.Sprintf("command: project statusLine %q is cwd-relative — it will not resolve in a git worktree; re-run `aiwf update --statusline` or switch to `--scope user`", cmd))
	}
	if fb := resolvedFallbackPath(cmd); fb != "" {
		if _, err := os.Stat(fb); err != nil {
			return append(in, subIndent+fmt.Sprintf("command: project statusLine fallback %q does not resolve (e.g. stale after a move/remount) — re-run `aiwf update --statusline` or switch to `--scope user`", fb))
		}
	}
	return in
}

// statusLineCommand reads a settings file and returns its
// statusLine.command value, or "" on any error or absence.
func statusLineCommand(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return ""
	}
	sl, ok := obj["statusLine"]
	if !ok {
		return ""
	}
	var v struct {
		Command string `json:"command"`
	}
	if json.Unmarshal(sl, &v) != nil {
		return ""
	}
	return v.Command
}

// resolvedFallbackPath extracts the fallback-resolved path from a
// `${CLAUDE_PROJECT_DIR:-<fallback>}<tail>` command — i.e. the path the
// command resolves to when CLAUDE_PROJECT_DIR is unset. Returns "" when
// the command is not in that form.
func resolvedFallbackPath(cmd string) string {
	const prefix = "${CLAUDE_PROJECT_DIR:-"
	rest, ok := strings.CutPrefix(cmd, prefix)
	if !ok {
		return ""
	}
	fallback, tail, ok := strings.Cut(rest, "}")
	if !ok {
		return ""
	}
	return fallback + tail
}

// hasStatusLineKey reads a JSON settings file and reports whether it
// contains a top-level "statusLine" key. Returns false on any error
// (missing file, malformed JSON, etc.) — best-effort, advisory.
func hasStatusLineKey(path string) bool {
	raw, rErr := os.ReadFile(path)
	if rErr != nil {
		return false
	}
	var obj map[string]json.RawMessage
	if uErr := json.Unmarshal(raw, &obj); uErr != nil {
		return false
	}
	_, ok := obj["statusLine"]
	return ok
}

// appendDriftCheck reports two independent statusline signals (G-0344):
//
//   - version relationship — how the installed copy's stamped version
//     relates to this binary's: an unmarked (legacy/foreign) copy, a
//     newer version shipping with this binary (refresh available), or an
//     installed version *newer* than this binary (which a plain
//     `aiwf update` deliberately does not downgrade — the fleet-safety
//     signal for a shared `~/.claude`).
//   - body drift — the script body (ignoring the version-marker line)
//     differs from the embedded copy, i.e. a local edit.
//
// The version line and the drift line are orthogonal: a copy can be the
// current version yet body-edited, or an older version yet
// byte-faithful. Both are SeverityWarn advisories (the caller tags them).
func appendDriftCheck(in []string, installedPath string) []string {
	onDisk, err := os.ReadFile(installedPath)
	if err != nil {
		return in //coverage:ignore installedPath was just resolved via os.Stat by resolveInstalledStatusline; a read failure here is a TOCTOU race unreachable from tests
	}
	return append(in, statuslineVersionLines(onDisk, version.Current())...)
}

// statuslineVersionLines returns the version-relationship and body-drift
// advisory sub-lines for an installed statusline copy, given the running
// binary's version. Pure and version-injectable so the
// ahead/behind/unmarked/equal branches are unit-testable without a real
// tagged binary (a `go test` binary always reports `(devel)`; G-0344).
func statuslineVersionLines(onDisk []byte, binary version.Info) []string {
	installedRaw, marked := skills.InstalledStatuslineVersion(onDisk)
	if !marked {
		return []string{subIndent + "version: installed statusline carries no aiwf version marker (legacy or foreign copy) — run `aiwf update --statusline` once to adopt versioned auto-refresh"}
	}
	var out []string
	switch version.Compare(binary, version.Parse(installedRaw)) {
	case version.SkewAhead:
		out = append(out, subIndent+fmt.Sprintf("version: statusline %s installed but this aiwf ships %s — run `aiwf update` to refresh", installedRaw, binary.Version))
	case version.SkewBehind:
		// Installed is a newer aiwf version. Report it, but return
		// without the body-drift hint below: a newer version legitimately
		// carries a different body, and the drift hint's remediation
		// (`aiwf update --statusline`, which rewrites to this older
		// binary's version) would downgrade the very copy we just said we
		// will not downgrade.
		return append(out, subIndent+fmt.Sprintf("version: installed statusline %s is newer than this aiwf binary %s — not downgraded (a shared ~/.claude across differing aiwf versions?)", installedRaw, binary.Version))
	}
	// Body drift is reported independently of the version relationship,
	// with the marker line normalized out so a mere version difference
	// is not mistaken for a local edit.
	if skills.StatuslineBodyDrifted(onDisk) {
		out = append(out, subIndent+"drift: on-disk statusline body differs from the embedded copy — run `aiwf update --statusline` to refresh it (the script is aiwf-owned; local edits are overwritten)")
	}
	return out
}
