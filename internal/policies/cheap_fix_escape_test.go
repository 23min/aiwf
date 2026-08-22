package policies

import (
	"strings"
	"testing"
)

// cheap_fix_escape_test.go — structural pins for the cheap-fix test and the
// reference-phrasing rule.
//
// Both rules narrow a standing mandate rather than adding one, which is what
// makes them worth pinning: a mandate that loses its escape reverts silently to
// the unconditional form, and nothing else in the tree would notice. Each
// assertion is scoped to the named markdown section that must carry the rule,
// per CLAUDE.md *Testing* §"Substring assertions are not structural
// assertions" — the literals here ("cheap-fix test") are short enough that
// file-wide presence would prove nothing about placement.
//
// The guidance fragment's own copies are additionally held by
// PolicyM0211GuidanceOperatingAnchors' curated anchor set; these tests pin the
// ritual surfaces, which that policy does not read.

// findStartMilestoneImplementationSection locates the implementation step by
// heading content rather than by its number, matching the convention its sibling
// finders in this package document: a future reshuffle that renumbers the step
// must not silently stop the structural check from finding anything.
func findStartMilestoneImplementationSection(body string) string {
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		return ""
	}
	for _, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.TrimPrefix(line, "### ")
		if strings.Contains(strings.ToLower(text), "implementation") {
			return extractMarkdownSection(body, 3, text)
		}
	}
	return ""
}

// TestFindStartMilestoneImplementationSection_BranchCoverage covers the
// defensive return arms the happy-path fixture test does not reach.
func TestFindStartMilestoneImplementationSection_BranchCoverage(t *testing.T) {
	t.Parallel()

	if got := findStartMilestoneImplementationSection("# No workflow heading here\n"); got != "" {
		t.Errorf("absent Workflow section: want empty, got %q", got)
	}
	if got := findStartMilestoneImplementationSection("## Workflow\n\n### 1. Something else\n\nbody\n"); got != "" {
		t.Errorf("Workflow without a matching step: want empty, got %q", got)
	}
}

// findWrapMilestoneSpecSectionsStep returns the wrap-side spec-sections step of
// the wrap-milestone workflow — the step whose `## Deferrals` bullet decides
// whether a punt becomes a gap entity.
func findWrapMilestoneSpecSectionsStep(body string) string {
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		return ""
	}
	for _, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.TrimPrefix(line, "### ")
		if strings.Contains(strings.ToLower(text), "wrap-side sections") {
			return extractMarkdownSection(body, 3, text)
		}
	}
	return ""
}

// TestFindWrapMilestoneSpecSectionsStep_BranchCoverage covers the defensive
// return arms the happy-path fixture test does not reach.
func TestFindWrapMilestoneSpecSectionsStep_BranchCoverage(t *testing.T) {
	t.Parallel()

	if got := findWrapMilestoneSpecSectionsStep("# No workflow heading here\n"); got != "" {
		t.Errorf("absent Workflow section: want empty, got %q", got)
	}
	if got := findWrapMilestoneSpecSectionsStep("## Workflow\n\n### 1. Something else\n\nbody\n"); got != "" {
		t.Errorf("Workflow without a matching step: want empty, got %q", got)
	}
}
