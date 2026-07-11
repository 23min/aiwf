package stresstest

import (
	"testing"
)

// verb_sequence_operations_test.go — M-0250/AC-2: pins the walker's
// operation table (walkOperationsFor / weightedPick) and the shared
// non-FSM step classifier (classifySimpleStep) against fabricated
// inputs, so every branch is exercised deterministically rather than
// hoping a random walk's seed happens to hit it. The structural
// assertion here — the table names move/archive/rename/retitle — is
// the acceptance criterion's own explicit requirement: a check against
// the table's shape, not a probabilistic "did a run happen to pick
// it" check.

func TestWalkOperationsFor_NamesAllFourExtensionOpsWithNonzeroWeight(t *testing.T) {
	t.Parallel()
	ops := walkOperationsFor(true)
	weights := map[string]int{}
	for _, op := range ops {
		weights[op.Name] = op.Weight
	}
	for _, want := range []string{"move", "archive", "rename", "retitle"} {
		w, ok := weights[want]
		if !ok {
			t.Errorf("walkOperationsFor(true) does not name %q", want)
			continue
		}
		if w <= 0 {
			t.Errorf("walkOperationsFor(true)[%q].Weight = %d, want > 0", want, w)
		}
	}
}

func TestWalkOperationsFor_MoveDisabledExcludesMove(t *testing.T) {
	t.Parallel()
	ops := walkOperationsFor(false)
	for _, op := range ops {
		if op.Name == moveOperationName {
			t.Fatalf("walkOperationsFor(false) unexpectedly includes %q", moveOperationName)
		}
	}
	// The base four (minus move) are still all present.
	weights := map[string]int{}
	for _, op := range ops {
		weights[op.Name] = op.Weight
	}
	for _, want := range []string{"promote", "archive", "rename", "retitle"} {
		if weights[want] <= 0 {
			t.Errorf("walkOperationsFor(false)[%q].Weight = %d, want > 0", want, weights[want])
		}
	}
}

func TestTotalWeight(t *testing.T) {
	t.Parallel()
	ops := []walkOperation{{Name: "a", Weight: 3}, {Name: "b", Weight: 5}, {Name: "c", Weight: 2}}
	if got := totalWeight(ops); got != 10 {
		t.Errorf("totalWeight = %d, want 10", got)
	}
}

// TestWeightedPick_EveryBoundaryResolvesToTheExpectedOperation
// exhaustively walks every draw 0..totalWeight-1 against a fixed
// three-entry table, pinning the cumulative-range boundary logic
// deterministically — proving every operation IS reachable at some
// draw, without depending on a random source's luck.
func TestWeightedPick_EveryBoundaryResolvesToTheExpectedOperation(t *testing.T) {
	t.Parallel()
	ops := []walkOperation{{Name: "promote", Weight: 6}, {Name: "rename", Weight: 1}, {Name: "retitle", Weight: 1}, {Name: "archive", Weight: 1}, {Name: "move", Weight: 1}}
	want := map[int]string{
		0: "promote", 1: "promote", 2: "promote", 3: "promote", 4: "promote", 5: "promote",
		6: "rename",
		7: "retitle",
		8: "archive",
		9: "move",
	}
	for r, name := range want {
		if got := weightedPick(ops, r); got != name {
			t.Errorf("weightedPick(ops, %d) = %q, want %q", r, got, name)
		}
	}
}

func TestClassifySimpleStep_OkIsNotAViolation(t *testing.T) {
	t.Parallel()
	violations := classifySimpleStep("M-0001: rename to \"x\"", verbEnvelope{Status: "ok"})
	if len(violations) != 0 {
		t.Errorf("violations = %+v, want none", violations)
	}
}

func TestClassifySimpleStep_RefusalIsAViolation(t *testing.T) {
	t.Parallel()
	violations := classifySimpleStep("M-0001: rename to \"x\"", verbEnvelope{Status: "error", Error: &verbEnvelopeError{Code: "some-code"}}) //enums:ignore deliberately fabricated non-code for the test, not a real finding
	if len(violations) != 1 {
		t.Fatalf("violations = %+v, want exactly 1", violations)
	}
}

func TestMoveState_TargetAndApplyMovedAlternate(t *testing.T) {
	t.Parallel()
	mv := &moveState{current: "E-0001", other: "E-0002"}
	if got := mv.target(); got != "E-0002" {
		t.Fatalf("target() = %q, want E-0002", got)
	}
	mv.applyMoved()
	if mv.current != "E-0002" || mv.other != "E-0001" {
		t.Fatalf("after applyMoved: current=%q other=%q, want current=E-0002 other=E-0001", mv.current, mv.other)
	}
	if got := mv.target(); got != "E-0001" {
		t.Fatalf("target() after applyMoved = %q, want E-0001", got)
	}
}
