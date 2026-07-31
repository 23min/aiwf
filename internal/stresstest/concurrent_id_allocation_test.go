package stresstest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// concurrent_id_allocation_test.go — real-subprocess coverage for
// ConcurrentIDAllocationScenario (M-0241/AC-2). The pure decision
// logic (classifyConcurrentIDAllocation) is pinned exhaustively in
// concurrent_id_allocation_classify_test.go against fabricated
// outcomes; these tests confirm real, concurrently-launched `aiwf
// add` subprocesses racing repolock actually produce distinct ids,
// repeated via M-0240's RunRepeated for statistical coverage.
//
// Untagged despite racing real subprocesses, because the lane is
// decided by oracle shape rather than by subject matter: the
// scenario's verdict rests on distinct ids, refusals that carry
// repolock's busy code, and at least one actor getting through, none
// of which shift with how loaded the machine is.

func TestConcurrentIDAllocationScenario_RealBinary_ErrorsWhenBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	dir := newVerbSequenceTestRepo(t)

	s := NewConcurrentIDAllocationScenario(filepath.Join(t.TempDir(), "no-such-aiwf-binary"), entity.KindGap, 3, 1)
	if err := s.Run(dir); err == nil {
		t.Fatal("expected Run to error when the aiwf binary path doesn't exist")
	} else if !strings.Contains(err.Error(), "running aiwf add") {
		t.Fatalf("expected the launch failure to name the add call, got: %v", err)
	}
}

// TestConcurrentIDAllocationScenario_RealBinary_NConcurrentActorsAllGetDistinctIDs
// is the AC-2 scenario itself: n real `aiwf add gap` subprocesses,
// launched close together via goroutines racing real OS process
// scheduling (no artificial delay), against one working copy.
func TestConcurrentIDAllocationScenario_RealBinary_NConcurrentActorsAllGetDistinctIDs(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	base := t.TempDir()

	const n = 8
	newScenario := func(seed int64) Scenario {
		return NewConcurrentIDAllocationScenario(bin, entity.KindGap, n, seed)
	}

	rw := newReportWriter(&countingWriter{})
	results, err := RunRepeated(newScenario, base, 3, seedSequence(1, 2, 3), rw, "", nil)
	if err != nil {
		t.Fatalf("RunRepeated: %v", err)
	}
	for i, r := range results {
		if !r.Passed {
			t.Fatalf("attempt %d found violations (dir preserved at %s):\n%+v", i, r.Dir, r.Violations)
		}
	}
}

// TestConcurrentIDAllocationScenario_RealBinary_DetectsAGenuineDivergence
// points Run at a stand-in `aiwf` that reports every `add` as ok while
// handing back one constant id — a real, subprocess-observable
// divergence that the duplicate-id branch must catch. The "all get
// distinct ids" test above cannot tell a correctly-wired Run from one
// that silently drops classifyConcurrentIDAllocation's result, because
// a healthy repo yields zero violations either way, which is the same
// vacuity gap TestConcurrentMoveScenario_RealBinary_DetectsAGenuineDivergence
// closes on the move side.
func TestConcurrentIDAllocationScenario_RealBinary_DetectsAGenuineDivergence(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	realBin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)

	broken := NewConcurrentIDAllocationScenario(writeFakeAiwfAdd(t, realBin), entity.KindGap, 3, 1)
	if err := broken.Run(dir); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Both violations matter, and the second is what pins Run's own
	// wiring: the stand-in reports three successes while committing
	// nothing, so a Run that failed to thread the real before/after
	// counts into the classifier would report only the collision.
	assertViolations(t, broken.Verify(dir), []string{
		"was allocated by 3 concurrent actors",
		"commit count 0 -> 0 after 3 successful adds, want exactly +3",
	})
}

// writeFakeAiwfAdd writes an executable shell script standing in for
// `aiwf`: every `add` reports ok carrying one constant entity id, and
// every other subcommand delegates to realBin — so the scenario's own
// post-run `aiwf check` still reads real, unchanged on-disk state.
func writeFakeAiwfAdd(t *testing.T, realBin string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "aiwf")
	script := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "add" ]; then
  echo '{"status":"ok","findings":[],"result":{},"metadata":{"entity_id":"G-0001"}}'
  exit 0
fi
exec %q "$@"
`, realBin)
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil { //nolint:gosec // an executable stand-in binary under the test's own temp dir
		t.Fatalf("writing fake aiwf: %v", err)
	}
	return path
}
