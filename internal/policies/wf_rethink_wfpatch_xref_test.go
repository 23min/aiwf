package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// wfRethinkFixturePath is the canonical authoring location for the
// `wf-rethink` skill body — the embedded ritual snapshot the aiwf
// binary ships.
const wfRethinkFixturePath = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-rethink/SKILL.md"

// loadWfRethinkFixture reads the fixture relative to repo root.
func loadWfRethinkFixture(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, wfRethinkFixturePath))
	if err != nil {
		t.Fatalf("loading %s: %v", wfRethinkFixturePath, err)
	}
	return string(data)
}

// TestWfRethink_WfPatchCrossReferenceStepNumberIsLive asserts
// wf-rethink's "The non-trivial-design trigger" section names the
// step of wf-patch that actually dispatches it (its independent-
// review step), rather than a step number that drifts silently
// whenever wf-patch's own step numbering shifts (G-0365 inserted a
// new step 4, renumbering every step after it). The test resolves the
// referenced number against wf-patch's live heading text instead of
// pinning a literal, so it catches future renumbering drift too.
func TestWfRethink_WfPatchCrossReferenceStepNumberIsLive(t *testing.T) {
	t.Parallel()
	rethinkBody := loadWfRethinkFixture(t)

	trigger := extractMarkdownSection(rethinkBody, 2, "The non-trivial-design trigger")
	if trigger == "" {
		t.Fatal("wf-rethink must have a `## The non-trivial-design trigger` section")
	}

	re := regexp.MustCompile("`wf-patch` — at its .+ \\(step (\\d+)\\)")
	m := re.FindStringSubmatch(trigger)
	if m == nil {
		t.Fatal("trigger section must name a `wf-patch` step in the form `(step N)`")
	}
	stepNum := m[1]

	patchBody := loadWfPatchFixture(t)
	workflow := extractMarkdownSection(patchBody, 2, "Workflow")
	if workflow == "" {
		t.Fatal("wf-patch must have a `## Workflow` section")
	}
	step := extractMarkdownSection(patchBody, 3, stepNum+". ")
	if step == "" {
		t.Fatalf("wf-patch has no `### %s. …` step — wf-rethink's cross-reference (step %s) points nowhere", stepNum, stepNum)
	}

	// The named step must be the one that actually dispatches wf-rethink.
	headingLine := ""
	for line := range strings.SplitSeq(workflow, "\n") {
		if strings.HasPrefix(line, "### "+stepNum+". ") {
			headingLine = line
			break
		}
	}
	if !strings.Contains(strings.ToLower(headingLine), "independent review") {
		t.Errorf("wf-rethink names wf-patch step %s as its dispatch point, but that step is %q — the actual dispatch happens in wf-patch's independent-review step; update the cross-reference", stepNum, headingLine)
	}
	if !strings.Contains(step, "wf-rethink") {
		t.Errorf("wf-patch step %s (the step wf-rethink's cross-reference names) must itself mention `wf-rethink`", stepNum)
	}
}
