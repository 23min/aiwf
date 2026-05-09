package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// aiwfxWhiteboardFixturePath is the canonical authoring location
// for the `aiwfx-whiteboard` skill body during M-079, per CLAUDE.md
// §"Cross-repo plugin testing". At wrap, the fixture content is
// copied to the rituals plugin repo (`plugins/aiwf-extensions/
// skills/aiwfx-whiteboard/SKILL.md` there); a drift-check test
// guards the long-term coupling.
const aiwfxWhiteboardFixturePath = "internal/policies/testdata/aiwfx-whiteboard/SKILL.md"

// loadAiwfxWhiteboardFixture reads the fixture relative to repo
// root. The tests under this file are seam-tests against the
// authored skill body — they assert the doctrinal content M-079's
// ACs require, scoped to the relevant markdown section per
// CLAUDE.md *Testing* §"Substring assertions are not structural
// assertions".
func loadAiwfxWhiteboardFixture(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, aiwfxWhiteboardFixturePath))
	if err != nil {
		t.Fatalf("loading %s: %v", aiwfxWhiteboardFixturePath, err)
	}
	return string(data)
}

// frontmatterField extracts a single-line frontmatter value (the
// pattern aiwfx-* skills use; block-scalar `|` form is not yet
// produced by this skill). Returns "" if not found.
func frontmatterField(body, key string) string {
	// Frontmatter ends at the second `---`.
	if !strings.HasPrefix(body, "---\n") {
		return ""
	}
	end := strings.Index(body[4:], "\n---")
	if end == -1 {
		return ""
	}
	front := body[4 : 4+end]
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `:\s*(.+?)\s*$`)
	m := re.FindStringSubmatch(front)
	if m == nil {
		return ""
	}
	return m[1]
}

// TestFrontmatterField_BranchCoverage exercises every reachable
// branch of frontmatterField against synthetic inputs. The helper
// is only ever called from this test file today, but each branch
// is reachable via real inputs (a body missing the frontmatter
// fence, an unterminated frontmatter, a frontmatter that doesn't
// carry the queried key) and the project's branch-coverage rule
// applies even to test-package helpers.
func TestFrontmatterField_BranchCoverage(t *testing.T) {
	cases := []struct {
		name string
		body string
		key  string
		want string
	}{
		{"no leading fence", "no frontmatter here\n", "name", ""},
		{"unterminated frontmatter", "---\nname: x\n", "name", ""},
		{"key not present", "---\ndescription: x\n---\n", "name", ""},
		{"key present", "---\nname: aiwfx-x\n---\nbody", "name", "aiwfx-x"},
		{"key present with surrounding whitespace", "---\nname:   aiwfx-x   \n---\n", "name", "aiwfx-x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := frontmatterField(tc.body, tc.key)
			if got != tc.want {
				t.Errorf("frontmatterField(%q) = %q; want %q", tc.body, got, tc.want)
			}
		})
	}
}

// TestAiwfxWhiteboard_AC1_SkillScaffolded asserts AC-1: the skill
// fixture exists with frontmatter declaring `name: aiwfx-whiteboard`
// (matching the directory) and a non-empty `description:`. This is
// the v1 single-SKILL.md layout (no template subdirs).
func TestAiwfxWhiteboard_AC1_SkillScaffolded(t *testing.T) {
	body := loadAiwfxWhiteboardFixture(t)

	name := frontmatterField(body, "name")
	if name != "aiwfx-whiteboard" {
		t.Errorf("AC-1: frontmatter `name:` must be `aiwfx-whiteboard` (got %q)", name)
	}

	desc := frontmatterField(body, "description")
	if desc == "" {
		t.Error("AC-1: frontmatter `description:` must be non-empty")
	}
}
