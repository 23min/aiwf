package stresstest

import "testing"

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
