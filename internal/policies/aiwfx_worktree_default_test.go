package policies

import (
	"strings"
	"testing"
)

// The start rituals keep the per-invocation worktree override. The fixture
// loaders and section helpers live in aiwfx_start_epic_test.go and
// aiwfx_start_milestone_test.go. Host-fragment selection and generated Claude
// compatibility are checked in the skills and CLI integration packages.

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
