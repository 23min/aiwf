package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// InContainer reports whether aiwf is running inside a containerised
// environment, and returns a human-readable label naming the detected
// signal(s). The two signals checked:
//
//   - `/.dockerenv` file existence (Docker convention; set by the
//     Docker runtime regardless of orchestrator).
//   - `AIWF_DEVCONTAINER` env var with a truthy value (`1` / `true`,
//     case-insensitive). Set by `.devcontainer/devcontainer.json`'s
//     `containerEnv` map for repos using the M-0132 devcontainer.
//
// Returns the boolean state plus the rendered label suitable for the
// `env:` line in `aiwf doctor` output:
//
//   - both signals → `devcontainer (/.dockerenv + AIWF_DEVCONTAINER)`
//   - only dockerenv → `devcontainer (/.dockerenv)`
//   - only env var → `devcontainer (AIWF_DEVCONTAINER)`
//   - neither → `host`
//
// The detection is informational; consumers of this function never
// increment doctor's problem count based on the result.
//
// M-0135/AC-1.
func InContainer() (inContainer bool, label string) {
	return detectContainer("/.dockerenv", os.Getenv("AIWF_DEVCONTAINER"))
}

// detectContainer is the testable shape of InContainer. It takes the
// dockerenv-path and AIWF_DEVCONTAINER value explicitly so unit tests
// can exercise the full signal-combination matrix without touching the
// filesystem root or mutating process-global env.
func detectContainer(dockerenvPath, devcontainerEnv string) (inContainer bool, label string) {
	var signals []string
	if _, err := os.Stat(dockerenvPath); err == nil {
		signals = append(signals, "/.dockerenv")
	}
	if isTruthy(devcontainerEnv) {
		signals = append(signals, "AIWF_DEVCONTAINER")
	}
	if len(signals) == 0 {
		return false, "host"
	}
	return true, "devcontainer (" + strings.Join(signals, " + ") + ")"
}

// isTruthy reports whether s is a truthy literal (1 or true,
// case-insensitive). Empty, `0`, `false`, and anything else is falsy.
func isTruthy(s string) bool {
	switch strings.ToLower(s) {
	case "1", "true":
		return true
	default:
		return false
	}
}

// mountState enumerates the observable shapes of the shadow-mount
// target at `<home>/.claude/plugins/`. Used by shadowMountStatus and
// rendered into the doctor `plugin-mount:` line.
//
// M-0135/AC-2.
type mountState int

const (
	mountStateUnknown mountState = iota
	mountStateOK                 // directory exists, has ≥1 non-hidden entry
	mountStateEmpty              // directory exists, no non-hidden entries
	mountStateMissing            // directory absent, or target is a regular file
	mountStateError              // os-level error probing the path
)

// shadowMountCountCap caps the operator-facing entry count so doctor
// output stays scannable on unusual setups with hundreds of cached
// plugin entries. The threshold is opinionated; matches typical cache
// sizes per the AC-2 spec.
const shadowMountCountCap = 100

// shadowMountStatus probes the bind-mount target
// `<home>/.claude/plugins/` and returns the observable state plus a
// non-hidden top-level entry count (capped at shadowMountCountCap).
//
// The shadow-mount workaround backs the in-container
// `~/.claude/plugins/` onto the host's `~/.claude-linux/plugins/`
// per .devcontainer/devcontainer.json. This probe only inspects the
// in-container target; the host side is the operator's responsibility
// (the devcontainer's initialize.sh seeds it).
//
// home is passed explicitly (rather than calling os.UserHomeDir
// internally) so callers can fall back through their own home-
// resolution strategy, and so tests can run against a t.TempDir
// fixture without mutating HOME for the whole process.
//
// M-0135/AC-2.
func shadowMountStatus(home string) (mountState, int, error) {
	target := pluginsTargetPath(home)
	info, err := os.Stat(target)
	if os.IsNotExist(err) {
		return mountStateMissing, 0, nil
	}
	if err != nil {
		return mountStateError, 0, err
	}
	if !info.IsDir() {
		return mountStateMissing, 0, nil
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return mountStateError, 0, err
	}
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		count++
		if count >= shadowMountCountCap {
			break
		}
	}
	if count == 0 {
		return mountStateEmpty, 0, nil
	}
	return mountStateOK, count, nil
}

// pluginsTargetPath returns the in-container shadow-mount target
// `<home>/.claude/plugins/`.
func pluginsTargetPath(home string) string {
	return filepath.Join(home, ".claude", "plugins")
}

