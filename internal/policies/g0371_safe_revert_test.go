package policies

import (
	"strings"
	"testing"
)

// g0371_safe_revert_test.go pins the fix for G-0371: a `wf-vacuity`
// mutation-probe revert run via `git stash`/`checkout`/`restore` against a
// working tree shared with a pending commit can silently desync the index
// from what actually lands — a reviewer's stash/pop in a live `wf-patch`
// review once shipped a commit missing its own fix, caught only by
// after-the-fact inspection. The fix has three parts, each pinned below:
//
//  1. wf-vacuity's mutation probe (the root cause: it named no safe revert
//     mechanism) now specifies capture-and-restore or an isolated worktree,
//     and forbids git-index-touching reverts explicitly.
//  2. wf-patch adds an orchestrator-side backstop: a diff fingerprint
//     captured before dispatching the reviewer, re-verified at the commit
//     gate and again immediately before `git commit`.
//  3. Both wf-patch (dispatch site) and wf-tdd-cycle (the other required
//     wf-vacuity call site) carry the reviewer-dispatch contract: no
//     mutation of the shared tree/index, ever.
//
// Per CLAUDE.md *Substring assertions are not structural assertions*, every
// assertion below is scoped to the specific section the fix landed in, not
// a flat body grep.

const wfVacuityFixturePath = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-vacuity/SKILL.md"

// TestWfVacuity_MutationProbeNamesSafeRevert pins fix (1): the mutation
// probe step names an explicit, git-index-free revert mechanism and
// forbids the verbs that caused the incident.
func TestWfVacuity_MutationProbeNamesSafeRevert(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfVacuityFixturePath)

	probe := extractMarkdownSection(body, 3, "1. Mutation probe")
	if probe == "" {
		t.Fatal("could not extract the `### 1. Mutation probe` section")
	}

	if !strings.Contains(probe, "git show HEAD:<path>") {
		t.Error("mutation probe must name `git show HEAD:<path>` as a safe way to capture pre-mutation content")
	}

	if !strings.Contains(probe, "byte-for-byte") {
		t.Error("mutation probe must require the captured content is written back byte-for-byte")
	}

	for _, forbidden := range []string{"git stash", "git checkout <path>", "git restore <path>"} {
		if !strings.Contains(probe, forbidden) {
			t.Errorf("mutation probe must explicitly name %q among the forbidden revert verbs", forbidden)
		}
	}
	if !strings.Contains(probe, "isolated worktree") {
		t.Error("mutation probe must offer an isolated worktree as the alternative for a dispatched reviewer sharing the orchestrator's checkout")
	}
}

// TestWfVacuity_ConstraintsForbidGitIndexRevert pins the mirrored
// constraint: the 🛑 revert rule itself must name the same prohibition, not
// just the workflow prose (a reader skimming only the Constraints section
// must still see it).
func TestWfVacuity_ConstraintsForbidGitIndexRevert(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfVacuityFixturePath)

	constraints := extractMarkdownSection(body, 2, "Constraints")
	if constraints == "" {
		t.Fatal("could not extract the `## Constraints` section")
	}

	revertLine := lineContaining(constraints, "Revert every mutation")
	if revertLine == "" {
		t.Fatal("Constraints must carry a 'Revert every mutation' bullet")
	}
	if !strings.Contains(revertLine, "git-index-touching verb") {
		t.Errorf("the revert constraint must name the git-index-touching-verb prohibition, got: %q", revertLine)
	}
	for _, forbidden := range []string{"stash", "checkout", "restore"} {
		if !strings.Contains(revertLine, forbidden) {
			t.Errorf("the revert constraint must list %q among the forbidden verbs, got: %q", forbidden, revertLine)
		}
	}
}

// TestWfVacuity_AntiPatternNamesUnsafeRevert pins the anti-pattern entry
// warning against the exact failure mode from the incident.
func TestWfVacuity_AntiPatternNamesUnsafeRevert(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfVacuityFixturePath)

	antiPatterns := extractMarkdownSection(body, 2, "Anti-patterns")
	if antiPatterns == "" {
		t.Fatal("could not extract the `## Anti-patterns` section")
	}
	if !strings.Contains(antiPatterns, "Reverting via `git stash`/`checkout`/`restore`") {
		t.Error("Anti-patterns must warn against reverting via git stash/checkout/restore")
	}
}

