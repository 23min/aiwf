package stresstest

import (
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
