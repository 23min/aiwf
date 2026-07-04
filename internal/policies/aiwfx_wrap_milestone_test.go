package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// aiwfx_wrap_milestone_test.go — M-0160 wrap cycle, G-0219 + G-0220
// follow-up.
//
// Structural drift-check pins for the aiwfx-wrap-milestone SKILL.md
// merge-step trailer prescription (the "Merge the milestone branch
// into the epic branch with a trailered merge commit" step). Mirrors
// internal/policies/aiwfx_wrap_epic_test.go's
// TestAiwfxWrapEpic_AC6_StructuralMergeStepDriftCheck pattern: walks
// the markdown heading hierarchy to locate the merge-step section,
// then asserts the trailer prescription appears INSIDE that section
// (not just somewhere in the file). Per CLAUDE.md §"Substring
// assertions are not structural assertions".
//
// The prescription itself landed at commit 5cf007f5 during M-0160
// wrap; this test is the mechanical backstop that prevents silent
// regression of the new content. Without this test, a future SKILL.md
// edit could drop the wrap-milestone trailer prescription and CI
// wouldn't notice — exactly the discipline gap G-0220 names.
//
// Discovered via G-0219 (the prescription asymmetry between
// aiwfx-wrap-milestone and aiwfx-wrap-epic) + G-0220 (ritual SKILL.md
// edits without structural AC pins have no mechanical backstop).

// aiwfxWrapMilestoneFixturePath is the canonical authoring location
// for the `aiwfx-wrap-milestone` skill body — the embedded ritual
// snapshot the aiwf binary ships. Per G-0182, AC content assertions
// read the embedded bytes directly rather than a duplicated fixture
// under internal/policies/testdata/.
const aiwfxWrapMilestoneFixturePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md"

// loadAiwfxWrapMilestoneFixture reads the fixture relative to repo
// root. The tests under this file are seam-tests against the
// authored skill body — they assert the doctrinal content G-0219's
// fix established, scoped to the relevant markdown section per
// CLAUDE.md *Testing* §"Substring assertions are not structural
// assertions".
func loadAiwfxWrapMilestoneFixture(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, aiwfxWrapMilestoneFixturePath))
	if err != nil {
		t.Fatalf("loading %s: %v", aiwfxWrapMilestoneFixturePath, err)
	}
	return string(data)
}

// TestAiwfxWrapMilestone_StructuralMergeStepDriftCheck pins the
// G-0219 fix structurally: the trailered-merge instructions live
// inside the merge-step section ("After merge"), not floating
// elsewhere. A reshuffle of the SKILL.md that moved the trailered
// commit into an unrelated section would fail this test even if a
// flat grep over the file still passed.
//
// Concretely: locate the merge-step subsection inside `## Workflow`
// (heading text references the merge action), then assert each
// required trailer flag appears *inside that subsection*. The
// trailer keys are quoted from CLAUDE.md §"Commit conventions"
// verbatim — variant casings (e.g., `Aiwf-Verb`) would fail the
// kernel's trailer-keys policy and must fail this test too.
//
// Mirrors TestAiwfxWrapEpic_AC6_StructuralMergeStepDriftCheck for
// the wrap-milestone surface; the two together pin the
// wrap-{epic,milestone} ritual symmetry G-0219 established.
func TestAiwfxWrapMilestone_StructuralMergeStepDriftCheck(t *testing.T) {
	t.Parallel()
	body := loadAiwfxWrapMilestoneFixture(t)

	// Step 1: the parent section exists.
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("SKILL.md must have a `## Workflow` section that contains the merge step as a subsection")
	}

	// Step 2: the merge-step subsection is reachable via a `### `
	// heading whose text references the merge action.
	merge := findWrapMilestoneMergeStepSection(body)
	if merge == "" {
		t.Fatal("`## Workflow` must contain a `### …merge…` subsection that documents the epic-integration merge")
	}

	// Step 3: each required trailer flag is named *inside* the
	// merge-step subsection. The keys are quoted from CLAUDE.md
	// §"Commit conventions" verbatim — variant casings would fail
	// the kernel's trailer-keys policy and must fail this test too.
	requiredTrailerFlags := []string{
		`--trailer "aiwf-verb: wrap-milestone"`,
		`--trailer "aiwf-entity: M-NNNN"`,
		`--trailer "aiwf-actor: human/`,
	}
	for _, flag := range requiredTrailerFlags {
		if !strings.Contains(merge, flag) {
			t.Errorf("merge-step subsection must name the trailer flag %q (in the right section, not just somewhere in the file)", flag)
		}
	}

	// Step 4: the trailered commit *follows* the staged merge.
	// Ordering matters — a fixture that documented the trailer
	// flags first and then a plain `git merge --no-ff` would
	// produce an untrailered commit at run time. Assert by index.
	stageIdx := strings.Index(merge, "git merge --no-ff --no-commit")
	commitIdx := strings.Index(merge, `--trailer "aiwf-verb: wrap-milestone"`)
	if stageIdx < 0 || commitIdx < 0 {
		t.Fatal("merge-step subsection must contain both the staged-merge command and the trailer-emitting commit")
	}
	if stageIdx > commitIdx {
		t.Error("the staged-merge (`--no-commit`) must appear *before* the trailered `git commit` so the commit-emitting step is the one carrying trailers")
	}

	// Step 5: Conventional Commits-shaped subject for the wrap
	// commit. The skill instruction's subject template
	// (chore(milestone): wrap M-NNNN — <title>) is what CLAUDE.md
	// §"Commit conventions" requires; assert by the leading shape.
	if !regexp.MustCompile(`chore\(milestone\):\s+wrap\s+M-NNNN`).MatchString(merge) {
		t.Error("merge-step subsection must use a Conventional Commits subject template `chore(milestone): wrap M-NNNN — <title>`")
	}
}

