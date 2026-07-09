package stresstest

import (
	"strings"
	"testing"
)

// force_override_durability_classify_test.go pins
// classifyForceOverrideDurability — the pure decision logic behind
// ForceOverrideDurabilityScenario (M-0243/AC-4; updated by
// M-0244/AC-2's G-0395 resolution) — against fabricated outcomes, so
// every branch is exercised deterministically rather than depending
// on a real rebase/cherry-pick's exact behavior.
//
// Item 5 (ack revocation via rebase): per D-0034, a revived
// illegal-transition finding after the ack commit is dropped is a
// confirmed, expected property (the pre-push gate — not this finding
// staying suppressed forever — is what actually prevents the
// corrupted history from being shared), so it is asserted but not
// itself a violation. G-0395's dangling-ack diagnostic firing when
// the revival happens IS required; its absence is the violation.
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
		name                  string
		preAckFlagged         bool
		postAckFlagged        bool
		postRebaseFlagged     bool
		postRebaseHintPresent bool
		forceAccepted         bool
		cherryPickClean       bool
		trailersPreserved     bool
		wantSubstrings        []string // nil means no violations expected
	}{
		{
			name:                  "clean: ack revoked by rebase, diagnostic hint fires, cherry-pick carryover confirmed",
			preAckFlagged:         true,
			postAckFlagged:        false,
			postRebaseFlagged:     true,
			postRebaseHintPresent: true,
			forceAccepted:         true,
			cherryPickClean:       true,
			trailersPreserved:     true,
			wantSubstrings:        nil,
		},
		{
			name:                  "item 5 premise broken: the manual illegal edit was never flagged",
			preAckFlagged:         false,
			postAckFlagged:        false,
			postRebaseFlagged:     true,
			postRebaseHintPresent: true,
			forceAccepted:         true,
			cherryPickClean:       true,
			trailersPreserved:     true,
			wantSubstrings:        []string{"was never flagged before acknowledging"},
		},
		{
			name:                  "item 5: acknowledge illegal did not suppress the finding",
			preAckFlagged:         true,
			postAckFlagged:        true,
			postRebaseFlagged:     true,
			postRebaseHintPresent: true,
			forceAccepted:         true,
			cherryPickClean:       true,
			trailersPreserved:     true,
			wantSubstrings:        []string{"did not suppress the illegal-transition"},
		},
		{
			name:                  "item 5: the ack unexpectedly survived the rebase (no revival) — not itself a violation",
			preAckFlagged:         true,
			postAckFlagged:        false,
			postRebaseFlagged:     false,
			postRebaseHintPresent: false,
			forceAccepted:         true,
			cherryPickClean:       true,
			trailersPreserved:     true,
			wantSubstrings:        nil,
		},
		{
			name:                  "item 5: revival happened but the dangling-ack diagnostic did not fire",
			preAckFlagged:         true,
			postAckFlagged:        false,
			postRebaseFlagged:     true,
			postRebaseHintPresent: false,
			forceAccepted:         true,
			cherryPickClean:       true,
			trailersPreserved:     true,
			wantSubstrings:        []string{"diagnostic did not name the dropped acknowledgment"},
		},
		{
			name:                  "item 6 premise broken: the original force-promote was not accepted",
			preAckFlagged:         true,
			postAckFlagged:        false,
			postRebaseFlagged:     true,
			postRebaseHintPresent: true,
			forceAccepted:         false,
			cherryPickClean:       true,
			trailersPreserved:     true,
			wantSubstrings:        []string{"was not accepted"},
		},
		{
			name:                  "item 6 premise broken: the cherry-pick produced a conflict",
			preAckFlagged:         true,
			postAckFlagged:        false,
			postRebaseFlagged:     true,
			postRebaseHintPresent: true,
			forceAccepted:         true,
			cherryPickClean:       false,
			trailersPreserved:     true,
			wantSubstrings:        []string{"produced an unexpected conflict"},
		},
		{
			name:                  "item 6 premise broken: the cherry-pick did not preserve the force/actor trailers",
			preAckFlagged:         true,
			postAckFlagged:        false,
			postRebaseFlagged:     true,
			postRebaseHintPresent: true,
			forceAccepted:         true,
			cherryPickClean:       true,
			trailersPreserved:     false,
			wantSubstrings:        []string{"trailer-preservation did not hold"},
		},
		{
			name:                  "every check fails at once",
			preAckFlagged:         false,
			postAckFlagged:        true,
			postRebaseFlagged:     true,
			postRebaseHintPresent: false,
			forceAccepted:         false,
			cherryPickClean:       false,
			trailersPreserved:     false,
			wantSubstrings: []string{
				"was never flagged before acknowledging",
				"did not suppress the illegal-transition",
				"diagnostic did not name the dropped acknowledgment",
				"was not accepted",
				"produced an unexpected conflict",
				"trailer-preservation did not hold",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyForceOverrideDurability(tc.preAckFlagged, tc.postAckFlagged, tc.postRebaseFlagged, tc.postRebaseHintPresent, tc.forceAccepted, tc.cherryPickClean, tc.trailersPreserved)
			if len(got) != len(tc.wantSubstrings) {
				t.Fatalf("violations = %+v, want %d matching %v", got, len(tc.wantSubstrings), tc.wantSubstrings)
			}
			for _, want := range tc.wantSubstrings {
				found := false
				for _, v := range got {
					if strings.Contains(v.Message, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("no violation contained %q; got %+v", want, got)
				}
			}
		})
	}
}
