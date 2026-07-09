package stresstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// parallel_branch_reallocate_test.go — real-subprocess coverage for
// ParallelBranchReallocateScenario (M-0243/AC-1). The pure decision
// logic (classifyParallelBranchReallocate) is pinned exhaustively in
// parallel_branch_reallocate_classify_test.go against fabricated
// findings; this is the actual scenario, driving two real, cloned
// aiwf-add subprocesses through a real merge/push/reallocate sequence.

func TestParallelBranchReallocateScenario_RealBinary_ConfirmsCleanResolution(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	base := t.TempDir()

	s := NewParallelBranchReallocateScenario(bin, entity.KindGap)
	result, err := RunScenario(s, base)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if !result.Passed {
		t.Fatalf("parallel-branch-reallocate scenario found violations (dir preserved at %s):\n%+v", result.Dir, result.Violations)
	}
}

func TestParallelBranchReallocateScenario_RealBinary_ErrorsWhenBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	base := t.TempDir()

	s := NewParallelBranchReallocateScenario(filepath.Join(t.TempDir(), "no-such-aiwf-binary"), entity.KindGap)
	if _, err := RunScenario(s, base); err == nil {
		t.Fatal("expected RunScenario to propagate the launch-failure error")
	} else if !strings.Contains(err.Error(), "operator A add") {
		t.Fatalf("expected the failure to name the operator A add step, got: %v", err)
	}
}

// TestParallelBranchReallocateScenario_RealBinary_RunErrorsWhenOperatorAddNotOK
// pre-seeds a colliding entity file in operator B's clone before Run
// so its `aiwf add` refuses at error severity (an id already exists at
// that path, mirroring M-0241/AC-5's same pre-seed technique), pinning
// that Run surfaces a non-"ok" add status.
func TestParallelBranchReallocateScenario_RealBinary_RunErrorsWhenOperatorAddNotOK(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := t.TempDir()

	s := NewParallelBranchReallocateScenario(bin, entity.KindGap)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	opBGapsDir := filepath.Join(dir, "operator-b", "work", "gaps")
	if mkdirErr := os.MkdirAll(opBGapsDir, 0o755); mkdirErr != nil {
		t.Fatalf("mkdir colliding gap dir: %v", mkdirErr)
	}
	if writeErr := os.WriteFile(filepath.Join(opBGapsDir, "G-0001-collision.md"), []byte("not valid frontmatter\n"), 0o644); writeErr != nil {
		t.Fatalf("write colliding gap file: %v", writeErr)
	}

	if err := s.Run(dir); err == nil {
		t.Fatal("expected Run to surface operator B's add refusal")
	} else if !strings.Contains(err.Error(), "did not report ok") {
		t.Fatalf("expected the refusal to name the add step, got: %v", err)
	}
}
