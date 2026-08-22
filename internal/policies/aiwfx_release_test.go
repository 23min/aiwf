package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aiwfxReleaseFixturePath is the canonical authoring location for the
// `aiwfx-release` skill body — the embedded ritual snapshot the aiwf
// binary ships. Per G-0182, AC content assertions read the embedded
// bytes directly rather than a duplicated fixture under
// internal/policies/testdata/.
const aiwfxReleaseFixturePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-release/SKILL.md"

// loadAiwfxReleaseFixture reads the fixture relative to repo root.
func loadAiwfxReleaseFixture(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, aiwfxReleaseFixturePath))
	if err != nil {
		t.Fatalf("loading %s: %v", aiwfxReleaseFixturePath, err)
	}
	return string(data)
}

// TestAiwfxRelease_DelegatesToDeployerAgent pins G-0361's fix: the
// `## When to use` section instructs dispatching the `deployer`
// subagent rather than running the ritual inline in the calling
// session, and its trigger-phrase list covers the phrasings that
// previously had no match at all ("let's ship", "let's release",
// "make a new version", "cut a release") — the gap that left the
// deployer agent at ~0 dispatches despite existing and being fully
// documented (session mining in the archived G-0353 gap).
//
// Scoped to `## When to use` per CLAUDE.md *Substring assertions are
// not structural assertions*: the delegation instruction and the
// phrase list could otherwise pass vacuously by appearing somewhere
// unrelated in the file (e.g. the Workflow section already names
// unrelated git commands that happen to share substrings).
func TestAiwfxRelease_DelegatesToDeployerAgent(t *testing.T) {
	t.Parallel()
	body := loadAiwfxReleaseFixture(t)

	whenToUse := extractMarkdownSection(body, 2, "When to use")
	if whenToUse == "" {
		t.Fatal("G-0361: aiwfx-release must have a `## When to use` section")
	}

	if !strings.Contains(whenToUse, "Dispatch the `deployer` subagent") {
		t.Error("G-0361: `## When to use` must instruct dispatching the `deployer` subagent rather than running inline")
	}

	requiredPhrases := []string{
		"let's release",
		"let's ship",
		"make a new version",
		"cut a release",
	}
	lower := strings.ToLower(whenToUse)
	for _, p := range requiredPhrases {
		if !strings.Contains(lower, p) {
			t.Errorf("G-0361: `## When to use` must name the trigger phrase %q", p)
		}
	}
}

// TestAiwfxRelease_FrontmatterDescriptionRoutesToDeployer pins the other
// half of G-0361's fix: the *frontmatter* `description:` field — not
// just the `## When to use` body section — names both the broadened
// trigger phrases and the `deployer` delegation. This matters because
// the description field, not the skill body, is what an assistant sees
// in the available-skills listing *before* deciding whether to invoke
// the skill at all; a body-only fix only helps after that decision is
// already made. Mirrors the "Use when the user says ..." pattern
// already present in sibling rituals' descriptions (e.g.
// aiwfx-wrap-epic), which aiwfx-release's description previously
// lacked entirely.
func TestAiwfxRelease_FrontmatterDescriptionRoutesToDeployer(t *testing.T) {
	t.Parallel()
	body := loadAiwfxReleaseFixture(t)

	desc := frontmatterField(body, "description")
	if desc == "" {
		t.Fatal("G-0361: aiwfx-release frontmatter `description:` must be non-empty")
	}

	if !strings.Contains(desc, "deployer") {
		t.Error("G-0361: frontmatter `description:` must name the `deployer` agent so the delegation is visible before the skill body is ever read")
	}

	lower := strings.ToLower(desc)
	for _, p := range []string{"let's ship", "cut a release"} {
		if !strings.Contains(lower, p) {
			t.Errorf("G-0361: frontmatter `description:` must name the trigger phrase %q", p)
		}
	}
}
