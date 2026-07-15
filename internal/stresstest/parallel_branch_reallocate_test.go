package stresstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
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

// TestParallelBranchReallocateScenario_BroadenedOracleCatchesAnInjectedRegression
// is M-0257/AC-3's synthetic regression test: it drives the real
// scenario to completion (mirroring
// TestParallelBranchReallocateScenario_RealBinary_ConfirmsCleanResolution's
// own clean-run premise), captures the real post-reallocate check
// envelope, and injects one extraneous finding a genuine check-rule
// regression might produce — a code outside both this scenario's
// baseline and its existing ids-unique-only assertion. It then
// confirms (a) classifyParallelBranchReallocate alone, exactly as this
// scenario already calls it, does NOT flag the injected finding (the
// blind spot G-0410 named), and (b) M-0257/AC-1's broadened
// classifyAgainstBaseline call DOES — proving the fix actually closes
// the gap, not just that new code runs.
func TestParallelBranchReallocateScenario_BroadenedOracleCatchesAnInjectedRegression(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	base := t.TempDir()

	s := NewParallelBranchReallocateScenario(bin, entity.KindGap)
	dir, mkdirErr := os.MkdirTemp(base, "regression-")
	if mkdirErr != nil {
		t.Fatalf("MkdirTemp: %v", mkdirErr)
	}
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if err := s.Run(dir); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if violations := s.Verify(dir); len(violations) != 0 {
		t.Fatalf("expected the real scenario run to be check-clean before injecting a regression, got: %+v", violations)
	}

	opB := filepath.Join(dir, "operator-b")
	postCheckEnv, err := runAiwfJSON(bin, opB, "check")
	if err != nil {
		t.Fatalf("capturing the real post-reallocate check envelope: %v", err)
	}
	injected := append(append([]verbEnvelopeFinding(nil), postCheckEnv.Findings...),
		verbEnvelopeFinding{Code: "synthetic-check-rule-regression", Severity: "warning"}) //enums:ignore deliberately fabricated non-code simulating an unrelated check-rule regression, not a real finding

	// A real, well-formed pre-reallocate checkFindings argument (this
	// scenario's own premise: the collision WAS surfaced as
	// ids-unique) isolates the assertion to what's under test — whether
	// classifyParallelBranchReallocate's postCheckFindings handling
	// alone catches the injected code, not an unrelated premise break.
	realCheckFindings := []verbEnvelopeFinding{{Code: check.CodeIDsUnique, Severity: "error"}}
	if got := classifyParallelBranchReallocate(realCheckFindings, "ok", injected, true); len(got) != 0 {
		t.Fatalf("expected the scenario's existing single-finding-code assertion NOT to catch the injected finding on its own, got: %+v", got)
	}

	if got := classifyAgainstBaseline(injected, parallelBranchReallocateExpectedWarnings); len(got) != 1 {
		t.Fatalf("expected the broadened check-clean oracle to flag exactly one violation for the injected finding, got: %+v", got)
	}
}