// renderMountLine formats the plugin-mount: line for the
// doctor report. Caller separates the gating (only emit when
// InContainer()) from the rendering (always-safe to call).
//
// M-0135/AC-2.
func renderMountLine(state mountState, count int, errMsg string) string {
	switch state {
	case mountStateOK:
		if count >= shadowMountCountCap {
			return fmt.Sprintf("plugin-mount: ok (%d+ plugin entries cached)", count)
		}
		return fmt.Sprintf("plugin-mount: ok (%d plugin entries cached)", count)
	case mountStateEmpty:
		return "plugin-mount: empty (mount target exists but no plugin entries — first rebuild before initialize.sh, or shadow-mount not yet seeded)"
	case mountStateMissing:
		return "plugin-mount: missing (mount target does not exist — devcontainer.json mount entry stripped or container rebuild failed mid-postcreate)"
	case mountStateError:
		return "plugin-mount: " + errMsg
	default:
		return "plugin-mount: unknown"
	}
}

// foreignHomePrefix returns the home-root path prefix that is foreign
// to the running OS — the marker of an anthropics/claude-code#31388
// cross-platform plugin-index leak. `/Users/` is macOS's home root; if
// it shows up in an index read on Linux (i.e. inside the container),
// the index was written by a macOS host and Claude's marketplace
// refresh will reject it. Returns "" on any non-Linux host, which makes
// foreignPluginPaths a no-op there — the in-container Linux case is the
// one that recurs (G-0174); the inverse is left until a forcing case
// appears (YAGNI).
func foreignHomePrefix() string {
	return foreignHomePrefixFor(runtime.GOOS)
}

// foreignHomePrefixFor is the testable shape of foreignHomePrefix: it
// takes the GOOS value explicitly so both arms are exercised on any
// platform, mirroring InContainer/detectContainer.
func foreignHomePrefixFor(goos string) string {
	if goos == "linux" {
		return "/Users/"
	}
	return ""
}

// pluginIndexFiles returns the Claude plugin-index files under
// `<home>/.claude/plugins/` that carry absolute install paths. The
// large `plugin-catalog-cache.json` is deliberately excluded — it
// holds skill content, not install locations, so scanning it would be
// slow and noise-prone.
func pluginIndexFiles(home string) []string {
	dir := pluginsTargetPath(home)
	return []string{
		filepath.Join(dir, "known_marketplaces.json"),
		filepath.Join(dir, "installed_plugins.json"),
	}
}

// foreignPluginPaths scans the plugin-index files for an absolute path
// value rooted at foreignPrefix — the claude-code#31388 corruption,
// where a macOS-pathed index (`/Users/...`) leaks into a Linux
// container (or the inverse). It returns the first offending path found
// (the operator only needs one example) and whether any were found.
//
// foreignPrefix is passed explicitly so tests can drive both directions
// without depending on runtime.GOOS. An empty prefix short-circuits to
// "not found" (the non-Linux no-op). The scan is best-effort: a missing
// or unreadable index file, or one that is mid-write / malformed JSON,
// is skipped silently rather than surfaced — the check is advisory and
// must never block doctor.
//
// Closes G-0174.
func foreignPluginPaths(home, foreignPrefix string) (sample string, found bool) {
	if foreignPrefix == "" {
		return "", false
	}
	for _, path := range pluginIndexFiles(home) {
		data, err := os.ReadFile(path)
		if err != nil {
			continue // missing or unreadable — best-effort scan
		}
		var doc any
		if err := json.Unmarshal(data, &doc); err != nil {
			continue // mid-write or malformed — skip silently
		}
		if s, ok := firstForeignPathLeaf(doc, foreignPrefix); ok {
			return s, true
		}
	}
	return "", false
}

// firstForeignPathLeaf walks a decoded-JSON value and returns the first
// string leaf that begins with foreignPrefix. Walking string leaves
// (rather than keying on specific field names) handles both index file
// shapes uniformly — the marketplace map and the installed-plugins
// nested arrays — and stays correct if Claude renames a path field.
func firstForeignPathLeaf(v any, foreignPrefix string) (string, bool) {
	switch t := v.(type) {
	case string:
		if strings.HasPrefix(t, foreignPrefix) {
			return t, true
		}
	case map[string]any:
		for _, child := range t {
			if s, ok := firstForeignPathLeaf(child, foreignPrefix); ok {
				return s, true
			}
		}
	case []any:
		for _, child := range t {
			if s, ok := firstForeignPathLeaf(child, foreignPrefix); ok {
				return s, true
			}
		}
	}
	return "", false
}

// renderPluginPathHintLine formats the advisory `plugin-paths:` line —
// the claude-code#31388 hint. It fires only when foreignPluginPaths
// found a leak, and the caller never counts it as a problem (advisory,
// like the env: and plugin-mount: lines).
//
// Closes G-0174.
func renderPluginPathHintLine(sample string) string {
	return "plugin-paths: advisory — index holds a foreign-OS path (" + sample +
		"); it was written on another OS, so Claude marketplace refresh will fail (claude-code#31388). " +
		"Fix: confirm the .devcontainer plugin-index shadow-mount is active (.devcontainer/initialize.sh), " +
		"or run `claude plugin marketplace remove <name>` and re-add at project scope."
}
