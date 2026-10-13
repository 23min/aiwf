package policies

import (
	"path/filepath"
	"testing"
)

func TestPolicy_RaceParallelCap(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyRaceParallelCap)
}

// TestPolicyRaceParallelCap_MessagesStateTheFix pins M-0337 AC-2: a
// race-mode invocation without the cap is told to add it, and a target
// with no race-mode invocation is told what to add or drop.
func TestPolicyRaceParallelCap_MessagesStateTheFix(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "Makefile"), "test:\n\tgo test ./...\n")
	mustWrite(t, filepath.Join(root, ".github", "workflows", "go.yml"), "run: go test -race ./...\n")
	mustWrite(t, filepath.Join(root, ".github", "workflows", "flake-hunt.yml"), "run: go test -race -parallel 8 ./...\n")
	vs, err := PolicyRaceParallelCap(root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveViolation(t, vs, "add `-parallel 8` to this invocation")
	mustHaveViolation(t, vs, "add the race-mode invocation as `go test -race -parallel 8 ...`, or drop the file from the targets")
}