// findWrapMilestoneMergeStepSection returns the body of the
// merge-step subsection inside `## Workflow`. The subsection's
// `### ` heading is identified by a case-insensitive substring match
// on "merge". Returns "" if no matching subsection is found.
//
// A single-substring "merge" match is sufficient: the declared-
// sequence gate step's own heading ("Declared-sequence gate — close
// the milestone") deliberately avoids the word "merge" so it can't be
// confused with the merge-action step ("Merge the milestone branch
// into the epic branch with a trailered merge commit") — the
// wrap-milestone skill's only `###` heading containing "merge" IS the
// relevant one.
func findWrapMilestoneMergeStepSection(body string) string {
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		return ""
	}
	lines := strings.Split(workflow, "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.TrimPrefix(line, "### ")
		if strings.Contains(strings.ToLower(text), "merge") {
			return extractMarkdownSection(body, 3, text)
		}
	}
	return ""
}

// TestAiwfxWrapMilestone_ReconcileEpicBranchBeforeMerge pins the
// reconcile-first practice scoped to this ritual's integration
// target — the EPIC branch, not mainline (a milestone wrap merges
// into the epic branch, never mainline directly). The reconcile step
// lives as its own `### ` subsection inside `## Workflow` (G-0359: not
// folded into the merge step's own prose), positioned BEFORE the
// merge subsection: before the milestone-to-epic merge, if the epic
// branch has advanced past the milestone branch's fork point, the
// epic branch must be integrated into the milestone branch and the
// full local gate re-run there — never resolved on the epic branch
// itself, mid-merge.
//
// Structural per CLAUDE.md *Substring assertions are not structural
// assertions*: the reconcile step must exist as its own subsection,
// ordered before the merge subsection, with the guard and the
// integrate-and-re-gate instruction in the right order inside it.
func TestAiwfxWrapMilestone_ReconcileEpicBranchBeforeMerge(t *testing.T) {
	t.Parallel()
	body := loadAiwfxWrapMilestoneFixture(t)

	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("SKILL.md must have a `## Workflow` section")
	}
	reconcileIdx, mergeIdx := -1, -1
	for i, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.ToLower(strings.TrimPrefix(line, "### "))
		switch {
		case reconcileIdx < 0 && strings.Contains(text, "reconcile") && strings.Contains(text, "epic branch"):
			reconcileIdx = i
		case mergeIdx < 0 && strings.Contains(text, "merge") && strings.Contains(text, "epic branch"):
			mergeIdx = i
		}
	}
	if reconcileIdx < 0 {
		t.Fatal("`## Workflow` must contain a `### …Reconcile…epic branch…` step")
	}
	if mergeIdx < 0 {
		t.Fatal("`## Workflow` must contain a `### …Merge…epic branch…` step")
	}
	if reconcileIdx >= mergeIdx {
		t.Errorf("the reconcile step must appear BEFORE the merge step in `## Workflow` (reconcile at line %d, merge at line %d), so the merge that follows is already-validated", reconcileIdx, mergeIdx)
	}

	reconcile := extractMarkdownSection(body, 3, "11. Reconcile")
	if reconcile == "" {
		t.Fatal("could not extract the `### 11. Reconcile the milestone branch with the epic branch` section")
	}

	wantGuard := "git merge-base --is-ancestor epic/E-NNNN-<slug> milestone/M-NNNN-<slug>"
	if !strings.Contains(reconcile, wantGuard) {
		t.Errorf("reconcile step must name the ancestor guard %q, scoped to the epic branch as the integration target", wantGuard)
	}

	guardIdx := strings.Index(reconcile, wantGuard)
	integrateIdx := strings.Index(reconcile, "Integrate the epic branch into the milestone branch")
	gateIdx := strings.Index(reconcile, "re-run the full local CI gate")
	if integrateIdx < 0 || gateIdx < 0 {
		t.Fatal("reconcile step must document integrate-the-epic-branch and re-run-the-gate")
	}
	if guardIdx >= integrateIdx || integrateIdx >= gateIdx {
		t.Errorf("reconcile step must order the ancestor guard -> integrate epic branch -> re-run gate (got indices guard=%d, integrate=%d, gate=%d)", guardIdx, integrateIdx, gateIdx)
	}
}

