package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// m0211AddSkillFixturePath is the canonical authoring location for the
// `aiwf-add` verb skill — the on-demand home for the full cross-branch
// id-allocation mechanics migrated off CLAUDE.md in M-0211/AC-1.
const m0211AddSkillFixturePath = "internal/skills/embedded/aiwf-add/SKILL.md"

// bulletByLead returns the top-level markdown bullet whose bolded lead-in is
// `lead` (e.g. "**Allocate ids on your working branch"), from that lead up to
// the next top-level `- **` bullet (or the section end). Scoping to the bullet
// rather than grepping the whole file is required by CLAUDE.md *Substring
// assertions are not structural assertions*.
func bulletByLead(t *testing.T, body, lead string) string {
	t.Helper()
	start := strings.Index(body, lead)
	if start < 0 {
		return ""
	}
	rest := body[start+len(lead):]
	if end := strings.Index(rest, "\n- **"); end >= 0 {
		return lead + rest[:end]
	}
	return lead + rest
}

// TestM0211_AC1_GuidanceCarriesCrossBranchRule asserts M-0211/AC-1 for the
// always-on guidance source: the tight cross-branch id-allocation operating
// rule (allocate on your working branch; `aiwf add --fetch`; push promptly) is
// present, so a consumer's materialized guidance carries it.
func TestM0211_AC1_GuidanceCarriesCrossBranchRule(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), m0209GuidanceFixturePath))
	if err != nil {
		t.Fatalf("reading %s: %v", m0209GuidanceFixturePath, err)
	}
	bullet := bulletByLead(t, string(data), "**Allocate ids on your working branch")
	if bullet == "" {
		t.Fatal("AC-1: guidance must carry an `**Allocate ids on your working branch…**` operating bullet (the tight cross-branch allocation rule)")
	}
	lower := strings.ToLower(bullet)
	for _, w := range []string{"--fetch", "push promptly", "working branch"} {
		if !strings.Contains(lower, strings.ToLower(w)) {
			t.Errorf("AC-1: the guidance cross-branch bullet must carry %q", w)
		}
	}
}

// TestM0211_AC1_AddSkillCarriesCrossBranchMechanics asserts M-0211/AC-1 for the
// on-demand tier: the `aiwf-add` verb skill carries the full cross-branch
// allocation mechanics (the detail G-0313 routes to an on-demand skill so the
// always-on guidance stays tight).
func TestM0211_AC1_AddSkillCarriesCrossBranchMechanics(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), m0211AddSkillFixturePath))
	if err != nil {
		t.Fatalf("reading %s: %v", m0211AddSkillFixturePath, err)
	}
	section := extractMarkdownSection(string(data), 2, "Allocating ids across branches and clones")
	if section == "" {
		t.Fatal("AC-1: aiwf-add skill must have a `## Allocating ids across branches and clones` section carrying the full mechanics")
	}
	lower := strings.ToLower(section)
	for _, w := range []string{"--fetch", "push promptly", "worktree", "separate clones", "invisible", "backtick"} {
		if !strings.Contains(lower, strings.ToLower(w)) {
			t.Errorf("AC-1: the aiwf-add cross-branch section must carry %q", w)
		}
	}
}

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
