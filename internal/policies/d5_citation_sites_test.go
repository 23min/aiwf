package policies

import (
	"strings"
	"testing"
)

// Structural tests for the surfaces that APPLY the D5 findings-become-checks
// force at the point a finding is disposed of (G-0489): the milestone wrap's
// review loop, the patch ritual's implement step, the milestone-planning scope
// fork, and the reviewer agent card.
//
// The canonical statement lives in wf-codebase-health D5 and is pinned by
// d5_findings_become_checks_test.go. These tests pin that each caller CITES it
// and states its own application — not that each caller restates the rule,
// which is the drift wf-codebase-health §"Relation to other skills" rules out
// ("They cross-reference; they don't duplicate"). So an assertion here targets
// the citation and the caller-specific specialization, never a phrase copied
// from D5 itself: pinning the copy would enforce the duplication.
//
// EVERY asserted literal must be one this change introduced. These sections are
// long pre-existing documents, so a plausible-sounding phrase is often already
// present a few lines up — an assertion matching it passes while pinning
// nothing, and still reds under a whole-block revert, which hides the hole.
// "over the milestone's full change-set" is the live example: it reads like the
// terminator's core claim and already appears in the step's opening dispatch.
// Verify a candidate literal is absent from the file at HEAD before asserting
// it, and prefer a contiguous phrase long enough to be unambiguous.
//
// Path literals for the three SKILL.md files are the shared consts declared by
// sibling policy tests, reused rather than redeclared; those declarations also
// discharge the skill-edit-structural-test-backstop for these edits.

// d5ReviewerAgentCardPath is the shipped reviewer role card. It is an agent
// card rather than a SKILL.md, so the skill-edit backstop does not cover it;
// this const is the only path literal for it in the policy suite.
const d5ReviewerAgentCardPath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/agents/reviewer.md"

// TestWrapMilestone_CorrectiveCommitCarriesAPinningCheck pins that the wrap's
// review loop applies the ratchet. Without it the wrap says only "fix every
// blocking finding", the disposition that leaves the project's checks the same
// size and lets the class return at the next milestone's review.
func TestWrapMilestone_CorrectiveCommitCarriesAPinningCheck(t *testing.T) {
	t.Parallel()
	step := readWrapMilestoneReviewStep(t)

	if !strings.Contains(step, "carries the check that pins it") {
		t.Error("the wrap's corrective-commit instruction must require the fix to carry the check that pins it")
	}
	if !strings.Contains(step, "`wf-codebase-health` D5") {
		t.Error("the wrap must cite wf-codebase-health D5 rather than restate the ratchet in its own wording")
	}
}

// TestWrapMilestone_RoutesBothFindingKindsToADurableSink pins the disposition
// routes. A defect left unpinned and a finding revealing a requirement no AC
// covers go to a gap, mirrored so the milestone can reach it; an accepted
// judgment finding becomes a rule or a recorded decision. Without these the
// findings survive only as prose in the Reviewer notes.
//
// Asserted as contiguous phrases: `aiwf add gap` alone already appears in this
// step's stale-TODO bullet, so the bare verb pins nothing.
func TestWrapMilestone_RoutesBothFindingKindsToADurableSink(t *testing.T) {
	t.Parallel()
	step := readWrapMilestoneReviewStep(t)

	if !strings.Contains(step, "becomes a gap (`aiwf add gap") {
		t.Error("the wrap must route an unpinned defect or an uncovered requirement to a gap, not merely mention the verb")
	}
	if !strings.Contains(step, "reveals a requirement no AC covers") {
		t.Error("the wrap must name the uncovered-requirement finding as one of the gap's sources")
	}
	if !strings.Contains(step, "mirrored under the spec") {
		t.Error("the gap id must be mirrored into the spec, or it is reachable only through --discovered-in")
	}
	if !strings.Contains(step, "aiwfx-record-decision") {
		t.Error("an accepted judgment finding must route to a recorded decision — D5's judgment half, not only its defect half")
	}
}