// TestWfPatch_ReviewerDispatchContractAndFingerprint pins fix (2) and the
// wf-patch half of fix (3): step 6 (Independent review) states the
// reviewer-dispatch contract and requires capturing a pre-dispatch diff
// fingerprint, in that order — the contract sets the rule, the fingerprint
// is the backstop for when it's violated anyway.
func TestWfPatch_ReviewerDispatchContractAndFingerprint(t *testing.T) {
	t.Parallel()
	body := loadWfPatchFixture(t)

	step6 := extractMarkdownSection(body, 3, "6. Independent review")
	if step6 == "" {
		t.Fatal("could not extract the `### 6. Independent review of the diff` section")
	}

	contractIdx := strings.Index(step6, "Reviewer-dispatch contract")
	if contractIdx < 0 {
		t.Fatal("step 6 must carry a **Reviewer-dispatch contract** paragraph")
	}
	if !strings.Contains(step6, "must not mutate the shared working tree or index") {
		t.Error("the reviewer-dispatch contract must forbid mutating the shared working tree or index")
	}
	if !strings.Contains(step6, "git show HEAD:<path>") {
		t.Error("the reviewer-dispatch contract must offer `git show HEAD:<path>` as the read-only alternative")
	}

	fingerprintIdx := strings.Index(step6, "Before dispatching")
	if fingerprintIdx < 0 {
		t.Fatal("step 6 must instruct capturing a fingerprint before dispatching the reviewer")
	}
	if !strings.Contains(step6, "git diff --cached") {
		t.Error("the fingerprint instruction must name `git diff --cached` as what gets captured")
	}
	if contractIdx >= fingerprintIdx {
		t.Errorf("the reviewer-dispatch contract (idx %d) must appear before the fingerprint-capture instruction (idx %d) — the rule comes first, the backstop second", contractIdx, fingerprintIdx)
	}
}

// TestWfPatch_CommitGateReverifiesFingerprintNotJustNonEmpty pins the rest
// of fix (2): the commit gate re-checks the staged diff against the
// pre-dispatch fingerprint, and explicitly rejects a bare non-empty check
// (the incident's corrupted diff was non-empty — it just lacked the fix).
func TestWfPatch_CommitGateReverifiesFingerprintNotJustNonEmpty(t *testing.T) {
	t.Parallel()
	body := loadWfPatchFixture(t)

	commitGate := extractMarkdownSection(body, 3, "8. 🛑 Commit gate")
	if commitGate == "" {
		t.Fatal("could not extract the `### 8. 🛑 Commit gate` section")
	}
	if !strings.Contains(commitGate, "git diff --cached") {
		t.Error("commit gate must re-run `git diff --cached`")
	}
	if !strings.Contains(commitGate, "byte-identical to the fingerprint") {
		t.Error("commit gate must compare against the pre-dispatch fingerprint, not merely check non-emptiness")
	}
	if !strings.Contains(commitGate, "bare non-empty check is not sufficient") {
		t.Error("commit gate must explicitly reject a bare non-empty check as insufficient")
	}

	afterApproval := extractMarkdownSection(body, 3, "9. After commit approval")
	if afterApproval == "" {
		t.Fatal("could not extract the `### 9. After commit approval` section")
	}
	if !strings.Contains(afterApproval, "immediately before running `git commit`") {
		t.Error("the after-approval step must re-verify the fingerprint immediately before running `git commit`, not rely on the gate-time check alone")
	}
}

// TestWfTddCycle_VacuityCheckNoSharedTreeMutation pins the wf-tdd-cycle
// half of fix (3): the *other* required wf-vacuity call site (G-0371's
// investigation found wf-tdd-cycle is a more frequent path than wf-patch)
// carries the same no-shared-tree-mutation rule, not just wf-patch.
func TestWfTddCycle_VacuityCheckNoSharedTreeMutation(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfTddCycleFixturePath)

	vacuity := extractMarkdownSection(body, 2, "Vacuity check")
	if vacuity == "" {
		t.Fatal("could not extract the `## Vacuity check` section")
	}
	if !strings.Contains(vacuity, "No shared-tree mutation during the probe") {
		t.Error("wf-tdd-cycle's vacuity check must carry the no-shared-tree-mutation rule")
	}
	for _, forbidden := range []string{"git stash", "git checkout", "git restore"} {
		if !strings.Contains(vacuity, forbidden) {
			t.Errorf("wf-tdd-cycle's vacuity check must name %q among the forbidden verbs", forbidden)
		}
	}
	if !strings.Contains(vacuity, "isolate that reviewer in its own worktree") {
		t.Error("wf-tdd-cycle's vacuity check must offer the isolated-worktree alternative for a dispatched reviewer")
	}
}
