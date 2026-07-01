package policies

import (
	"strings"
	"testing"
)

// findNumberedStep returns the body of the flat `## Workflow` step whose
// bolded heading contains keyword (case-insensitive), or "" if none. The
// aiwfx-plan-epic and aiwfx-plan-milestones skills write their workflow as a
// flat numbered list (`1.`, `2.`, … under `## Workflow`, each `N. **Title.**
// body`), so this parameterized locator finds the body-fill step across both
// skills. Mirrors findDependsOnStep's fence-aware walk but keyed by an
// arbitrary heading keyword rather than the hardcoded "depend".
//
// A step runs from its `N. ` line up to (but not including) the next
// column-0 `N. ` step or the end of `## Workflow`. Fenced code blocks are
// skipped so a code-comment or example line inside a ```bash block neither
// matches as a step start nor terminates the step prematurely.
func findNumberedStep(body, keyword string) string {
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		return ""
	}
	lines := strings.Split(workflow, "\n")
	stepStart := -1
	inFence := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if !isNumberedStepStart(line) {
			continue
		}
		text := strings.TrimLeft(line, "0123456789")
		text = strings.TrimPrefix(text, ". ")
		boldTitle := extractBoldedHeading(text)
		if strings.Contains(strings.ToLower(boldTitle), strings.ToLower(keyword)) {
			stepStart = i
			break
		}
	}
	if stepStart == -1 {
		return ""
	}
	end := len(lines)
	inFence = false
	for i := stepStart + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if isNumberedStepStart(lines[i]) {
			end = i
			break
		}
	}
	return strings.Join(lines[stepStart:end], "\n")
}

// TestFindNumberedStep_BranchCoverage exercises every reachable branch of
// findNumberedStep against synthetic inputs (missing workflow, no matching
// step, fence not confused for a step, happy path with correct termination).
func TestFindNumberedStep_BranchCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		body     string
		keyword  string
		wantHas  string
		wantNone bool
	}{
		{
			name:     "missing-workflow",
			body:     "no headings here",
			keyword:  "replace",
			wantNone: true,
		},
		{
			name:     "workflow-without-matching-step",
			body:     "## Workflow\n\n1. **Alpha.** body\n2. **Beta.** body\n",
			keyword:  "replace",
			wantNone: true,
		},
		{
			name:    "fenced-numbered-line-not-confused-for-step",
			body:    "## Workflow\n\n1. **Intro.** body\n\n   ```bash\n   1. not-a-step\n   ```\n\n2. **Replace the body.** here\n",
			keyword: "replace",
			wantHas: "Replace the body",
		},
		{
			name:    "happy-path-terminates-at-next-step",
			body:    "## Workflow\n\n5. **Replace the body with the rich template.** fill\n\n6. **Next thing.** more\n",
			keyword: "rich template",
			wantHas: "Replace the body",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := findNumberedStep(tc.body, tc.keyword)
			if tc.wantNone {
				if got != "" {
					t.Errorf("findNumberedStep(%q) = %q; want empty", tc.name, got)
				}
				return
			}
			if !strings.Contains(got, tc.wantHas) {
				t.Errorf("findNumberedStep(%q) = %q; want it to contain %q", tc.name, got, tc.wantHas)
			}
			if tc.name == "happy-path-terminates-at-next-step" && strings.Contains(got, "Next thing") {
				t.Errorf("findNumberedStep(%q): step body leaked into the next step (got %q)", tc.name, got)
			}
		})
	}
}

// TestPlanningBodyFill_AC1_PlanSkillsRouteThroughEditBody pins AC-1 for the
// two planning skills: the body-fill step (step 5, "Replace … the rich
// template") in aiwfx-plan-epic and aiwfx-plan-milestones routes through
// `aiwf edit-body` (the trailered route) instead of an unspecified /
// plain-commit path. Scoped to the body-fill step — not a flat body match —
// because aiwfx-plan-milestones already names `aiwf edit-body` in a later
// step (the frontmatter-hand-edit warning), so a flat assertion would pass
// vacuously without the step-5 edit.
func TestPlanningBodyFill_AC1_PlanSkillsRouteThroughEditBody(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		path string
	}{
		{"plan-epic", aiwfxPlanEpicFixturePath},
		{"plan-milestones", aiwfxPlanMilestonesFixturePath},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := loadPolishFixture(t, tc.path)
			step := findNumberedStep(body, "rich template")
			if step == "" {
				t.Fatalf("AC-1: %s must retain a `## Workflow` body-fill step (heading names the rich template)", tc.name)
			}
			if !strings.Contains(step, "aiwf edit-body") {
				t.Errorf("AC-1: %s body-fill step must land the body via `aiwf edit-body` (the trailered route)", tc.name)
			}
		})
	}
}

// TestPlanningNextStep_AC3_StatusAwareRouting pins AC-3: the planning skills'
// `## Next step` routing is status-aware and does not leapfrog
// aiwfx-start-epic. aiwfx-plan-milestones must route to aiwfx-start-epic for
// a still-proposed epic (naming both the skill and the proposed/active
// condition); aiwfx-plan-epic's own Next-step must likewise name
// aiwfx-start-epic rather than jumping straight to start-milestone.
func TestPlanningNextStep_AC3_StatusAwareRouting(t *testing.T) {
	t.Parallel()

	t.Run("plan-milestones", func(t *testing.T) {
		t.Parallel()
		body := loadPolishFixture(t, aiwfxPlanMilestonesFixturePath)
		section := extractMarkdownSection(body, 2, "Next step")
		if section == "" {
			t.Fatal("AC-3: aiwfx-plan-milestones must have a `## Next step` section")
		}
		if !strings.Contains(section, "aiwfx-start-epic") {
			t.Error("AC-3: plan-milestones `## Next step` must route to `aiwfx-start-epic` for a still-proposed epic (no leapfrog to start-milestone)")
		}
		if !strings.Contains(strings.ToLower(section), "proposed") {
			t.Error("AC-3: plan-milestones `## Next step` must be status-aware — name the `proposed` epic case that routes to start-epic")
		}
	})

	t.Run("plan-epic", func(t *testing.T) {
		t.Parallel()
		body := loadPolishFixture(t, aiwfxPlanEpicFixturePath)
		section := extractMarkdownSection(body, 2, "Next step")
		if section == "" {
			t.Fatal("AC-3: aiwfx-plan-epic must have a `## Next step` section")
		}
		if !strings.Contains(section, "aiwfx-start-epic") {
			t.Error("AC-3: plan-epic `## Next step` must route through `aiwfx-start-epic` rather than leapfrogging to start-milestone")
		}
	})
}
