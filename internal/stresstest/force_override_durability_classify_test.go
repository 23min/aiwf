package stresstest

import "testing"

// force_override_durability_classify_test.go pins
// classifyForceOverrideDurability — the pure decision logic behind
// ForceOverrideDurabilityScenario (M-0243/AC-4) — against fabricated
// outcomes, so every branch is exercised deterministically rather
// than depending on a real rebase/cherry-pick's exact behavior.
//
// Item 5 (ack revocation via rebase): a revived illegal-transition
// finding after the ack commit is dropped is itself the confirmed
// risk — a real audit-trail regression, treated as a violation.
//
// Item 6 (cherry-picked force-override carryover): the cherry-picked
// commit not being re-flagged on its new branch is the CURRENT,
// by-design trust model (aiwf-force + human actor is trusted
// wherever it appears) — there is no alternative correct behavior a
// mechanical check could assert instead, so this half only reports
// premise breaks, never the carryover itself.

func TestClassifyForceOverrideDurability(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		preAckFlagged     bool
		postAckFlagged    bool
		postRebaseFlagged bool
		forceAccepted     bool
		cherryPickClean   bool
		trailersPreserved bool
		wantViolations    int
	}{
		{
			name:              "confirmed: ack revoked by rebase; cherry-pick carryover confirmed cleanly",
			preAckFlagged:     true,
			postAckFlagged:    false,
			postRebaseFlagged: true,
			forceAccepted:     true,
			cherryPickClean:   true,
			trailersPreserved: true,
			wantViolations:    1,
		},
		{
			name:              "item 5 premise broken: the manual illegal edit was never flagged",
			preAckFlagged:     false,
			postAckFlagged:    false,
			postRebaseFlagged: true,
			forceAccepted:     true,
			cherryPickClean:   true,
			trailersPreserved: true,
			wantViolations:    2,
		},
		{
			name:              "item 5: acknowledge illegal did not suppress the finding",
			preAckFlagged:     true,
			postAckFlagged:    true,
			postRebaseFlagged: true,
			forceAccepted:     true,
			cherryPickClean:   true,
			trailersPreserved: true,
			wantViolations:    2,
		},
		{
			name:              "item 5: the ack unexpectedly survived the rebase (no revival) — not itself a violation",
			preAckFlagged:     true,
			postAckFlagged:    false,
			postRebaseFlagged: false,
			forceAccepted:     true,
			cherryPickClean:   true,
			trailersPreserved: true,
			wantViolations:    0,
		},
		{
			name:              "item 6 premise broken: the original force-promote was not accepted",
			preAckFlagged:     true,
			postAckFlagged:    false,
			postRebaseFlagged: true,
			forceAccepted:     false,
			cherryPickClean:   true,
			trailersPreserved: true,
			wantViolations:    2,
		},
		{
			name:              "item 6 premise broken: the cherry-pick produced a conflict",
			preAckFlagged:     true,
			postAckFlagged:    false,
			postRebaseFlagged: true,
			forceAccepted:     true,
			cherryPickClean:   false,
			trailersPreserved: true,
			wantViolations:    2,
		},
		{
			name:              "item 6 premise broken: the cherry-pick did not preserve the force/actor trailers",
			preAckFlagged:     true,
			postAckFlagged:    false,
			postRebaseFlagged: true,
			forceAccepted:     true,
			cherryPickClean:   true,
			trailersPreserved: false,
			wantViolations:    2,
		},
		{
			name:              "every check fails at once",
			preAckFlagged:     false,
			postAckFlagged:    true,
			postRebaseFlagged: false,
			forceAccepted:     false,
			cherryPickClean:   false,
			trailersPreserved: false,
			wantViolations:    5,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyForceOverrideDurability(tc.preAckFlagged, tc.postAckFlagged, tc.postRebaseFlagged, tc.forceAccepted, tc.cherryPickClean, tc.trailersPreserved)
			if len(got) != tc.wantViolations {
				t.Errorf("violations = %d (%+v), want %d", len(got), got, tc.wantViolations)
			}
		})
	}
}
