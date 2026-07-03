package policies

import (
	"strings"
	"testing"
)

// M-0190 — the start rituals (aiwfx-start-epic / aiwfx-start-milestone)
// default to in-repo worktree placement under the configured worktree.dir,
// keep the per-invocation override, and record the devcontainer-sandbox
// rationale citing ADR-0023. These tests assert the doc-shaped ACs against
// the embedded ritual snapshot bytes, scoped to the relevant SKILL.md
// section per CLAUDE.md §"Substring assertions are not structural
// assertions". The start-epic / start-milestone fixture loaders and the
// findWorktreePromptSection / extractMarkdownSection helpers live in
// aiwfx_start_epic_test.go and aiwfx_start_milestone_test.go (same package).

// findStartMilestoneCutSection locates the `### 5. Cut the milestone branch`
// subsection inside `## Workflow`. Heading-content driven (case-insensitive
// match on "cut") so a future reshuffle that moves the step to a different
// number does not silently break the structural check — what matters is that
// the worktree-placement note lives with the branch-cut step, not which
// number carries it.
func findStartMilestoneCutSection(body string) string {
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		return ""
	}
	for _, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.TrimPrefix(line, "### ")
		if strings.Contains(strings.ToLower(text), "cut") {
			return extractMarkdownSection(body, 3, text)
		}
	}
	return ""
}

// TestFindStartMilestoneCutSection_BranchCoverage covers the defensive
// return arms the happy-path fixture test does not reach.
func TestFindStartMilestoneCutSection_BranchCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		want string
	}{
		{"missing-workflow", "prose only", ""},
		{"workflow-without-cut-heading", "## Workflow\n\n### 1. Some other step\n\nbody\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := findStartMilestoneCutSection(tc.body); got != tc.want {
				t.Errorf("findStartMilestoneCutSection(%q) = %q; want %q", tc.name, got, tc.want)
			}
		})
	}
}

// marker is a substring expectation against a section. fold matches
// case-insensitively (for prose); otherwise the substring is matched
// verbatim (for path literals and ids).
type marker struct {
	name   string
	needle string
	fold   bool
}

func assertMarkers(t *testing.T, where, section string, markers []marker) {
	t.Helper()
	lower := strings.ToLower(section)
	for _, m := range markers {
		hay, needle := section, m.needle
		if m.fold {
			hay, needle = lower, strings.ToLower(m.needle)
		}
		if !strings.Contains(hay, needle) {
			t.Errorf("%s must name %s (substring %q)", where, m.name, m.needle)
		}
	}
}

// TestAiwfxStartEpic_M0190_AC1_WorktreeDefaultsToInRepo pins M-0190/AC-1:
// the start-epic worktree-placement step recommends in-repo placement as the
// DEFAULT, reading the configured worktree.dir from the kernel (via
// `aiwf doctor`) rather than hardcoding it. Heading-scoped to the worktree
// subsection — the default claim must live inside the placement step, not
// float elsewhere in the skill body.
func TestAiwfxStartEpic_M0190_AC1_WorktreeDefaultsToInRepo(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartEpicFixture(t)

	section := findWorktreePromptSection(body)
	if section == "" {
		t.Fatal("AC-1: start-epic `## Workflow` must contain a `### …worktree…` subsection")
	}
	assertMarkers(t, "AC-1: start-epic worktree subsection", section, []marker{
		{"the worktree.dir knob", "worktree.dir", false},
		{"the in-repo default directory", ".claude/worktrees", false},
		{"the kernel read mechanism (aiwf doctor)", "aiwf doctor", false},
		{"the default framing", "default", true},
		{"the in-repo placement", "in-repo", true},
	})

	// The `## Principles` summary must agree with step 8. An LLM reads
	// Principles before reaching the workflow steps, so a stale neutral-prompt
	// bullet there ("a prompt rather than picking on the operator's behalf")
	// silently steers it back to the pre-M-0190 behavior — a contradiction the
	// step-8-scoped check above cannot see (different section). Pin it: the
	// Principles worktree bullet now states the in-repo default. (M-0229/AC-3
	// dropped the ADR-0023 citation marker — the id-bearing doc-link is gone;
	// the in-repo/default behavioral markers pin the bullet.)
	principles := extractMarkdownSection(body, 2, "Principles")
	if principles == "" {
		t.Fatal("AC-1: start-epic must contain a `## Principles` section")
	}
	assertMarkers(t, "AC-1: start-epic Principles worktree bullet", principles, []marker{
		{"the in-repo default", "in-repo", true},
		{"the default framing", "default", true},
	})
}

// TestStartRituals_M0190_AC2_OverrideRetained pins M-0190/AC-2: both start
// rituals keep the per-invocation override — in-repo is the default, not a
// lock. start-epic retains all three placements; start-milestone names the
// main-checkout / sibling override. Heading-scoped to each ritual's worktree
// guidance.
func TestStartRituals_M0190_AC2_OverrideRetained(t *testing.T) {
	t.Parallel()

	epic := findWorktreePromptSection(loadAiwfxStartEpicFixture(t))
	if epic == "" {
		t.Fatal("AC-2: start-epic must contain a `### …worktree…` subsection")
	}
	assertMarkers(t, "AC-2: start-epic worktree subsection", epic, []marker{
		{"the override framing", "override", true},
		{"the no-worktree (main checkout) placement", "no worktree", true},
		{"the in-repo placement", ".claude/worktrees", false},
		{"the sibling placement", "../aiwf-", false},
	})

	ms := findStartMilestoneCutSection(loadAiwfxStartMilestoneFixture(t))
	if ms == "" {
		t.Fatal("AC-2: start-milestone must contain a `### …cut…` subsection (step 5)")
	}
	assertMarkers(t, "AC-2: start-milestone cut subsection", ms, []marker{
		{"the override framing", "override", true},
		{"the main-checkout override", "main-checkout", true},
		{"the sibling override", "sibling", true},
	})
}

// TestStartRituals_M0190_AC3_SandboxRationale pins M-0190/AC-3: both start
// rituals record the devcontainer-sandbox rationale for the in-repo default.
// Heading-scoped to each ritual's worktree guidance — the rationale must live
// with the placement guidance, not in an unrelated section; the rationale
// prose is matched case-insensitively. (M-0229/AC-3 dropped the ADR-0023
// citation marker: the shipped skill's id-bearing doc-link is gone, and the
// sandbox/devcontainer/rebuild behavioral markers pin the guidance.)
func TestStartRituals_M0190_AC3_SandboxRationale(t *testing.T) {
	t.Parallel()

	sections := []struct {
		ritual  string
		section string
	}{
		{"start-epic", findWorktreePromptSection(loadAiwfxStartEpicFixture(t))},
		{"start-milestone", findStartMilestoneCutSection(loadAiwfxStartMilestoneFixture(t))},
	}
	for _, s := range sections {
		if s.section == "" {
			t.Fatalf("AC-3: %s must contain its worktree-guidance subsection", s.ritual)
		}
		assertMarkers(t, "AC-3: "+s.ritual+" worktree guidance", s.section, []marker{
			{"the sandbox confinement rationale", "sandbox", true},
			{"the devcontainer context", "devcontainer", true},
			{"the container-rebuild loss rationale", "rebuild", true},
		})
	}
}
