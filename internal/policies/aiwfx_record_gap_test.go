package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aiwfxRecordGapSkillDir is the authoring location of the aiwfx-record-gap
// ritual. The embedded snapshot is canonical, so this seam-test asserts
// against the same bytes the binary embeds.
const aiwfxRecordGapSkillDir = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-record-gap"

// TestAiwfxRecordGap_DescriptionCarriesFilingTriggers pins the phrases a
// host matches to reach this ritual. The `description:` is the routing
// surface — read from the skill listing before any body is — and the
// ritual's value is firing mid-flow, when the work in hand is something
// else and nobody is looking for a gap-filing skill by name. A description
// reworded past these phrases leaves the ritual installed and unreachable,
// with nothing else to notice.
//
// The three pinned here are the ones that carry dispatch on their own: two
// name the act, one is how a defect is called out in conversation. The
// description lists further phrasings that are near-synonyms of these; they
// widen the match and are free to be reworded, which is why they are not
// pinned.
//
// Two properties a reader might expect here are held elsewhere and
// deliberately not duplicated: `name:` agreeing with the directory is
// PolicySkillCoverageMatchesVerbs, and the template path the body cites is
// TestShippedSurfaces_CiteOnlyMaterializedTemplatePaths. Both name this
// file by path when they fire.
func TestAiwfxRecordGap_DescriptionCarriesFilingTriggers(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(root, aiwfxRecordGapSkillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("loading %s/SKILL.md: %v", aiwfxRecordGapSkillDir, err)
	}
	desc := frontmatterField(string(body), "description")
	if desc == "" {
		t.Fatal("frontmatter `description:` must be non-empty")
	}
	lower := strings.ToLower(desc)
	for _, p := range []string{"file a gap", "record this defect", "that's a gap"} {
		if !strings.Contains(lower, p) {
			t.Errorf("description must carry the filing trigger phrase %q", p)
		}
	}
}
