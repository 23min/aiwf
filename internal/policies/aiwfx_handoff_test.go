package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Authoring locations for the aiwfx-handoff skill and the two boundary
// rituals that reference it (G-0351). Per G-0182 the embedded snapshot
// is the canonical authoring location; these seam-tests assert against
// the same bytes the binary embeds.
const (
	aiwfxHandoffFixturePath        = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-handoff/SKILL.md"
	aiwfxHandoffStartMilestonePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md"
	aiwfxHandoffWrapMilestonePath  = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md"
)

// loadHandoffFixture reads an embedded-rituals SKILL.md relative to repo
// root for the aiwfx-handoff seam-tests below.
func loadHandoffFixture(t *testing.T, relPath string) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, relPath))
	if err != nil {
		t.Fatalf("loading %s: %v", relPath, err)
	}
	return string(data)
}

// TestAiwfxHandoff_SkillScaffolded pins that the skill exists with
// frontmatter declaring `name: aiwfx-handoff` (matching its directory)
// and a `description:` that carries the on-request trigger phrases —
// the description is the routing surface a host matches against, so the
// on-demand affordance the gap specifies lives there.
func TestAiwfxHandoff_SkillScaffolded(t *testing.T) {
	t.Parallel()
	body := loadHandoffFixture(t, aiwfxHandoffFixturePath)

	if name := frontmatterField(body, "name"); name != "aiwfx-handoff" {
		t.Errorf("frontmatter `name:` must be `aiwfx-handoff` (got %q)", name)
	}

	desc := frontmatterField(body, "description")
	if desc == "" {
		t.Fatal("frontmatter `description:` must be non-empty")
	}
	lower := strings.ToLower(desc)
	// On-request trigger phrasings per the gap spec; the skill fires
	// mid-conversation on these, not only at a boundary.
	phrases := []string{"give me a handoff", "prime the compact", "where are we for /compact"}
	for _, p := range phrases {
		if !strings.Contains(lower, p) {
			t.Errorf("description must carry the on-request trigger phrase %q", p)
		}
	}
}
