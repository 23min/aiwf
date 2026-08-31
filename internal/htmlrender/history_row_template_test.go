package htmlrender

import (
	"bytes"
	"strings"
	"testing"
)

// history_row_template_test.go executes the page templates that render history
// rows, with rows carrying the shapes those columns exist for.
//
// The display rules — an absent trailer reads "-", an agent acting for a
// principal reads "principal via agent" — are applied once before a row is
// built, so the templates print what they are handed. Nothing else executes
// them with history rows, so a template that went back to composing the actor
// column itself would render a doubled attribution unnoticed. HistoryRow
// carries no Principal field, which makes that a render error rather than a
// wrong page — and this is the test that surfaces it.
func TestHistoryRowTemplates_RenderTheResolvedColumns(t *testing.T) {
	t.Parallel()

	tmpls, err := loadTemplates()
	if err != nil {
		t.Fatalf("loadTemplates: %v", err)
	}
	rows := []HistoryRow{
		// The shape D-0071 creates: no verb, no actor, both columns marked.
		{Date: "2026-08-31", Commit: "abc1234", Verb: "-", Actor: "-", Detail: "fix(x): a shipped surface"},
		{
			Date: "2026-08-31", Commit: "def5678", Verb: "promote", To: "done",
			Actor: "human/peter via ai/claude", Detail: "aiwf promote M-0001 done",
		},
	}

	for _, tc := range []struct {
		tmpl string
		data any
	}{
		{"entity.tmpl", EntityData{Entity: &EntityRef{ID: "G-0001", Title: "x"}, History: rows}},
		{"epic.tmpl", EpicData{Epic: &EntityRef{ID: "E-0001", Title: "x"}, History: rows}},
		{"milestone.tmpl", MilestoneData{Milestone: &EntityRef{ID: "M-0001", Title: "x"}, Commits: rows}},
		{"status.tmpl", StatusData{RecentActivity: rows}},
	} {
		t.Run(tc.tmpl, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if err := tmpls.ExecuteTemplate(&buf, tc.tmpl, tc.data); err != nil {
				t.Fatalf("executing %s with history rows: %v", tc.tmpl, err)
			}
			out := buf.String()
			for _, want := range []string{"<td>-</td>", "human/peter via ai/claude"} {
				if !strings.Contains(out, want) {
					t.Errorf("%s output missing %q", tc.tmpl, want)
				}
			}
			// A template composing the attribution itself doubles the "via".
			if strings.Contains(out, "via human/peter via") || strings.Contains(out, "via </td>") {
				t.Errorf("%s composed the actor column itself", tc.tmpl)
			}
		})
	}
}