// TestWrapMilestone_ReviewLoopEndsOnAFullSurfacePass pins the terminator on the
// caller that previously had none: the wrap confirmed fixes with a reviewer
// scoped to the changed surface and then proceeded, while D5's stop rule says a
// scoped pass cannot declare convergence.
//
// The assertions are the citation and the two caller-specific clauses. The
// phrase that reads like the terminator's core claim — "over the milestone's
// full change-set" — is deliberately NOT asserted: it already appears in this
// step's opening dispatch, so it would pass with the terminator inverted.
func TestWrapMilestone_ReviewLoopEndsOnAFullSurfacePass(t *testing.T) {
	t.Parallel()
	step := readWrapMilestoneReviewStep(t)

	if !strings.Contains(step, `§"When the loop ends"`) {
		t.Fatal("the wrap must cite wf-review-code §\"When the loop ends\" for its terminator rather than restate the stop rule")
	}
	if !strings.Contains(step, "narrowed to what the last fix touched") {
		t.Error("the terminator must exclude a pass narrowed to the last fix's footprint — the shape the wrap previously stopped at")
	}
	if !strings.Contains(step, "provided the slices together cover") {
		t.Error("the terminator must reconcile with this step's slice-for-depth instruction, or the two read as contradictory")
	}
}

// TestWfPatch_DefectFixLandsWithItsPinningCheck pins the ratchet on the patch
// ritual, where the inversion was sharpest: the CHANGELOG entry was mandatory
// with no skip while the regression test was only a judgment-gated escalation,
// so a defect could land documented and unpinned.
func TestWfPatch_DefectFixLandsWithItsPinningCheck(t *testing.T) {
	t.Parallel()
	step := normalizeProse(readWfPatchSection(t, 3, "3. Implement the change"))

	if !strings.Contains(step, "fails without the fix") {
		t.Error("wf-patch's implement step must require a defect fix to land the check that fails without it")
	}
	if !strings.Contains(step, "`wf-codebase-health` D5") {
		t.Error("wf-patch must cite wf-codebase-health D5 rather than restate the ratchet")
	}
	if !strings.Contains(strings.ToLower(step), "same footing as the changelog") {
		t.Error("wf-patch must put the pinning check on the same footing as the mandatory CHANGELOG entry")
	}
}

// TestWfPatch_PinningConstraintNamesItsTwoEscapes pins the constraint that
// makes the requirement binding, together with the escapes that make it
// workable. A pin mandated with no escape is absurd for a typo fix, and an
// unworkable rule gets ignored wholesale rather than selectively. The recording
// obligation is asserted because an escape whose second half is optional lets
// the defect be fixed silently after all.
func TestWfPatch_PinningConstraintNamesItsTwoEscapes(t *testing.T) {
	t.Parallel()
	section := normalizeProse(readWfPatchSection(t, 2, "Constraints"))

	if !strings.Contains(section, "lands the check that pins it") {
		t.Error("wf-patch's constraints must bind the defect-fix pinning requirement")
	}
	low := strings.ToLower(section)
	if !strings.Contains(low, "no logic to pin") {
		t.Error("the pinning constraint must except a change with no logic to pin")
	}
	if !strings.Contains(low, "recorded in the project's tracker") {
		t.Error("the unpinned-defect escape must name a real sink, or the escape's second half evaporates and the fix is silent")
	}
}

// TestWfPatch_TDDEscalationIsNotAPinningExemption pins the correction to the
// bullet that previously read as one: "does not run a TDD cycle" invited the
// reading that a patch owes no test at all. Scoped to the bullet's own line —
// the section-wide form passes with the clause moved to an unrelated sibling
// bullet.
func TestWfPatch_TDDEscalationIsNotAPinningExemption(t *testing.T) {
	t.Parallel()
	section := readWfPatchSection(t, 2, "What this skill explicitly does not do")

	line := lineContaining(section, "Does not run a full TDD cycle")
	if line == "" {
		t.Fatal("wf-patch must keep a `Does not run a full TDD cycle` bullet in its does-not-do list")
	}
	if !strings.Contains(strings.ToLower(normalizeProse(line)), "not an exemption from pinning") {
		t.Error("the TDD-escalation bullet must state on its own line that it is not an exemption from pinning, or it reads as one")
	}
}

