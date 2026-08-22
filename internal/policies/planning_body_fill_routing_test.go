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
// findNumberedStep against synthetic inputs alone (no reliance on the live
// fixtures): missing workflow, no matching step, a fence before the match
// (first-loop fence-skip), a fence inside the matched step's body
// (second-loop fence-skip), and the happy path with correct termination.
func TestFindNumberedStep_BranchCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		body         string
		keyword      string
		wantHas      string
		wantExcludes string
		wantNone     bool
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
			// Fence BEFORE the matched step exercises the first
			// (locate) loop's fence-skip: the `1. not-a-step` line
			// inside step 1's fence must not be mistaken for a step.
			name:    "fenced-numbered-line-before-match-first-loop",
			body:    "## Workflow\n\n1. **Intro.** body\n\n   ```bash\n   1. not-a-step\n   ```\n\n2. **Replace the body.** here\n",
			keyword: "replace",
			wantHas: "Replace the body",
		},
		{
			// Fence INSIDE the matched step's body exercises the
			// second (termination) loop's fence-skip: the fenced
			// `6. not-a-real-step` line must not terminate step 5 —
			// only the real step 6 does.
			name:         "fenced-numbered-line-inside-match-second-loop",
			body:         "## Workflow\n\n5. **Replace the rich template.** intro\n\n   ```bash\n   6. not-a-real-step\n   ```\n\n   tail\n\n6. **Next thing.** more\n",
			keyword:      "rich template",
			wantHas:      "not-a-real-step",
			wantExcludes: "Next thing",
		},
		{
			name:         "happy-path-terminates-at-next-step",
			body:         "## Workflow\n\n5. **Replace the body with the rich template.** fill\n\n6. **Next thing.** more\n",
			keyword:      "rich template",
			wantHas:      "Replace the body",
			wantExcludes: "Next thing",
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
			if tc.wantExcludes != "" && strings.Contains(got, tc.wantExcludes) {
				t.Errorf("findNumberedStep(%q): step body leaked past its terminator (got %q; must exclude %q)", tc.name, got, tc.wantExcludes)
			}
		})
	}
}
