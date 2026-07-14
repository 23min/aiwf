package policies

import "testing"

// g0413_enterworktree_chain_test.go pins the fix for G-0413: `aiwf worktree
// add` creates the git worktree on disk but never relocates the Claude Code
// harness session into it — only the harness `EnterWorktree` tool call does
// that. None of the three ritual skills that shell out to `aiwf worktree
// add` for their own direct-work sessions (aiwfx-start-milestone,
// aiwfx-start-epic, wf-patch) chained that second step, leaving the
// statusline, the worktree exit-prompt tracking, and cwd-dependent caches
// blind to worktrees entered by a plain shell `cd`.
//
// The fix is skill-content-only: each call site now captures the created
// path via `aiwf worktree add --print-path` and instructs the caller to
// invoke `EnterWorktree(path: <printed path>)` as an explicit second step —
// scoped to the direct-work case, not CLAUDE.md's "Subagent worktree
// isolation" flow, where a dispatched `Agent` receives an explicit path
// rather than the harness session itself relocating.
//
// Per CLAUDE.md *Substring assertions are not structural assertions*, every
// assertion below is scoped to the specific section the fix landed in
// (reusing the fixture loaders and section locators from
// aiwfx_start_milestone_test.go, aiwfx_start_epic_test.go, and
// wf_patch_reconcile_test.go, same package), not a flat body grep.

// TestG0413_StartMilestoneCutStepChainsEnterWorktree pins the
// aiwfx-start-milestone call site (step 5's worktree-isolation paragraph):
// --print-path is captured and EnterWorktree is chained, scoped to the
// direct-work case.
func TestG0413_StartMilestoneCutStepChainsEnterWorktree(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartMilestoneFixture(t)

	section := findStartMilestoneCutSection(body)
	if section == "" {
		t.Fatal("start-milestone must contain a `### …cut…` subsection (step 5)")
	}
	assertMarkers(t, "start-milestone cut subsection", section, []marker{
		{"the --print-path flag", "--print-path", false},
		{"the harness EnterWorktree tool call", "EnterWorktree(path:", false},
		{"the direct-work vs subagent-dispatch scoping", "Subagent worktree isolation", false},
	})
}

// TestG0413_StartEpicWorktreeStepChainsEnterWorktree pins the
// aiwfx-start-epic call site (step 8's worktree-placement section):
// --print-path is captured and EnterWorktree is chained for placements 1
// and 3 (the two that create a new worktree), scoped to the direct-work
// case.
func TestG0413_StartEpicWorktreeStepChainsEnterWorktree(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartEpicFixture(t)

	section := findWorktreePromptSection(body)
	if section == "" {
		t.Fatal("start-epic must contain a `### …worktree…` subsection (step 8)")
	}
	assertMarkers(t, "start-epic worktree subsection", section, []marker{
		{"the --print-path flag", "--print-path", false},
		{"the harness EnterWorktree tool call", "EnterWorktree(path:", false},
		{"the direct-work vs subagent-dispatch scoping", "Subagent worktree isolation", false},
	})
}

// TestG0413_WfPatchBranchStepChainsEnterWorktree pins wf-patch's own
// "Create a descriptive branch" step: --print-path is captured and
// EnterWorktree is chained, since wf-patch always runs as the calling
// session's own direct work.
func TestG0413_WfPatchBranchStepChainsEnterWorktree(t *testing.T) {
	t.Parallel()
	body := loadWfPatchFixture(t)

	section := extractMarkdownSection(body, 3, "2. Create a descriptive branch")
	if section == "" {
		t.Fatal("wf-patch must contain a `### 2. Create a descriptive branch` subsection")
	}
	assertMarkers(t, "wf-patch branch-creation subsection", section, []marker{
		{"the --print-path flag", "--print-path", false},
		{"the harness EnterWorktree tool call", "EnterWorktree(path:", false},
		{"the direct-work framing", "own direct work", false},
	})
}
