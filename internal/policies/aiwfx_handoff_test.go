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
// the same bytes the binary embeds. Naming the two ritual paths here
// also satisfies the skill-edit-structural-test-backstop, which needs a
// policy test referencing each edited embedded-rituals SKILL.md.
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

// TestAiwfxHandoff_BlockFormatAndRule pins the two load-bearing body
// sections: the volatile-first block-format template, and the
// volatile-vs-durable rule that is the skill's whole point (carry what
// git/aiwf cannot reconstruct; point into the tree for the rest).
func TestAiwfxHandoff_BlockFormatAndRule(t *testing.T) {
	t.Parallel()
	body := loadHandoffFixture(t, aiwfxHandoffFixturePath)

	block := extractMarkdownSection(body, 2, "Block format")
	if block == "" {
		t.Fatal("SKILL.md must have a `## Block format` section")
	}
	// The template carries the tree-pointer line (durable half, by
	// reference) and the volatile payload markers. Assert both so a
	// future edit that drops the pointer line or the payload trips.
	for _, want := range []string{"aiwf show", "aiwf status", "aiwf history", "Next:", "Watch out"} {
		if !strings.Contains(block, want) {
			t.Errorf("§Block format template must contain %q", want)
		}
	}

	rule := extractMarkdownSection(body, 2, "Volatile vs durable")
	if rule == "" {
		t.Fatal("SKILL.md must have a `## Volatile vs durable` rule section")
	}
	lowerRule := strings.ToLower(rule)
	for _, want := range []string{"volatile", "durable", "point into"} {
		if !strings.Contains(lowerRule, want) {
			t.Errorf("§Volatile vs durable must name %q", want)
		}
	}
}

// TestAiwfxHandoff_BoundaryReferences pins that both boundary rituals
// reference aiwfx-handoff in the correct section — the AC boundary
// lives in start-milestone §6 (the per-AC loop), and the milestone
// close lives in wrap-milestone §Next step. Scoped to the named section
// so a stray mention elsewhere can't satisfy the assertion vacuously.
func TestAiwfxHandoff_BoundaryReferences(t *testing.T) {
	t.Parallel()

	start := loadHandoffFixture(t, aiwfxHandoffStartMilestonePath)
	step6 := extractMarkdownSection(start, 3, "6. Implementation")
	if step6 == "" {
		t.Fatal("aiwfx-start-milestone must retain its `### 6. Implementation` section")
	}
	if !strings.Contains(step6, "aiwfx-handoff") {
		t.Error("aiwfx-start-milestone §6 must reference `aiwfx-handoff` at the AC boundary")
	}

	wrap := loadHandoffFixture(t, aiwfxHandoffWrapMilestonePath)
	next := extractMarkdownSection(wrap, 2, "Next step")
	if next == "" {
		t.Fatal("aiwfx-wrap-milestone must retain its `## Next step` section")
	}
	if !strings.Contains(next, "aiwfx-handoff") {
		t.Error("aiwfx-wrap-milestone §Next step must reference `aiwfx-handoff` at the milestone close")
	}
}