// TestWfPatch_AntiPatternRejectsDeferringTheRegressionTest pins the bullet that
// names the ratchet's most common evasion. The constraint states the rule; this
// bullet is what a reader recognizes their own intent in, and it was previously
// revertible with the whole suite green.
func TestWfPatch_AntiPatternRejectsDeferringTheRegressionTest(t *testing.T) {
	t.Parallel()
	section := readWfPatchSection(t, 2, "Anti-patterns")

	line := lineContaining(section, "regression test in a follow-up")
	if line == "" {
		t.Fatal("wf-patch's anti-patterns must reject deferring the regression test to a follow-up")
	}
	if !strings.Contains(strings.ToLower(normalizeProse(line)), "land the check with the fix") {
		t.Error("the follow-up anti-pattern must state the corrective action, not only name the smell")
	}
}

// TestPlanMilestones_ScopeForkHandsOffToRecordDecision pins the capture route
// at the fork where milestone planning surfaces work the epic didn't scope.
// Left uncaptured, that ambiguity is rediscovered during implementation as a
// review finding — divergence manufactured at plan time. Scoped to the bullet's
// own line for the same reason as the TDD-escalation test.
func TestPlanMilestones_ScopeForkHandsOffToRecordDecision(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, aiwfxPlanMilestonesFixturePath)

	section := extractMarkdownSection(body, 2, "Anti-patterns")
	if section == "" {
		t.Fatal("aiwfx-plan-milestones must have an `## Anti-patterns` section")
	}
	line := lineContaining(section, "Scope creep mid-decomposition")
	if line == "" {
		t.Fatal("aiwfx-plan-milestones must keep the scope-creep anti-pattern that names the amend-vs-defer fork")
	}
	if !strings.Contains(normalizeProse(line), "aiwfx-record-decision") {
		t.Error("the amend-vs-defer fork must hand off to aiwfx-record-decision on its own line, or the reasoning is lost and re-derived later")
	}
}

// TestReviewerAgentCard_CarriesTheKindConstraint pins the propagation to the
// shipped role card. The card restates wf-review-code's constraints for an
// agent that may never open the skill, so a constraint present in the skill and
// absent here is invisible to exactly the reader it governs. The citation is
// asserted by section name: wf-review-code's steps are numbered, so a numeric
// reference silently rots when a step is inserted.
func TestReviewerAgentCard_CarriesTheKindConstraint(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, d5ReviewerAgentCardPath)

	section := extractMarkdownSection(body, 2, "Constraints")
	if section == "" {
		t.Fatal("the reviewer agent card must have a `## Constraints` section")
	}
	flat := normalizeProse(section)
	low := strings.ToLower(flat)
	if !strings.Contains(low, "every finding carries its kind") {
		t.Error("the reviewer card must require every finding to carry its kind — defect or judgment")
	}
	if !strings.Contains(low, "the decision not to pin it is recorded") {
		t.Error("the reviewer card's kind constraint must preserve the unpinnable-defect escape")
	}
	if !strings.Contains(flat, `§"Verdict"`) {
		t.Error("the reviewer card must cite wf-review-code by section name; a numeric step reference rots when a step is inserted")
	}
}

// readWrapMilestoneReviewStep returns the wrap ritual's independent-review step
// normalized for prose matching, failing the test if the step is gone.
func readWrapMilestoneReviewStep(t *testing.T) string {
	t.Helper()
	body := readVerbSkill(t, aiwfxWrapMilestoneFixturePath)
	step := extractMarkdownSection(body, 3, "2. Independent two-lens review")
	if step == "" {
		t.Fatal("aiwfx-wrap-milestone must have its `### 2. Independent two-lens review` step")
	}
	return normalizeProse(step)
}

// readWfPatchSection returns a named wf-patch section with line structure
// intact; callers normalize the whole section or a single line as their
// granularity requires.
func readWfPatchSection(t *testing.T, level int, heading string) string {
	t.Helper()
	section := extractMarkdownSection(readVerbSkill(t, wfPatchFixturePath), level, heading)
	if section == "" {
		t.Fatalf("wf-patch must have a %q section", heading)
	}
	return section
}
