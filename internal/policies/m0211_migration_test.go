package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestM0211_AC1_ClaudeMdIdCollisionSplitInPlace asserts M-0211/AC-1 for the
// hybrid CLAUDE.md id-collision section (Option B): the consumer-operating
// avoidance blocks are reduced to a pointer at the shipped homes, while the
// merge-time repo-development specialization stays.
func TestM0211_AC1_ClaudeMdIdCollisionSplitInPlace(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "CLAUDE.md"))
	if err != nil {
		t.Fatalf("reading CLAUDE.md: %v", err)
	}
	section := extractMarkdownSection(string(data), 2, "Id-collision resolution at merge time")
	if section == "" {
		t.Fatal("AC-1: CLAUDE.md must retain the `## Id-collision resolution at merge time` section")
	}
	lower := strings.ToLower(section)

	// The pointer at the shipped homes must be present.
	for _, w := range []string{"ships", "embedded guidance", "aiwf-add"} {
		if !strings.Contains(lower, strings.ToLower(w)) {
			t.Errorf("AC-1: the id-collision section must point at the shipped homes — missing %q", w)
		}
	}
	// The merge-time repo-development specialization must stay.
	for _, w := range []string{"git mv", "E-0033"} {
		if !strings.Contains(section, w) {
			t.Errorf("AC-1: the id-collision section must retain its merge-time repo-development content — missing %q", w)
		}
	}
	// The migrated consumer-operating blocks must be gone (split, not duplicated).
	for _, w := range []string{"How to avoid collisions:", "What to expect:"} {
		if strings.Contains(section, w) {
			t.Errorf("AC-1: the consumer-operating %q block must move to the shipped homes, not stay duplicated in CLAUDE.md", w)
		}
	}
}

// TestM0211_AC3_AuthoringRuleNamesDividingPrinciple asserts M-0211/AC-3: CLAUDE.md
// carries an authoring section that names the audience-based dividing principle,
// points at the embedded guidance source as the shippable home, and references
// the mechanical chokepoint.
func TestM0211_AC3_AuthoringRuleNamesDividingPrinciple(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "CLAUDE.md"))
	if err != nil {
		t.Fatalf("reading CLAUDE.md: %v", err)
	}
	section := extractMarkdownSection(string(data), 2, "Consumer-operating guidance vs repo-development guidance")
	if section == "" {
		t.Fatal("AC-3: CLAUDE.md must have a `## Consumer-operating guidance vs repo-development guidance` authoring section")
	}
	lower := strings.ToLower(section)
	// Names the dividing principle and the split-hybrid rule.
	for _, w := range []string{"audience, not importance", "split"} {
		if !strings.Contains(lower, strings.ToLower(w)) {
			t.Errorf("AC-3: the authoring section must name %q", w)
		}
	}
	// Points at the shippable home (the guidance source path) and the chokepoint.
	if !strings.Contains(section, "internal/skills/embedded-guidance/aiwf-guidance.md") {
		t.Error("AC-3: the authoring section must point at the embedded guidance source as the shippable home")
	}
	if !strings.Contains(section, "PolicyM0211GuidanceOperatingAnchors") {
		t.Error("AC-3: the authoring section must reference the drift chokepoint (PolicyM0211GuidanceOperatingAnchors)")
	}
}
