package cliutil

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// TestRefuseNonHumanSovereignForce covers every arm of the guard.
//
// Serial: the refusing arm writes to the process-global os.Stderr, so
// it captures the streams (see setup_test.go's serial list).
func TestRefuseNonHumanSovereignForce(t *testing.T) {
	cases := []struct {
		name     string
		actor    string
		force    bool
		wantCode int
		wantOK   bool
	}{
		{
			name:     "no force flag is not this guard's business",
			actor:    "ai/claude",
			force:    false,
			wantCode: ExitOK,
			wantOK:   true,
		},
		{
			name:     "human actor with force passes",
			actor:    "human/peter",
			force:    true,
			wantCode: ExitOK,
			wantOK:   true,
		},
		{
			name:     "non-human actor with force is refused",
			actor:    "ai/claude",
			force:    true,
			wantCode: ExitFindings,
			wantOK:   false,
		},
		{
			// A bot is as non-human as an agent; the rule keys on the
			// absence of the human/ prefix, not on a known role list.
			name:     "bot actor with force is refused",
			actor:    "bot/release",
			force:    true,
			wantCode: ExitFindings,
			wantOK:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var code int
			var ok bool
			_, stderr := captureStdStreams(t, func() {
				code, ok = RefuseNonHumanSovereignForce("aiwf promote", tc.actor, tc.force)
			})
			if code != tc.wantCode || ok != tc.wantOK {
				t.Errorf("got (code=%d, ok=%v), want (code=%d, ok=%v)", code, ok, tc.wantCode, tc.wantOK)
			}
			if tc.wantOK {
				if stderr != "" {
					t.Errorf("an accepted invocation wrote to stderr: %q", stderr)
				}
				return
			}
			if !strings.Contains(stderr, "aiwf promote") {
				t.Errorf("the refusal does not name the verb it refused: %q", stderr)
			}
			if !strings.Contains(stderr, tc.actor) {
				t.Errorf("the refusal does not name the offending actor: %q", stderr)
			}
		})
	}
}

// TestRefuseNonHumanSovereignForce_SpeaksTheSeamsMessage is what makes
// this guard a second moment rather than a second opinion.
//
// The refusal an operator reads at the dispatcher must be the one the
// apply seam would have produced for the same act, because both consult
// verb.CheckForceTrailerCoherence. Deriving the expected text through
// that function rather than pasting a literal is what would catch the
// two drifting apart — a pasted string keeps passing while the seam's
// wording moves on.
//
// Serial: captures the process-global os.Stderr.
func TestRefuseNonHumanSovereignForce_SpeaksTheSeamsMessage(t *testing.T) {
	const actor = "ai/claude"

	seamErr := verb.CheckForceTrailerCoherence([]gitops.Trailer{
		{Key: gitops.TrailerActor, Value: actor},
		{Key: gitops.TrailerForce, Value: "any reason"},
	})
	if seamErr == nil {
		t.Fatal("the seam accepted a force trailer from a non-human actor; this test's premise " +
			"is gone and the guard it describes is asserting nothing")
	}

	_, stderr := captureStdStreams(t, func() {
		_, _ = RefuseNonHumanSovereignForce("aiwf promote", actor, true)
	})
	if !strings.Contains(stderr, seamErr.Error()) {
		t.Errorf("the dispatcher refusal does not carry the seam's own message.\n"+
			"  dispatcher: %q\n  seam:       %q\n"+
			"An operator meeting two wordings for one rule has to work out whether they are "+
			"the same rule", stderr, seamErr.Error())
	}
}
