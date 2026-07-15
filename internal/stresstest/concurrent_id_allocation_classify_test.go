package stresstest

import (
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// concurrent_id_allocation_classify_test.go pins
// classifyConcurrentIDAllocation — the pure decision logic behind
// ConcurrentIDAllocationScenario (M-0241/AC-2) — against fabricated
// actor outcomes, so the duplicate-id branch (which never fires
// against a correctly-working repolock) is exercised deterministically
// rather than depending on repolock actually being broken.

func TestClassifyConcurrentIDAllocation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		outcomes       []actorOutcome
		n              int
		wantViolations int
	}{
		{
			name: "all succeed with distinct ids — no violations",
			outcomes: []actorOutcome{
				{status: "ok", entityID: "G-0001"},
				{status: "ok", entityID: "G-0002"},
				{status: "ok", entityID: "G-0003"},
			},
			n:              3,
			wantViolations: 0,
		},
		{
			name: "two actors allocate the same id — a violation",
			outcomes: []actorOutcome{
				{status: "ok", entityID: "G-0001"},
				{status: "ok", entityID: "G-0001"},
				{status: "ok", entityID: "G-0002"},
			},
			n:              3,
			wantViolations: 1,
		},
		{
			name: "three actors allocate the same id — still exactly one violation (aggregate, not per-pair)",
			outcomes: []actorOutcome{
				{status: "ok", entityID: "G-0001"},
				{status: "ok", entityID: "G-0001"},
				{status: "ok", entityID: "G-0001"},
			},
			n:              3,
			wantViolations: 1,
		},
		{
			name: "an actor reports a non-ok status under contention — a violation",
			outcomes: []actorOutcome{
				{status: "ok", entityID: "G-0001"},
				{status: "error", entityID: ""},
				{status: "ok", entityID: "G-0002"},
			},
			n:              3,
			wantViolations: 2, // the non-ok status itself, plus the resulting success-count shortfall
		},
		{
			name:           "zero actors run — trivially zero violations, not a false success claim",
			outcomes:       nil,
			n:              0,
			wantViolations: 0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			violations := classifyConcurrentIDAllocation(tc.outcomes, tc.n)
			if len(violations) != tc.wantViolations {
				t.Errorf("violations = %d (%+v), want %d", len(violations), violations, tc.wantViolations)
			}
		})
	}
}

// TestConcurrentIDAllocationExpectedWarnings pins M-0257/AC-1's
// broadened check-clean baseline for this scenario.
func TestConcurrentIDAllocationExpectedWarnings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		findings       []verbEnvelopeFinding
		wantViolations int
	}{
		{name: "no findings", findings: nil, wantViolations: 0},
		{
			name:           "the baseline provenance-scope-undefined warning is accepted",
			findings:       []verbEnvelopeFinding{{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "warning"}},
			wantViolations: 0,
		},
		{
			name:           "an unbaselined warning code is a violation",
			findings:       []verbEnvelopeFinding{{Code: "some-unexpected-code", Severity: "warning"}}, //enums:ignore deliberately fabricated non-code for the test, not a real finding
			wantViolations: 1,
		},
		{
			name:           "an error-severity finding is a violation even for a baselined code",
			findings:       []verbEnvelopeFinding{{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "error"}},
			wantViolations: 1,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyAgainstBaseline(tc.findings, concurrentIDAllocationExpectedWarnings)
			if len(got) != tc.wantViolations {
				t.Fatalf("violations = %+v, want %d", got, tc.wantViolations)
			}
		})
	}
}