// TestAiwfxWrapMilestone_RoadmapRegenWriteOnlyAfterPromote pins G-0350's
// fix: the ritual's own `### ` step sequence inside `## Workflow` places
// "Regenerate the roadmap" strictly between "Promote the milestone to
// `done`" and "Local cleanup" (so it captures the milestone's actual
// `done` status rather than a stale pre-promote snapshot), and that
// step's content documents `--write` no longer commits and hand-composes
// its own commit — because it's landing after the status-flip promote
// commit — rather than relying on `aiwf render roadmap --write` to
// commit for it. A regression that dropped the hand-composed commit
// (reverting to the old commits-automatically verb contract) or
// reordered the step before the promote would fail this test even
// though a flat grep for "render roadmap" would still pass. Mirrors
// TestAiwfxWrapMilestone_ReconcileEpicBranchBeforeMerge's heading-walk
// shape (G-0359 restructured this ritual's sub-steps into their own
// numbered `### ` headings rather than nested bold sub-items).
func TestAiwfxWrapMilestone_RoadmapRegenWriteOnlyAfterPromote(t *testing.T) {
	t.Parallel()
	body := loadAiwfxWrapMilestoneFixture(t)

	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("SKILL.md must have a `## Workflow` section")
	}
	promoteIdx, regenIdx, cleanupIdx := -1, -1, -1
	for i, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		lower := strings.ToLower(strings.TrimPrefix(line, "### "))
		switch {
		case promoteIdx < 0 && strings.Contains(lower, "promote the milestone to `done`"):
			promoteIdx = i
		case regenIdx < 0 && strings.Contains(lower, "regenerate the roadmap"):
			regenIdx = i
		case cleanupIdx < 0 && strings.Contains(lower, "local cleanup"):
			cleanupIdx = i
		}
	}
	if promoteIdx < 0 {
		t.Fatal("`## Workflow` must contain a `### …Promote the milestone to `done`…` step")
	}
	if regenIdx < 0 {
		t.Fatal("`## Workflow` must contain a `### …Regenerate the roadmap` step")
	}
	if cleanupIdx < 0 {
		t.Fatal("`## Workflow` must contain a `### …Local cleanup` step")
	}
	if promoteIdx >= regenIdx || regenIdx >= cleanupIdx {
		t.Errorf("roadmap-regen step must land after promote-done and before local cleanup (got line indices: promote=%d, regen=%d, cleanup=%d)", promoteIdx, regenIdx, cleanupIdx)
	}

	regen := extractMarkdownSection(body, 3, "14. Regenerate the roadmap")
	if regen == "" {
		t.Fatal("could not extract the `### 14. Regenerate the roadmap` section")
	}
	if !strings.Contains(regen, "aiwf render roadmap --write") {
		t.Error("roadmap-regen step must run `aiwf render roadmap --write`")
	}
	if !strings.Contains(regen, "never commits") {
		t.Error("roadmap-regen step must document that --write never commits (G-0350)")
	}
	requiredTrailerFlags := []string{
		`--trailer "aiwf-verb: wrap-milestone"`,
		`--trailer "aiwf-entity: M-NNNN"`,
		`--trailer "aiwf-actor: human/`,
	}
	for _, flag := range requiredTrailerFlags {
		if !strings.Contains(regen, flag) {
			t.Errorf("roadmap-regen step must hand-compose the commit with trailer flag %q", flag)
		}
	}
}
