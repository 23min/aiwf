package policies

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPolicy_RaceParallelCap(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyRaceParallelCap)
}

// TestPolicyRaceParallelCap_MissingRaceLineStatesTheFix pins M-0337 AC-2: a
// target with no race-mode invocation is told what to add or drop.
func TestPolicyRaceParallelCap_MissingRaceLineStatesTheFix(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "Makefile"), "test:\n\tgo test ./...\n")
	mustWrite(t, filepath.Join(root, ".github", "workflows", "go.yml"), "run: go test -race -parallel 8 ./...\n")
	mustWrite(t, filepath.Join(root, ".github", "workflows", "flake-hunt.yml"), "run: go test -race -parallel 8 ./...\n")
	vs, err := PolicyRaceParallelCap(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 || !strings.Contains(vs[0].Detail, "add the race-mode invocation as `go test -race -parallel 8 ...`, or drop the file from the targets") {
		t.Errorf("violations = %+v, want one on Makefile stating the fix", vs)
	}
}
