package doctor

import (
	"os"
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
func InContainer() (bool, string) {
	return detectContainer("/.dockerenv", os.Getenv("AIWF_DEVCONTAINER"))
}

// detectContainer is the testable shape of InContainer. It takes the
// dockerenv-path and AIWF_DEVCONTAINER value explicitly so unit tests
// can exercise the full signal-combination matrix without touching the
// filesystem root or mutating process-global env.
func detectContainer(dockerenvPath, devcontainerEnv string) (bool, string) {
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
