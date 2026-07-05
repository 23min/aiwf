package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// wf_patch_reconcile_test.go pins the reconcile-first practice in
// wf-patch's Workflow: before merging a patch branch to mainline, if
// mainline has advanced past the branch's fork point, mainline must
// be integrated into the patch branch first and the full local CI
// gate re-run there — never resolved on mainline itself, mid-merge,
// with the "gate green before merge" precondition passing vacuously
// against a tree that omits mainline's newer commits.
//
// The reconcile step lives as its own `### ` heading (G-0359: a dense
// paragraph nested inside the merge bullet was easy to skim and
// misremember at the point of use), mirroring
// aiwfx_wrap_epic_test.go's TestAiwfxWrapEpic_ReconcileMainlineBeforeMerge
// and aiwfx_wrap_milestone_test.go's
// TestAiwfxWrapMilestone_ReconcileEpicBranchBeforeMerge — all three
// rituals document the same reconcile-before-merge procedure, scoped
// to their own integration target (mainline for wf-patch and
// aiwfx-wrap-epic; the epic branch for aiwfx-wrap-milestone).

// wfPatchFixturePath is the canonical authoring location for the
// `wf-patch` skill body — the embedded ritual snapshot the aiwf
// binary ships. Per G-0182, AC content assertions read the embedded
// bytes directly rather than a duplicated fixture under
// internal/policies/testdata/.
const wfPatchFixturePath = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md"

// loadWfPatchFixture reads the fixture relative to repo root.
func loadWfPatchFixture(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, wfPatchFixturePath))
	if err != nil {
		t.Fatalf("loading %s: %v", wfPatchFixturePath, err)
	}
	return string(data)
}

// TestWfPatch_ReconcileMainlineBeforeMerge asserts the reconcile step
// exists as its own `### ` subsection inside `## Workflow`, positioned
// BEFORE the merge step, and orders its content: the fetch/fast-
// forward preamble, then the ancestor guard, then the integrate-and-
// re-gate instruction. Per CLAUDE.md *Substring assertions are not
// structural assertions*, every assertion is scoped to the reconcile
// step's own section, not the file as a whole.
func TestWfPatch_ReconcileMainlineBeforeMerge(t *testing.T) {
	t.Parallel()
	body := loadWfPatchFixture(t)

	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		t.Fatal("wf-patch must have a `## Workflow` section")
	}

	reconcileIdx, mergeIdx := -1, -1
	for i, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.ToLower(strings.TrimPrefix(line, "### "))
		switch {
		case reconcileIdx < 0 && strings.Contains(text, "reconcile") && strings.Contains(text, "mainline"):
			reconcileIdx = i
		case mergeIdx < 0 && strings.Contains(text, "merge the patch branch"):
			mergeIdx = i
		}
	}
	if reconcileIdx < 0 {
		t.Fatal("`## Workflow` must contain a `### …Reconcile…mainline…` step")
	}
	if mergeIdx < 0 {
		t.Fatal("`## Workflow` must contain a `### …Merge the patch branch…` step")
	}
	if reconcileIdx >= mergeIdx {
		t.Errorf("the reconcile step must appear BEFORE the merge step in `## Workflow` (reconcile at line %d, merge at line %d), so the merge that follows is already-validated", reconcileIdx, mergeIdx)
	}

	reconcile := extractMarkdownSection(body, 3, "11. Reconcile")
	if reconcile == "" {
		t.Fatal("could not extract the `### 11. Reconcile mainline with the patch branch` section")
	}

	// The ancestor guard compares against *local* mainline, not the
	// remote-tracking ref: local `main` advancing under a concurrent
	// session is a divergence `origin/main` would not reflect. The
	// remote-tracking ref appears only in the fetch/fast-forward
	// preamble that folds in the origin axis before the check runs.
	wantGuard := "git merge-base --is-ancestor main <branch>"
	if !strings.Contains(reconcile, wantGuard) {
		t.Errorf("reconcile step must name the ancestor guard %q (local mainline, not origin/main)", wantGuard)
	}

	fetchIdx := strings.Index(reconcile, "git fetch")
	ffIdx := strings.Index(reconcile, "--ff-only origin/main")
	guardIdx := strings.Index(reconcile, wantGuard)
	if fetchIdx < 0 || ffIdx < 0 {
		t.Fatal("reconcile step must document `git fetch` and fast-forwarding local main to origin/main before the ancestor guard")
	}
	if fetchIdx >= guardIdx || ffIdx >= guardIdx {
		t.Errorf("reconcile step must run the fetch/fast-forward preamble BEFORE the ancestor guard (fetch=%d, ff=%d, guard=%d)", fetchIdx, ffIdx, guardIdx)
	}

	integrateIdx := strings.Index(reconcile, "integrate mainline into the patch branch")
	gateIdx := strings.Index(reconcile, "re-run the full local CI gate")
	if integrateIdx < 0 || gateIdx < 0 {
		t.Fatal("reconcile step must document integrating mainline into the patch branch and re-running the full local CI gate")
	}
	if integrateIdx >= gateIdx {
		t.Errorf("reconcile step must order integrate-mainline before re-run-the-gate (got indices integrate=%d, gate=%d)", integrateIdx, gateIdx)
	}
}

// TestWfPatch_MergeHardcodesNoFF asserts the merge step names the
// mechanism directly — `--no-ff --no-commit` — rather than deferring
// to an unwritten project policy. Every patch merge ever landed in
// this repo's own history is a `--no-ff` merge commit; the retired
// "fast-forward, rebase-and-merge, cherry-pick... follows the
// project's policy" framing never matched observed practice, and the
// project's CLAUDE.md never actually named a mechanism to defer to.
// Structural per CLAUDE.md *Substring assertions are not structural
// assertions*: the hardcoded command is scoped to the merge step's
// own section; the retired framing's absence is checked file-wide
// since it named a *removed* concept that must not resurface anywhere.
func TestWfPatch_MergeHardcodesNoFF(t *testing.T) {
	t.Parallel()
	body := loadWfPatchFixture(t)

	merge := extractMarkdownSection(body, 3, "12. Merge")
	if merge == "" {
		t.Fatal("could not extract the `### 12. Merge the patch branch to mainline` section")
	}
	if !strings.Contains(merge, "git merge --no-ff --no-commit <branch>") {
		t.Error("merge step must hardcode `git merge --no-ff --no-commit <branch>`, not defer the mechanism to project policy")
	}

	if strings.Contains(body, "The skill does not prescribe the mechanism") {
		t.Error(`the retired "skill does not prescribe the mechanism" framing must be removed — the merge mechanism is now hardcoded`)
	}
	if strings.Contains(strings.ToLower(body), "fast-forward, rebase-and-merge, cherry-pick") {
		t.Error("the retired mechanism-enumeration framing must be removed")
	}
}

// TestWfPatch_TrackerClosureMechanicalGuard asserts the tracker-
// closure step retains the mechanical backstop note: `aiwf` refuses a
// `--by-commit` SHA not reachable from `HEAD`, so a closure written
// before the merge lands is rejected rather than recording a commit
// mainline does not contain.
func TestWfPatch_TrackerClosureMechanicalGuard(t *testing.T) {
	t.Parallel()
	body := loadWfPatchFixture(t)

	closure := extractMarkdownSection(body, 3, "13. Tracker closure")
	if closure == "" {
		t.Fatal("could not extract the `### 13. Tracker closure` section")
	}
	if !strings.Contains(closure, "reachable from `HEAD`") {
		t.Error("tracker-closure section must note the mechanical guard (a --by-commit SHA must be reachable from `HEAD`)")
	}
}
