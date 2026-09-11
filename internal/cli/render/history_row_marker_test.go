package render

import (
	"testing"

	"github.com/23min/aiwf/internal/entityview"
)

// history_row_marker_test.go pins the two columns historyEventToRow resolves.
//
// The rule they carry — an absent trailer renders "-" — lives in entityview so
// three surfaces cannot disagree about it. Nothing else asserts that the HTML
// surface applies it: the templates print whatever the row holds, and the
// single-pass differential compares this bucket against ReadHistory, which
// moves with it. Without these cases both call sites revert with the suite
// green, and the rendered site goes back to blank cells and a dangling
// separator where a principal has no agent.
func TestHistoryEventToRow_RendersTheAbsentTrailerMarker(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name            string
		event           entityview.HistoryEvent
		wantVerb, wantA string
	}{
		{
			// The shape D-0071 creates: provenance is the entity trailer
			// alone, so neither column has a trailer to name.
			name:     "no verb and no actor",
			event:    entityview.HistoryEvent{Detail: "fix(x): a shipped surface"},
			wantVerb: "-", wantA: "-",
		},
		{
			name:     "an ordinary verb event is untouched",
			event:    entityview.HistoryEvent{Verb: "promote", Actor: "human/peter"},
			wantVerb: "promote", wantA: "human/peter",
		},
		{
			// An empty agent side must not print a dangling separator, and
			// the principal is provenance the commit does carry.
			name:     "a principal without an actor renders alone",
			event:    entityview.HistoryEvent{Verb: "promote", Principal: "human/peter"},
			wantVerb: "promote", wantA: "human/peter",
		},
		{
			name:     "an agent acting for a principal",
			event:    entityview.HistoryEvent{Verb: "promote", Actor: "ai/claude", Principal: "human/peter"},
			wantVerb: "promote", wantA: "human/peter via ai/claude",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			row := historyEventToRow(&tc.event)
			if row.Verb != tc.wantVerb {
				t.Errorf("Verb = %q, want %q", row.Verb, tc.wantVerb)
			}
			if row.Actor != tc.wantA {
				t.Errorf("Actor = %q, want %q", row.Actor, tc.wantA)
			}
		})
	}
}
