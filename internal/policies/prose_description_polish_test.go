package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-0200 (G-0298) — structural tests pinning the five prose/description
// polish fixes. Each test reads the authored skill body from the embedded
// ritual snapshot (the source of truth per ADR-0016) and asserts the
// corrected content, so a future edit that reintroduces the defect reddens.
//
// The aiwfx-whiteboard and aiwfx-wrap-epic path constants are declared by
// their own test files; this file reuses them rather than redeclaring
// (H1 in practice: one source per path).
const (
	aiwfxPlanEpicFixturePath    = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-epic/SKILL.md"
	wfCodebaseHealthFixturePath = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-codebase-health/SKILL.md"
)

// loadPolishFixture reads a skill body relative to repo root.
func loadPolishFixture(t *testing.T, path string) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatalf("loading %s: %v", path, err)
	}
	return string(data)
}

// firstSentence returns the prefix of s up to (and excluding) the first
// sentence terminator ". " — the description's opening clause. If none is
// found the whole string is the opening.
func firstSentence(s string) string {
	if i := strings.Index(s, ". "); i != -1 {
		return s[:i]
	}
	return s
}

// TestFirstSentence_BranchCoverage exercises both reachable branches of the
// firstSentence helper (terminator present / absent).
func TestFirstSentence_BranchCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, in, want string
	}{
		{"terminator present", "First one. Second one.", "First one"},
		{"no terminator", "Only a fragment", "Only a fragment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := firstSentence(tc.in); got != tc.want {
				t.Errorf("firstSentence(%q) = %q; want %q", tc.in, got, tc.want)
			}
		})
	}
}
