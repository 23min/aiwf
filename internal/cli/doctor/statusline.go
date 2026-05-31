package doctor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/23min/aiwf/internal/skills"
)

// appendStatuslineReport adds the statusline advisory block to the
// doctor output. Production entry point — resolves home via
// os.UserHomeDir(). Tests use appendStatuslineReportWithHome.
//
// M-0157.
func appendStatuslineReport(in []string, rootDir string) []string {
	home, _ := os.UserHomeDir()
	inContainer, _ := InContainer()
	return appendStatuslineReportWithHome(in, rootDir, home, inContainer)
}

// appendStatuslineReportWithHome is the testable core. Emitted only
// when `.claude/statusline.sh` exists in the repo (project scope) or
// the user's home (user scope). Advisory only — never increments the
// problem count.
//
// Reports: dep availability (jq, gh), wiring state, embedded-vs-on-disk
// drift, and a container + project-scope nudge toward --scope user.
func appendStatuslineReportWithHome(in []string, rootDir, home string, inContainer bool) []string {
	projectPath := filepath.Join(rootDir, ".claude", "statusline.sh")
	userPath := ""
	if home != "" {
		userPath = filepath.Join(home, ".claude", "statusline.sh")
	}

	installedPath, scope := resolveInstalledStatusline(projectPath, userPath)
	if installedPath == "" {
		return in
	}

	out := in
	out = append(out, fmt.Sprintf("%sinstalled (%s scope: %s)", label("statusline:"), scope, installedPath))

	out = appendDepCheck(out, "jq", jqInstallHint())
	out = appendDepCheck(out, "gh", ghInstallHint())

	out = appendWiringCheck(out, rootDir, home, scope)
	out = appendDriftCheck(out, installedPath)

	if inContainer && scope == "project" {
		out = append(out, subIndent+"nudge: running in a container with project scope — consider `aiwf update --statusline --scope user` so the statusline works across all repos in this container")
	}

	return out
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
		cmdPath := statuslineCmdPathForScope(scope, filepath.Join(rootDir, ".claude", "statusline.sh"))
		return append(in,
			subIndent+"wiring: statusLine key not found in any settings file — the script is installed but inactive",
			subIndent+fmt.Sprintf("run `aiwf update --statusline --wire-settings` or add to your settings: %s", skills.FormatStatuslineSnippet(cmdPath)),
		)
	}
	return in
}

// statuslineCmdPathForScope returns the command path for the
// snippet based on scope — relative for project, absolute for user.
func statuslineCmdPathForScope(scope, installedPath string) string {
	if scope == "project" {
		return ".claude/statusline.sh"
	}
	return installedPath
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

// appendDriftCheck compares the on-disk statusline to the embedded
// copy and reports when they differ.
func appendDriftCheck(in []string, installedPath string) []string {
	onDisk, err := os.ReadFile(installedPath)
	if err != nil {
		return in
	}
	embedded := skills.StatuslineBytes()
	if len(embedded) == 0 {
		return in
	}
	if !bytes.Equal(onDisk, embedded) {
		return append(in, subIndent+"drift: on-disk statusline differs from the embedded copy — run `aiwf update --statusline` to see the latest (your edits will be preserved; only a fresh scaffold overwrites)")
	}
	return in
}
