package stresstest

import (
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// reachability_isolation_classify_test.go pins
// classifyReachabilityIsolation — the pure decision logic behind
// ReachabilityIsolationScenario (M-0241/AC-5) — against fabricated
// envelopes for each of the six real `aiwf` calls the scenario makes.
// showFoundBeforeMerge is a plain bool rather than an envelope: `aiwf
// show`'s not-found path doesn't honor --format=json (G-0389), so
// the scenario classifies it by exit status alone.

func envWithFindings(findings ...verbEnvelopeFinding) verbEnvelope {
	e := verbEnvelope{Status: "findings", Findings: findings}
	e.Metadata.Entities = 0
	return e
}

func TestClassifyReachabilityIsolation(t *testing.T) {
	t.Parallel()

	baseline := envWithFindings(verbEnvelopeFinding{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "warning"})
	afterUnchanged := baseline
	afterChanged := envWithFindings(verbEnvelopeFinding{Code: "some-other-code", Severity: "warning"})
	// Same Findings as baseline, but a different entity count — isolates
	// the Metadata.Entities half of the outcome-changed check from the
	// Findings half, so a mutation that drops only the entities
	// comparison can't hide behind the findings comparison instead.
	afterSameFindingsDifferentEntityCount := envWithFindings(verbEnvelopeFinding{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "warning"})
	afterSameFindingsDifferentEntityCount.Metadata.Entities = 1

	historyEmpty := verbEnvelope{Status: "ok"}
	historyEmpty.Metadata.Events = 0
	historyWithEvents := verbEnvelope{Status: "ok"}
	historyWithEvents.Metadata.Events = 1
	historyErrored := verbEnvelope{Status: "error"}

	postMergeMoreEntities := verbEnvelope{Status: "findings"}
	postMergeMoreEntities.Metadata.Entities = 1
	postMergeSameEntities := verbEnvelope{Status: "findings"}
	postMergeSameEntities.Metadata.Entities = 0
	// One entity higher than afterSameFindingsDifferentEntityCount's 1,
	// so that row's own postMerge check (Entities > after's) stays
	// satisfied and doesn't add a second, unrelated violation.
	postMergeMoreEntitiesThanAlteredAfter := verbEnvelope{Status: "findings"}
	postMergeMoreEntitiesThanAlteredAfter.Metadata.Entities = 2

	tests := []struct {
		name                 string
		after                verbEnvelope
		showFoundBeforeMerge bool
		history              verbEnvelope
		postMerge            verbEnvelope
		postMergeHist        verbEnvelope
		wantViolations       int
	}{
		{
			name:                 "everything as expected: no violations",
			after:                afterUnchanged,
			showFoundBeforeMerge: false,
			history:              historyEmpty,
			postMerge:            postMergeMoreEntities,
			postMergeHist:        historyWithEvents,
			wantViolations:       0,
		},
		{
			name:                 "check's outcome changed from an invisible sibling commit — a violation",
			after:                afterChanged,
			showFoundBeforeMerge: false,
			history:              historyEmpty,
			postMerge:            postMergeMoreEntities,
			postMergeHist:        historyWithEvents,
			wantViolations:       1,
		},
		{
			name:                 "show found the sibling's entity before any merge — a violation",
			after:                afterUnchanged,
			showFoundBeforeMerge: true,
			history:              historyEmpty,
			postMerge:            postMergeMoreEntities,
			postMergeHist:        historyWithEvents,
			wantViolations:       1,
		},
		{
			name:                 "check's entity count changed even though findings matched baseline — a violation",
			after:                afterSameFindingsDifferentEntityCount,
			showFoundBeforeMerge: false,
			history:              historyEmpty,
			postMerge:            postMergeMoreEntitiesThanAlteredAfter,
			postMergeHist:        historyWithEvents,
			wantViolations:       1,
		},
		{
			name:                 "history errored instead of returning an empty result — a violation",
			after:                afterUnchanged,
			showFoundBeforeMerge: false,
			history:              historyErrored,
			postMerge:            postMergeMoreEntities,
			postMergeHist:        historyWithEvents,
			wantViolations:       1,
		},
		{
			name:                 "history leaked events for an unreachable commit — a violation",
			after:                afterUnchanged,
			showFoundBeforeMerge: false,
			history:              historyWithEvents,
			postMerge:            postMergeMoreEntities,
			postMergeHist:        historyWithEvents,
			wantViolations:       1,
		},
		{
			name:                 "merge did not actually expose the sibling's entity — a violation",
			after:                afterUnchanged,
			showFoundBeforeMerge: false,
			history:              historyEmpty,
			postMerge:            postMergeSameEntities,
			postMergeHist:        historyWithEvents,
			wantViolations:       1,
		},
		{
			name:                 "history still shows no events after merge — a violation",
			after:                afterUnchanged,
			showFoundBeforeMerge: false,
			history:              historyEmpty,
			postMerge:            postMergeMoreEntities,
			postMergeHist:        historyEmpty,
			wantViolations:       1,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			violations := classifyReachabilityIsolation(baseline, tc.after, tc.showFoundBeforeMerge, tc.history, tc.postMerge, tc.postMergeHist)
			if len(violations) != tc.wantViolations {
				t.Errorf("violations = %d (%+v), want %d", len(violations), violations, tc.wantViolations)
			}
		})
	}
}
