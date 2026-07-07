package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// m0234_worktree_add_rewire_test.go pins M-0234: every ritual call site
// and CLAUDE.md section that creates a git worktree names the atomic
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

// TestM0234_AC2_WfPatchBranchStepInvokesWorktreeAdd pins AC-2: wf-patch's
// "Create a descriptive branch" step names `aiwf worktree add`, scoped to
// the step's own subsection.
func TestM0234_AC2_WfPatchBranchStepInvokesWorktreeAdd(t *testing.T) {
	t.Parallel()
	body := loadWfPatchFixture(t)

	section := extractMarkdownSection(body, 3, "2. Create a descriptive branch")
	if section == "" {
		t.Fatal("AC-2: wf-patch must contain a `### 2. Create a descriptive branch` subsection")
	}
	assertMarkers(t, "AC-2: wf-patch branch-creation subsection", section, []marker{
		{"the aiwf worktree add verb", "aiwf worktree add", false},
		{"the default-to-a-worktree cross-reference", "Default to a worktree", false},
	})
}

// TestM0234_AC3_StartEpicWorktreeStepInvokesWorktreeAdd pins AC-3:
// aiwfx-start-epic's worktree-placement step (step 8) names `aiwf
// worktree add` for the placements that create a new worktree (in-repo
// default and sibling), resolving the epic spec's open question — this
// ritual does create a worktree directly, so it is rewired here alongside
// start-milestone and wf-patch.
func TestM0234_AC3_StartEpicWorktreeStepInvokesWorktreeAdd(t *testing.T) {
	t.Parallel()
	body := loadAiwfxStartEpicFixture(t)

	section := findWorktreePromptSection(body)
	if section == "" {
		t.Fatal("AC-3: start-epic must contain a `### …worktree…` subsection (step 8)")
	}
	assertMarkers(t, "AC-3: start-epic worktree subsection", section, []marker{
		{"the aiwf worktree add verb", "aiwf worktree add", false},
		{"the aiwf doctor materialization check", "aiwf doctor", false},
	})

	// The no-new-worktree placement (main checkout) legitimately keeps
	// plain `git checkout -b` — only the two placements that create a
	// worktree are rewired. Guard against a regression where the whole
	// section stops naming git checkout -b at all (placement 2 would
	// have nothing to do).
	if !strings.Contains(section, "git checkout -b") {
		t.Error("AC-3: start-epic worktree subsection must still name `git checkout -b` for the no-new-worktree (main checkout) placement")
	}
}

// TestM0234_AC4_ClaudeMdWorktreeSectionsCiteWorktreeAdd pins AC-4:
// CLAUDE.md's "Default to a worktree for any branch work" and "Subagent
// worktree isolation" sections cite `aiwf worktree add` instead of the
// raw `git worktree add` two-command sequence, and the subagent-dispatch
// procedure still passes the absolute worktree path into the subagent's
// prompt rather than relying on `cd` (unchanged by this milestone — the
// new verb has no more ability to change a subagent's cwd than the raw
// command did).
func TestM0234_AC4_ClaudeMdWorktreeSectionsCiteWorktreeAdd(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("reading CLAUDE.md: %v", err)
	}
	body := string(raw)

	defaultSection := extractMarkdownSection(body, 2, "Default to a worktree for any branch work in this repo")
	if defaultSection == "" {
		t.Fatal("AC-4: CLAUDE.md must contain a `## Default to a worktree for any branch work in this repo` section")
	}
	if !strings.Contains(defaultSection, "aiwf worktree add") {
		t.Error("AC-4: `## Default to a worktree for any branch work in this repo` must name `aiwf worktree add`")
	}

	subagentSection := extractMarkdownSection(body, 2, "Subagent worktree isolation")
	if subagentSection == "" {
		t.Fatal("AC-4: CLAUDE.md must contain a `## Subagent worktree isolation` section")
	}
	assertMarkers(t, "AC-4: CLAUDE.md Subagent worktree isolation section", subagentSection, []marker{
		{"the aiwf worktree add verb", "aiwf worktree add", false},
		{"the aiwf doctor materialization check", "aiwf doctor", false},
		{"the absolute-path-not-cd instruction", "absolute paths", false},
	})
	if strings.Contains(subagentSection, "git worktree add") {
		t.Error("AC-4: `## Subagent worktree isolation` must not name the raw `git worktree add` two-command sequence anymore")
	}
}
