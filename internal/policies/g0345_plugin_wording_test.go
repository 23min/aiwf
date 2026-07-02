package policies

import (
	"strings"
	"testing"
)

// TestNoObsoletePluginWording_G0345 pins the terminology cleanup half of
// G-0345: the ritual snapshot is one embedded bundle (ADR-0014/0016), so prose
// implying a live "plugin" a skill belongs to — or a marketplace-era plugin
// install — is obsolete. Each edited skill must be free of the specific stale
// phrasings this patch removed.
//
// Scoped to the exact obsolete substrings, NOT the word "plugin" wholesale:
// "plugin" remains legitimate architectural vocabulary elsewhere (e.g.
// ADR-0007's kernel-vs-plugin tiering), so a blanket ban would over-reach.
func TestNoObsoletePluginWording_G0345(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		path     string
		obsolete []string
	}{
		{"record-decision", aiwfxRecordDecisionFixturePath, []string{"plugin template", "this plugin's"}},
		{"plan-epic", aiwfxPlanEpicFixturePath, []string{"plugin's template", "aiwf-extensions plugin", "this plugin's"}},
		{"plan-milestones", aiwfxPlanMilestonesFixturePath, []string{"plugin's template", "this plugin's"}},
		{"wf-doc-lint", wfDocLintFixturePath, []string{"The plugin ships"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := loadPolishFixture(t, tc.path)
			for _, phrase := range tc.obsolete {
				if strings.Contains(body, phrase) {
					t.Errorf("G-0345: %s must drop obsolete plugin wording %q (the rituals are one embedded snapshot, not a live plugin)", tc.name, phrase)
				}
			}
		})
	}
}
