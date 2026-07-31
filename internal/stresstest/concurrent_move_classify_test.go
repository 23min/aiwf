package stresstest

import (
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
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
		name          string
		outcomes      []moveActorOutcome
		n             int
		targetEpic    string
		before, after int
		// wantSubstrings names each expected violation by a fragment of
		// its message; see assertViolations for why the count alone is
		// not enough.
		wantSubstrings []string
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
		},
		{
			name: "an actor fails for a reason other than contention — a violation",
			outcomes: []moveActorOutcome{
				{milestoneID: "M-0001", status: "ok", parent: "E-0002"},
				{milestoneID: "M-0002", status: "error", parent: ""},
				{milestoneID: "M-0003", status: "ok", parent: "E-0002"},
			},
			n: 3, targetEpic: "E-0002",
			before: 5, after: 7,
			wantSubstrings: []string{`M-0002: aiwf move failed for a reason other than repolock contention (status=error, code="")`},
		},
		{
			name: "the lock could not be taken at all — not the busy refusal, so a violation",
			outcomes: []moveActorOutcome{
				{milestoneID: "M-0001", status: "ok", parent: "E-0002"},
				{milestoneID: "M-0002", status: "error", errorCode: cliutil.CodeRepoLockAcquireFailed},
			},
			n: 2, targetEpic: "E-0002",
			before: 5, after: 6,
			wantSubstrings: []string{`code="repo-lock-acquire-failed"`},
		},
		{
			name: "an actor is refused because another held the lock — not a violation, and it commits nothing",
			outcomes: []moveActorOutcome{
				{milestoneID: "M-0001", status: "ok", parent: "E-0002"},
				{milestoneID: "M-0002", status: "error", errorCode: cliutil.CodeRepoLockBusy},
				{milestoneID: "M-0003", status: "ok", parent: "E-0002"},
			},
			n: 3, targetEpic: "E-0002",
			before: 5, after: 7,
		},
		{
			name: "a refused actor that nonetheless moved the commit count — a violation",
			outcomes: []moveActorOutcome{
				{milestoneID: "M-0001", status: "ok", parent: "E-0002"},
				{milestoneID: "M-0002", status: "error", errorCode: cliutil.CodeRepoLockBusy},
			},
			n: 2, targetEpic: "E-0002",
			before: 5, after: 7, // want 6 (5+1): the refused actor must have committed nothing
			wantSubstrings: []string{"commit count 5 -> 7 after 1 successful moves, want exactly +1"},
		},
		{
			name: "no actor at all succeeds — a deadlock, and commits appearing anyway is a second violation",
			outcomes: []moveActorOutcome{
				{milestoneID: "M-0001", status: "error", errorCode: cliutil.CodeRepoLockBusy},
				{milestoneID: "M-0002", status: "error", errorCode: cliutil.CodeRepoLockBusy},
			},
			n: 2, targetEpic: "E-0002",
			before: 5, after: 6,
			wantSubstrings: []string{
				"none of 2 concurrent move actors succeeded",
				"commit count 5 -> 6 after 0 successful moves, want exactly +0",
			},
		},
		{
			name: "an actor reports ok but the milestone didn't actually land under the target epic — a violation",
			outcomes: []moveActorOutcome{
				{milestoneID: "M-0001", status: "ok", parent: "E-0002"},
				{milestoneID: "M-0002", status: "ok", parent: "E-0001"}, // stale/wrong parent
			},
			n: 2, targetEpic: "E-0002",
			before: 5, after: 7,
			wantSubstrings: []string{`M-0002: move reported ok but final parent is "E-0001", want "E-0002"`},
		},
		{
			name: "all succeed but the commit count landed short — a violation",
			outcomes: []moveActorOutcome{
				{milestoneID: "M-0001", status: "ok", parent: "E-0002"},
				{milestoneID: "M-0002", status: "ok", parent: "E-0002"},
			},
			n: 2, targetEpic: "E-0002",
			before: 5, after: 6, // want 7 (5+2)
			wantSubstrings: []string{"commit count 5 -> 6 after 2 successful moves, want exactly +2"},
		},
		{
			name:       "zero actors run — trivially zero violations, not a false success claim",
			outcomes:   nil,
			n:          0,
			targetEpic: "E-0002",
			before:     5, after: 5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertViolations(t, classifyConcurrentMove(tc.outcomes, tc.n, tc.targetEpic, tc.before, tc.after), tc.wantSubstrings)
		})
	}
}

// TestNewMoveActorOutcome pins the reduction from envelope to the
// fields the classifier judges, for the reason TestNewActorOutcome
// records.
func TestNewMoveActorOutcome(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		milestoneID string
		env         verbEnvelope
		parent      string
		want        moveActorOutcome
	}{
		{
			name:        "a successful move carries the parent Run resolved and no error code",
			milestoneID: "M-0001",
			env:         verbEnvelope{Status: "ok"},
			parent:      "E-0002",
			want:        moveActorOutcome{milestoneID: "M-0001", status: "ok", parent: "E-0002"},
		},
		{
			name:        "a busy refusal carries the code the classifier matches on",
			milestoneID: "M-0002",
			env: verbEnvelope{
				Status: "error",
				Error:  &verbEnvelopeError{Code: cliutil.CodeRepoLockBusy, Message: "another aiwf process holds the lock"},
			},
			want: moveActorOutcome{milestoneID: "M-0002", status: "error", errorCode: cliutil.CodeRepoLockBusy},
		},
		{
			name:        "an error envelope with no code at all reduces to an empty code, not a busy refusal",
			milestoneID: "M-0003",
			env:         verbEnvelope{Status: "error"},
			want:        moveActorOutcome{milestoneID: "M-0003", status: "error"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := newMoveActorOutcome(tc.milestoneID, tc.env, tc.parent); got != tc.want {
				t.Errorf("newMoveActorOutcome() = %+v, want %+v", got, tc.want)
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
