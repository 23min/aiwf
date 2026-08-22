package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// m0209GuidanceFixturePath is the canonical authoring location for the
// per-turn LLM guidance fragment (ADR-0018). `.claude/aiwf-guidance.md` in a
// consumer repo is materialized from these embedded bytes by `aiwf init` /
// `aiwf update`, so AC-1's content claims are asserted against the source,
// never the gitignored rendered artifact.
const m0209GuidanceFixturePath = "internal/skills/embedded-guidance/aiwf-guidance.md"

// gateDisciplineBullet returns the "Gate discipline survives compaction"
// bullet block from CLAUDE.md's `## Working with the user` section — from the
// bolded lead-in up to the next top-level `- **` bullet (or the section end).
//
// Scoping to the bullet (rather than grepping the whole file) is required by
// CLAUDE.md *Testing* §"Substring assertions are not structural assertions":
// the generalized-gate language must live in the gate-discipline rule itself,
// not float anywhere in a 600-line file.
func gateDisciplineBullet(t *testing.T, claudeMd string) string {
	t.Helper()
	section := extractMarkdownSection(claudeMd, 2, "Working with the user")
	if section == "" {
		t.Fatal("CLAUDE.md must have a `## Working with the user` section carrying the gate-discipline rule")
	}
	const lead = "**Gate discipline survives compaction.**"
	start := strings.Index(section, lead)
	if start < 0 {
		t.Fatalf("`## Working with the user` must contain the %q bullet", lead)
	}
	rest := section[start+len(lead):]
	// The bullet ends at the next top-level list item.
	if end := strings.Index(rest, "\n- **"); end >= 0 {
		return lead + rest[:end]
	}
	return lead + rest
}

// TestM0209_AC1_GeneralizedGateInClaudeMd asserts M-0209/AC-1 for CLAUDE.md:
// the declared-sequence gate is documented as a *general* capability for any
// local, reversible mutation sequence (with the bright line that excludes
// outward/irreversible and timing-bearing actions), and the false "wf-patch
// only; milestone and epic wraps keep per-action gates" scoping is gone.
func TestM0209_AC1_GeneralizedGateInClaudeMd(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "CLAUDE.md"))
	if err != nil {
		t.Fatalf("reading CLAUDE.md: %v", err)
	}
	bullet := gateDisciplineBullet(t, string(data))
	lower := strings.ToLower(bullet)

	// The generalized capability and its bright line must be present.
	wantPresent := []string{
		"declared-sequence gate",
		"local, reversible",
		"outward",
		"timing-bearing",
	}
	for _, w := range wantPresent {
		if !strings.Contains(lower, strings.ToLower(w)) {
			t.Errorf("AC-1: gate-discipline bullet must document the generalized gate — missing %q", w)
		}
	}

	// The false restrictive scoping must be gone. G-0295: CLAUDE.md asserted
	// the wraps "keep per-action gates," which was untrue.
	wantAbsent := []string{
		"Scope is wf-patch only",
		"milestone and epic wraps keep per-action gates",
	}
	for _, w := range wantAbsent {
		if strings.Contains(bullet, w) {
			t.Errorf("AC-1: the false %q scoping must be rewritten (G-0295)", w)
		}
	}
}

// findWorkflowSubsection returns the `### ` subsection of `## Workflow` whose
// heading text (lowercased) contains needle, or "" if none matches.
func findWorkflowSubsection(body, needle string) string {
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		return ""
	}
	for _, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.TrimPrefix(line, "### ")
		if strings.Contains(strings.ToLower(text), needle) {
			return extractMarkdownSection(body, 3, text)
		}
	}
	return ""
}

// TestFindWorkflowSubsection_BranchCoverage covers the defensive return arms
// the fixture-driven callers never reach. Every caller guards on `== ""` and
// fatals, so an unexercised no-match arm would leave each of those guards
// resting on a path nothing had ever traversed.
func TestFindWorkflowSubsection_BranchCoverage(t *testing.T) {
	t.Parallel()

	if got := findWorkflowSubsection("# No workflow heading here\n", "merge"); got != "" {
		t.Errorf("absent Workflow section: want empty, got %q", got)
	}
	if got := findWorkflowSubsection("## Workflow\n\n### 1. Something else\n\nbody\n", "merge"); got != "" {
		t.Errorf("Workflow without a matching heading: want empty, got %q", got)
	}
}
