package policies

import (
	"testing"
)

// m0234_worktree_add_rewire_test.go pins M-0234: every ritual call site
// that creates a git worktree names the atomic
// `aiwf worktree add` verb (M-0233) instead of a raw `git worktree add`
// (or, for the call sites that never showed a concrete worktree-creation
// command at all, instead of leaving the step silent on it) — so a
// freshly-cut worktree actually gets its rituals materialized in
// practice, closing the gap E-0059 exists to fix.
//
// Fixture loaders (loadAiwfxStartMilestoneFixture, loadAiwfxStartEpicFixture,
// loadWfPatchFixture) and section locators (findStartMilestoneCutSection,
// findWorktreePromptSection) live in the sibling M-0190 test files in this
// package; reused here rather than duplicated.

// TestM0234_AC1_StartMilestoneCutStepInvokesWorktreeAdd pins AC-1:
// aiwfx-start-milestone's cut-branch step (step 5) names `aiwf worktree
// add` for the "isolate this milestone in its own worktree" case, scoped
// to the step's own subsection.
func TestM0234_AC1_StartMilestoneCutStepInvokesWorktreeAdd(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartMilestoneFixture(t)

	section := findStartMilestoneCutSection(body)
	if section == "" {
		t.Fatal("AC-1: start-milestone must contain a `### …cut…` subsection (step 5)")
	}
	assertMarkers(t, "AC-1: start-milestone cut subsection", section, []marker{
		{"the aiwf worktree add verb", "aiwf worktree add", false},
		{"the aiwf-worktree skill cross-reference", "aiwf-worktree", false},
	})
}
