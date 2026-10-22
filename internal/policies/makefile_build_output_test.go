package policies

import (
	"regexp"
	"strings"
	"testing"
)

// TestMakefileBuild_LeavesNoBinAiwf pins M-0363/AC-3: `make build` writes
// no executable named `aiwf` under bin/, where it would be run in place of
// the installed release, and `make selfcheck` runs the binary `make build`
// produced.
func TestMakefileBuild_LeavesNoBinAiwf(t *testing.T) {
	t.Parallel()
	output := regexp.MustCompile(`-o (\S+) \./cmd/aiwf`)

	m := output.FindStringSubmatch(makeDryRun(t, repoRoot(t), "build"))
	if m == nil {
		t.Fatal("make build's recipe names no -o output for ./cmd/aiwf")
	}
	if built := m[1]; built == "bin/aiwf" {
		t.Errorf("make build writes %s, which is reached for in place of the installed aiwf", built)
	}
	if selfcheck := makeDryRun(t, repoRoot(t), "selfcheck"); !strings.Contains(selfcheck, "./"+m[1]+" doctor --self-check") {
		t.Errorf("make selfcheck does not run the binary make build produced (%s):\n%s", m[1], selfcheck)
	}
}
