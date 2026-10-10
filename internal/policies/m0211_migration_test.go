package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
