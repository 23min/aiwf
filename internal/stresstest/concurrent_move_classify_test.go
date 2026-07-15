package stresstest

import (
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// concurrent_move_classify_test.go pins classifyConcurrentMove — the
// pure decision logic behind ConcurrentMoveScenario (M-0250/AC-4) —
// against fabricated actor outcomes, mirroring
// concurrent_id_allocation_classify_test.go's own shape: the
// violation branches (which never fire against a correctly-working
// repolock) are exercised deterministically rather than depending on
// repolock actually being broken.

func TestClassifyConcurrentMove(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		outcomes       []moveActorOutcome
		n              int
		targetEpic     string
		before, after  int
		wantViolations int
	}{
		{
			name: "all succeed and land under the target epic with exactly one commit each — no violations",
			outcomes: []moveActorOutcome{
				{milestoneID: "M-0001", status: "ok", parent: "E-0002"},
				{milestoneID: "M-0002", status: "ok", parent: "E-0002"},
				{milestoneID: "M-0003", status: "ok", parent: "E-0002"},
			},
			n: 3, targetEpic: "E-0002",
			before: 5, after: 8,
			wantViolations: 0,
		},
		{
			name: "an actor reports a non-ok status under contention — a violation",
			outcomes: []moveActorOutcome{
				{milestoneID: "M-0001", status: "ok", parent: "E-0002"},
				{milestoneID: "M-0002", status: "error", parent: ""},
				{milestoneID: "M-0003", status: "ok", parent: "E-0002"},
			},
			n: 3, targetEpic: "E-0002",
			before: 5, after: 7,
			wantViolations: 2, // the non-ok status itself, plus the resulting success-count shortfall
		},
		{
			name: "an actor reports ok but the milestone didn't actually land under the target epic — a violation",
			outcomes: []moveActorOutcome{
				{milestoneID: "M-0001", status: "ok", parent: "E-0002"},
				{milestoneID: "M-0002", status: "ok", parent: "E-0001"}, // stale/wrong parent
			},
			n: 2, targetEpic: "E-0002",
			before: 5, after: 7,
			wantViolations: 1,
		},
		{
			name: "all succeed but the commit count landed short — a violation",
			outcomes: []moveActorOutcome{
				{milestoneID: "M-0001", status: "ok", parent: "E-0002"},
				{milestoneID: "M-0002", status: "ok", parent: "E-0002"},
			},
			n: 2, targetEpic: "E-0002",
			before: 5, after: 6, // want 7 (5+2)
			wantViolations: 1,
		},
		{
			name:       "zero actors run — trivially zero violations, not a false success claim",
			outcomes:   nil,
			n:          0,
			targetEpic: "E-0002",
			before:     5, after: 5,
			wantViolations: 0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			violations := classifyConcurrentMove(tc.outcomes, tc.n, tc.targetEpic, tc.before, tc.after)
			if len(violations) != tc.wantViolations {
				t.Errorf("violations = %d (%+v), want %d", len(violations), violations, tc.wantViolations)
			}
		})
	}
}

// TestConcurrentMoveExpectedWarnings pins M-0257/AC-1's broadened
// check-clean baseline for this scenario.
func TestConcurrentMoveExpectedWarnings(t *testing.T) {
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
			got := classifyAgainstBaseline(tc.findings, concurrentMoveExpectedWarnings)
			if len(got) != tc.wantViolations {
				t.Fatalf("violations = %+v, want %d", got, tc.wantViolations)
			}
		})
	}
}
