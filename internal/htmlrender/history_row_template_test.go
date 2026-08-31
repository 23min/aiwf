package htmlrender

import (
	"bytes"
	"strings"
	"testing"
)

// history_row_template_test.go executes every page template that renders
// history rows, with rows carrying the shapes those columns exist for.
//
// The display rules — an absent trailer reads "-", an agent acting for a
// principal reads "principal via agent" — are applied once before a row is
// built, so the templates print what they are handed. Nothing else executes
// them, so a template that went back to composing the actor column itself
// would render a doubled attribution unnoticed; HistoryRow carries no
// Principal field, which makes that a render error this test surfaces.
//
// Counts, not presence. Both the verb and actor columns render the marker for
// an event carrying neither trailer, so asserting it once is satisfied by
// whichever comes first and leaves the other free to drop its field.
func TestHistoryRowTemplates_RenderTheResolvedColumns(t *testing.T) {
	t.Parallel()

	rows := []HistoryRow{
		// The shape D-0071 creates: no verb, no actor, both columns marked.
		{Date: "2026-08-31", Commit: "abc1234", Verb: "-", Actor: "-", Detail: "fix(x): a shipped surface"},
		{
			Date: "2026-08-31", Commit: "def5678", Verb: "promote", To: "done",
			Actor: "human/peter via ai/claude", Detail: "aiwf promote M-0001 done",
		},
	}

	// Two marked table cells per row-1: the verb column and the actor column.
	tableCells := map[string]int{"<td>-</td>": 2, "human/peter via ai/claude": 1}

	for _, tc := range []struct {
		tmpl  string
		data  any
		wants map[string]int
	}{
		{"entity.tmpl", EntityData{Entity: &EntityRef{ID: "G-0001", Title: "x"}, History: rows}, tableCells},
		{"epic.tmpl", EpicData{Epic: &EntityRef{ID: "E-0001", Title: "x"}, History: rows}, tableCells},
		{"milestone.tmpl", MilestoneData{Milestone: &EntityRef{ID: "M-0001", Title: "x"}, Commits: rows}, tableCells},
		// The Provenance tab renders the same rows through spans the Commits
		// tab above never reaches, so each span is named on its own.
		{"milestone.tmpl/provenance", MilestoneData{
			Milestone:  &EntityRef{ID: "M-0001", Title: "x"},
			Provenance: ProvenanceData{Timeline: rows},
		}, map[string]int{
			`<span class="verb">-`:      1,
			`<span class="actor">-`:     1,
			"human/peter via ai/claude": 1,
		}},
		{"status.tmpl", StatusData{RecentActivity: rows}, tableCells},
	} {
		t.Run(tc.tmpl, func(t *testing.T) {
			t.Parallel()
			tmpls, err := loadTemplates()
			if err != nil {
				t.Fatalf("loadTemplates: %v", err)
			}
			name, _, _ := strings.Cut(tc.tmpl, "/")
			var buf bytes.Buffer
			if err := tmpls.ExecuteTemplate(&buf, name, tc.data); err != nil {
				t.Fatalf("executing %s with history rows: %v", name, err)
			}
			out := buf.String()
			for lit, want := range tc.wants {
				if got := strings.Count(out, lit); got < want {
					t.Errorf("%s renders %q %d times, want at least %d — a column dropped its field",
						tc.tmpl, lit, got, want)
				}
			}
			// A template composing the attribution itself doubles the "via".
			if strings.Contains(out, "via human/peter via") || strings.Contains(out, "via </td>") {
				t.Errorf("%s composed the actor column itself", tc.tmpl)
			}
		})
	}
}
