package stresstest

import (
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
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
		name          string
		outcomes      []actorOutcome
		n             int
		before, after int
		// wantSubstrings names each expected violation by a fragment of
		// its message, and its length is the expected count. Asserting
		// the count alone would pass a classifier that emitted the right
		// number of the wrong violations — and the message is the whole
		// of what the harness reports, so it is the product here, not a
		// presentational detail.
		wantSubstrings []string
	}{
		{
			name: "all succeed with distinct ids — no violations",
			outcomes: []actorOutcome{
				{status: "ok", entityID: "G-0001"},
				{status: "ok", entityID: "G-0002"},
				{status: "ok", entityID: "G-0003"},
			},
			n:      3,
			before: 0, after: 3,
		},
		{
			name: "two actors allocate the same id — a violation",
			outcomes: []actorOutcome{
				{status: "ok", entityID: "G-0001"},
				{status: "ok", entityID: "G-0001"},
				{status: "ok", entityID: "G-0002"},
			},
			n:              3,
			wantSubstrings: []string{"id G-0001 was allocated by 2 concurrent actors"},
			before:         0, after: 3,
		},
		{
			name: "three actors allocate the same id — still exactly one violation (aggregate, not per-pair)",
			outcomes: []actorOutcome{
				{status: "ok", entityID: "G-0001"},
				{status: "ok", entityID: "G-0001"},
				{status: "ok", entityID: "G-0001"},
			},
			n:              3,
			wantSubstrings: []string{"id G-0001 was allocated by 3 concurrent actors"},
			before:         0, after: 3,
		},
		{
			name: "two separate ids are each duplicated — one violation apiece, not one overall",
			outcomes: []actorOutcome{
				{status: "ok", entityID: "G-0001"},
				{status: "ok", entityID: "G-0001"},
				{status: "ok", entityID: "G-0002"},
				{status: "ok", entityID: "G-0002"},
			},
			n: 4,
			wantSubstrings: []string{
				"id G-0001 was allocated by 2 concurrent actors",
				"id G-0002 was allocated by 2 concurrent actors",
			},
			before: 0, after: 4,
		},
		{
			name: "an actor fails for a reason other than contention — a violation",
			outcomes: []actorOutcome{
				{status: "ok", entityID: "G-0001"},
				{status: "error", entityID: ""},
				{status: "ok", entityID: "G-0002"},
			},
			n:              3,
			wantSubstrings: []string{`actor 1: aiwf add failed for a reason other than repolock contention (status=error, code="")`},
			before:         0, after: 2,
		},
		{
			name: "an actor is refused because another held the lock — not a violation",
			outcomes: []actorOutcome{
				{status: "ok", entityID: "G-0001"},
				{status: "error", errorCode: cliutil.CodeRepoLockBusy},
				{status: "ok", entityID: "G-0002"},
			},
			n:      3,
			before: 0, after: 2,
		},
		{
			name: "every actor but one is refused — still not a violation, however lopsided",
			outcomes: []actorOutcome{
				{status: "ok", entityID: "G-0001"},
				{status: "error", errorCode: cliutil.CodeRepoLockBusy},
				{status: "error", errorCode: cliutil.CodeRepoLockBusy},
			},
			n:      3,
			before: 0, after: 1,
		},
		{
			name: "no actor at all succeeds — a deadlock, which contention never explains",
			outcomes: []actorOutcome{
				{status: "error", errorCode: cliutil.CodeRepoLockBusy},
				{status: "error", errorCode: cliutil.CodeRepoLockBusy},
			},
			n:              2,
			wantSubstrings: []string{"none of 2 concurrent actors succeeded"},
			before:         0, after: 0,
		},
		{
			name: "the lock could not be taken at all — not the busy refusal, so a violation",
			outcomes: []actorOutcome{
				{status: "ok", entityID: "G-0001"},
				{status: "error", errorCode: cliutil.CodeRepoLockAcquireFailed},
			},
			n:              2,
			wantSubstrings: []string{`code="repo-lock-acquire-failed"`},
			before:         0, after: 1,
		},
		{
			name: "a refused actor that nonetheless committed — a violation",
			outcomes: []actorOutcome{
				{status: "ok", entityID: "G-0001"},
				{status: "error", errorCode: cliutil.CodeRepoLockBusy},
			},
			n:              2,
			wantSubstrings: []string{"commit count 0 -> 2 after 1 successful adds, want exactly +1"},
			before:         0, after: 2,
		},
		{
			name: "refused actors still cannot mask a duplicate id among those that succeeded",
			outcomes: []actorOutcome{
				{status: "ok", entityID: "G-0001"},
				{status: "ok", entityID: "G-0001"},
				{status: "error", errorCode: cliutil.CodeRepoLockBusy},
			},
			n:              3,
			wantSubstrings: []string{"id G-0001 was allocated by 2 concurrent actors"},
			before:         0, after: 2,
		},
		{
			name:     "zero actors run — trivially zero violations, not a false success claim",
			outcomes: nil,
			n:        0,
			before:   0, after: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertViolations(t, classifyConcurrentIDAllocation(tc.outcomes, tc.n, tc.before, tc.after), tc.wantSubstrings)
		})
	}
}

// TestNewActorOutcome pins the reduction from envelope to the fields
// the classifier judges. The classifier's own table is built from
// fabricated outcomes, so it cannot see whether Run populates them —
// and an error code dropped on the way in makes every refusal look
// alike, which is exactly the conflation this scenario's oracle
// exists to avoid.
func TestNewActorOutcome(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		env  verbEnvelope
		want actorOutcome
	}{
		{
			name: "a successful add carries its allocated id and no error code",
			env: func() verbEnvelope {
				var e verbEnvelope
				e.Status = "ok"
				e.Metadata.EntityID = "G-0001"
				return e
			}(),
			want: actorOutcome{status: "ok", entityID: "G-0001"},
		},
		{
			name: "a busy refusal carries the code the classifier matches on",
			env: verbEnvelope{
				Status: "error",
				Error:  &verbEnvelopeError{Code: cliutil.CodeRepoLockBusy, Message: "another aiwf process holds the lock"},
			},
			want: actorOutcome{status: "error", errorCode: cliutil.CodeRepoLockBusy},
		},
		{
			name: "an error envelope with no code at all reduces to an empty code, not a busy refusal",
			env:  verbEnvelope{Status: "error"},
			want: actorOutcome{status: "error"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := newActorOutcome(tc.env); got != tc.want {
				t.Errorf("newActorOutcome() = %+v, want %+v", got, tc.want)
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
