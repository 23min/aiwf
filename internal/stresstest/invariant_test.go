package stresstest

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// stubInvariant is a fixed answer standing in for a real property, so
// the seam's aggregation and error-forwarding are testable without
// driving subprocesses.
type stubInvariant struct {
	name       string
	violations []Violation
	err        error
	sawLabel   *string
}

func (s stubInvariant) Name() string { return s.name }

func (s stubInvariant) Evaluate(_, _, label string) ([]Violation, error) {
	if s.sawLabel != nil {
		*s.sawLabel = label
	}
	return s.violations, s.err
}

func TestEvaluateInvariants_CollectsEveryPropertysViolationsInRegistrationOrder(t *testing.T) {
	t.Parallel()

	got, err := evaluateInvariants([]Invariant{
		stubInvariant{name: "first", violations: []Violation{{Message: "a"}}},
		stubInvariant{name: "second", violations: []Violation{{Message: "b"}, {Message: "c"}}},
		stubInvariant{name: "silent"},
	}, "bin", "dir", "step 1")
	if err != nil {
		t.Fatalf("evaluateInvariants: %v", err)
	}

	want := []Violation{{Message: "a"}, {Message: "b"}, {Message: "c"}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("evaluateInvariants() mismatch (-want +got):\n%s", diff)
	}
}

func TestEvaluateInvariants_PassesTheStepLabelThrough(t *testing.T) {
	t.Parallel()

	var seen string
	if _, err := evaluateInvariants([]Invariant{
		stubInvariant{name: "first", sawLabel: &seen},
	}, "bin", "dir", "M-0001 step 3 (promote)"); err != nil {
		t.Fatalf("evaluateInvariants: %v", err)
	}
	if seen != "M-0001 step 3 (promote)" {
		t.Errorf("invariant saw label %q, want the step label", seen)
	}
}

func TestEvaluateInvariants_WrapsAnEvaluationErrorWithThePropertyAndStep(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("subprocess would not launch")
	got, err := evaluateInvariants([]Invariant{
		stubInvariant{name: "first", violations: []Violation{{Message: "a"}}},
		stubInvariant{name: "read-path agreement", err: sentinel},
	}, "bin", "dir", "step 4")

	if !errors.Is(err, sentinel) {
		t.Fatalf("evaluateInvariants() err = %v, want it to wrap the evaluation error", err)
	}
	for _, want := range []string{"read-path agreement", "step 4"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err.Error(), want)
		}
	}
	// An unevaluable property is a harness fault, not a verdict: the
	// partial violations must not be reported as if the run completed.
	if got != nil {
		t.Errorf("evaluateInvariants() returned %+v alongside an error, want nil", got)
	}
}

func TestEvaluateInvariants_NoRegisteredPropertiesFindsNothing(t *testing.T) {
	t.Parallel()

	got, err := evaluateInvariants(nil, "bin", "dir", "step 5")
	if err != nil {
		t.Fatalf("evaluateInvariants: %v", err)
	}
	if got != nil {
		t.Errorf("evaluateInvariants() = %+v, want nil", got)
	}
}

// TestWalkInvariants_JudgesEveryStateAgainstBothProperties pins the
// registry itself. An oracle that reports coverage it does not have is
// this milestone's named risk, and a property silently dropped from the
// registry is exactly that: every scenario keeps passing.
func TestWalkInvariants_JudgesEveryStateAgainstBothProperties(t *testing.T) {
	t.Parallel()

	var got []string
	for _, inv := range walkInvariants() {
		got = append(got, inv.Name())
	}

	want := []string{"list-vs-ground-truth", "read-path agreement"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("walkInvariants() mismatch (-want +got):\n%s", diff)
	}
}

// TestNewVerbSequenceScenario_RegistersTheWalkInvariants closes the same
// vacuity gap at the constructor: the walk evaluates whatever the
// scenario carries, so a scenario built with an empty set judges nothing
// and passes everything.
func TestNewVerbSequenceScenario_RegistersTheWalkInvariants(t *testing.T) {
	t.Parallel()

	s := NewVerbSequenceScenario("bin", 1, 6)
	if len(s.invariants) != len(walkInvariants()) {
		t.Fatalf("scenario carries %d invariants, want %d", len(s.invariants), len(walkInvariants()))
	}
}

func TestListGroundTruthInvariant_ReportsWhatCheckListInvariantFinds(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	realBin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)

	if _, err := runAiwfJSON(realBin, dir, "add", "epic", "--title", "epic a", "--body", "b"); err != nil {
		t.Fatalf("add epic: %v", err)
	}

	inv := listGroundTruthInvariant{}
	if inv.Name() != "list-vs-ground-truth" {
		t.Errorf("Name() = %q", inv.Name())
	}
	if violations, err := inv.Evaluate(realBin, dir, "label"); err != nil || len(violations) != 0 {
		t.Fatalf("Evaluate() = %+v, %v; want no violations against the real binary", violations, err)
	}

	fake := writeFakeAiwfList(t, `{"status":"ok","findings":[],"result":[]}`)
	violations, err := inv.Evaluate(fake, dir, "label")
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("Evaluate() = %+v, want the divergence the stand-in manufactures", violations)
	}
}
