package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestM0195_AC5_SkillBodyDisciplineInClaudeMd asserts M-0195/AC-5: the strict
// skill-body id-reference discipline (G-0299) is documented in CLAUDE.md's
// Skills-policy section. Scoped to that section — not a flat file grep — per
// CLAUDE.md *Testing* §"Substring assertions are not structural assertions":
// the rule must live in the skills policy, not float anywhere in the file.
func TestM0195_AC5_SkillBodyDisciplineInClaudeMd(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "CLAUDE.md"))
	if err != nil {
		t.Fatalf("reading CLAUDE.md: %v", err)
	}
	section := extractMarkdownSection(string(data), 3, "Skills policy")
	if section == "" {
		t.Fatal("CLAUDE.md must have a `### Skills policy` section carrying the skill-body discipline")
	}
	lower := strings.ToLower(section)

	// Each required element of the standing rule must appear inside the
	// Skills-policy section.
	wantPresent := []struct{ phrase, why string }{
		{"skill-body-id", "names the mechanical chokepoint (the check)"},
		{"canonical", "the canonical placeholder convention"},
		{"placeholder", "illustrative content uses placeholders"},
		{"carve-out", "the design/ADR doc-link carve-out"},
		{"consumer", "the cross-tree-leakage rationale"},
	}
	for _, w := range wantPresent {
		if !strings.Contains(lower, w.phrase) {
			t.Errorf("`### Skills policy` missing %q (%s)", w.phrase, w.why)
		}
	}

	// The core prohibition itself must be stated.
	if !strings.Contains(lower, "no real entity id") && !strings.Contains(lower, "cite no real") {
		t.Error("`### Skills policy` must state shipped skill bodies cite no real entity id, path, or inline status")
	}
}
