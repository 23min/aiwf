package policies

import (
	"regexp"
	"strings"
	"testing"
)

// Cross-document reference checks over the embedded rituals: a reference in
// one document resolved against its target in another. The needle comes from
// the citing document rather than from this test, which is what puts the class
// outside D-0070's predicate entirely — no rewording of either side satisfies
// it falsely, and drift on either side goes red.
//
// Its sibling `TestEmbeddedRituals_CrossSkillCitationsResolve` covers the
// named-section form, `` `skill` §"Section" ``. This covers the numbered-step
// form, which that walk cannot see.

const (
	wfRethinkFixturePath = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-rethink/SKILL.md"
	wfPatchFixturePath   = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md"
)

// wfPatchStepCitation matches wf-rethink's reference to a numbered wf-patch
// step — "`wf-patch` — at its independent-review step (step 6)".
var wfPatchStepCitation = regexp.MustCompile("`wf-patch` — at its .+ \\(step (\\d+)\\)")

// TestWfRethink_WfPatchStepCitationResolves reads the step number out of
// wf-rethink and resolves it against wf-patch's live headings. A citation by
// number rots silently when a step is inserted above it — G-0365 inserted a
// step 4 and renumbered everything after — so the number is never pinned as a
// literal here; it is extracted and looked up.
func TestWfRethink_WfPatchStepCitationResolves(t *testing.T) {
	t.Parallel()

	rethink := readVerbSkill(t, wfRethinkFixturePath)
	trigger := extractMarkdownSection(rethink, 2, "The non-trivial-design trigger")
	if trigger == "" {
		t.Fatal("wf-rethink must have a `## The non-trivial-design trigger` section carrying the wf-patch cross-reference")
	}
	m := wfPatchStepCitation.FindStringSubmatch(trigger)
	if m == nil {
		t.Fatal("the trigger section must cite a wf-patch step in the form `(step N)`; nothing to resolve")
	}
	stepNum := m[1]

	patch := readVerbSkill(t, wfPatchFixturePath)
	workflow := extractMarkdownSection(patch, 2, "Workflow")
	if workflow == "" {
		t.Fatal("wf-patch must have a `## Workflow` section")
	}

	headingLine := ""
	for _, line := range strings.Split(workflow, "\n") {
		if strings.HasPrefix(line, "### "+stepNum+". ") {
			headingLine = line
			break
		}
	}
	if headingLine == "" {
		t.Fatalf("wf-rethink cites wf-patch step %s, but wf-patch has no `### %s.` step — the cross-reference points nowhere", stepNum, stepNum)
	}

	// Resolving to *a* step is not enough: the citation claims a particular
	// step dispatches wf-rethink, and renumbering can leave it resolving to
	// the wrong one. The label the citing document itself uses is the
	// expectation, so this stays a comparison between the two documents.
	claimed := strings.TrimSpace(wfPatchStepCitation.FindStringSubmatch(trigger)[0])
	claimedLabel := strings.TrimSuffix(strings.SplitN(claimed, "at its ", 2)[1], " (step "+stepNum+")")
	claimedLabel = strings.TrimSuffix(claimedLabel, " step")
	// The citing side hyphenates a compound the heading writes open
	// ("independent-review" against "Independent review"), so compare with
	// hyphens folded to spaces rather than pinning either spelling.
	fold := func(s string) string { return strings.ToLower(strings.ReplaceAll(s, "-", " ")) }
	if !strings.Contains(fold(headingLine), fold(claimedLabel)) {
		t.Errorf("wf-rethink cites wf-patch step %s as its %q step, but that step is %q — update the cross-reference",
			stepNum, claimedLabel, strings.TrimSpace(headingLine))
	}
}
