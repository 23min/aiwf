package doctor

import (
	"fmt"
	"os"
	"path/filepath"
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
// rendered into the doctor `plugin-index-mount:` line.
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

// renderMountLine formats the plugin-index-mount: line for the
// doctor report. Caller separates the gating (only emit when
// InContainer()) from the rendering (always-safe to call).
//
// M-0135/AC-2.
func renderMountLine(state mountState, count int, errMsg string) string {
	switch state {
	case mountStateOK:
		if count >= shadowMountCountCap {
			return fmt.Sprintf("plugin-index-mount: ok (%d+ plugin entries cached)", count)
		}
		return fmt.Sprintf("plugin-index-mount: ok (%d plugin entries cached)", count)
	case mountStateEmpty:
		return "plugin-index-mount: empty (mount target exists but no plugin entries — first rebuild before initialize.sh, or shadow-mount not yet seeded)"
	case mountStateMissing:
		return "plugin-index-mount: missing (mount target does not exist — devcontainer.json mount entry stripped or container rebuild failed mid-postcreate)"
	case mountStateError:
		return "plugin-index-mount: " + errMsg
	default:
		return "plugin-index-mount: unknown"
	}
}
