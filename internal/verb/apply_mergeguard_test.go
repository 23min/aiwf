package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// unrelatedPlan is a minimal, non-conflicting write plan — used across
// this file's tests so a refusal can only be attributed to the
// in-progress-operation guard, never to the pre-staged-conflict guard
// (checkStagedConflict), which fires against a disjoint path set.
func unrelatedPlan() *verb.Plan {
	return &verb.Plan{
		Subject:  "test write during pending git operation",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpWrite, Path: "work/epics/E-0099-unrelated/epic.md", Content: []byte("---\nid: E-0099\n---\n")},
		},
	}
}

// commitFeatureBranch checks out a new branch from the repo's current
// HEAD, commits one new file on it, and returns to the original
// branch — leaving the feature branch ready to merge/cherry-pick.
// Returns the feature branch's tip SHA.
func commitFeatureBranch(t *testing.T, r *applyTestRepo, branch string) string {
	t.Helper()
	if _, err := runGit(r.ctx, r.root, "checkout", "-b", branch); err != nil {
		t.Fatalf("checkout -b %s: %v", branch, err)
	}
	featurePath := "work/epics/E-0002-feature/epic.md"
	full := filepath.Join(r.root, featurePath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("---\nid: E-0002\n---\nfeature body\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(r.ctx, r.root, featurePath); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(r.ctx, r.root, "feature commit", "", nil); err != nil {
		t.Fatal(err)
	}
	sha := headSHA(t, r.root)
	if _, err := runGit(r.ctx, r.root, "checkout", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}
	return sha
}

// TestApply_RefusesDuringPendingMerge reproduces G-0329's precondition:
// a `git merge --no-ff --no-commit` staged in the index. Apply must
// refuse before touching anything, naming the merge, and MERGE_HEAD
// must survive the refusal untouched — the operator can still
// complete or abort the merge normally.
func TestApply_RefusesDuringPendingMerge(t *testing.T) {
	t.Parallel()
	r := newApplyTestRepo(t)
	commitFeatureBranch(t, r, "feature")

	if _, err := runGit(r.ctx, r.root, "merge", "--no-ff", "--no-commit", "feature"); err != nil {
		t.Fatalf("merge --no-ff --no-commit: %v", err)
	}
	mergeHead := filepath.Join(r.root, ".git", "MERGE_HEAD")
	if _, statErr := os.Stat(mergeHead); statErr != nil {
		t.Fatalf("precondition: MERGE_HEAD must exist after merge --no-commit: %v", statErr)
	}
	preHead := headSHA(t, r.root)

	_, err := verb.Apply(r.ctx, r.root, unrelatedPlan())
	if err == nil {
		t.Fatal("expected Apply to refuse while a merge is in progress; got nil")
	}
	if !strings.Contains(err.Error(), "merge") {
		t.Errorf("error should name the in-progress merge: %v", err)
	}

	if got := headSHA(t, r.root); got != preHead {
		t.Errorf("HEAD advanced despite the pending merge; guard must fire before any commit")
	}
	if _, statErr := os.Stat(mergeHead); statErr != nil {
		t.Errorf("MERGE_HEAD was disturbed by the refused Apply call: %v", statErr)
	}
}

// TestApply_RefusesDuringPendingCherryPick mirrors the merge case for
// a real cherry-pick conflict, which pauses git with CHERRY_PICK_HEAD
// set until the operator resolves and runs `--continue` or `--abort`.
// Deliberately not `--no-commit`: git (as of 2.5x) does not write
// CHERRY_PICK_HEAD for a `--no-commit` cherry-pick, conflict or not —
// that flag bypasses sequencer state entirely, so it doesn't reproduce
// the "operation left mid-flight" precondition this guard exists for.
func TestApply_RefusesDuringPendingCherryPick(t *testing.T) {
	t.Parallel()
	r := newApplyTestRepo(t)
	full := filepath.Join(r.root, r.trackedPath)

	if _, err := runGit(r.ctx, r.root, "checkout", "-b", "feature"); err != nil {
		t.Fatalf("checkout -b feature: %v", err)
	}
	if err := os.WriteFile(full, []byte("feature change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(r.ctx, r.root, r.trackedPath); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(r.ctx, r.root, "feature change", "", nil); err != nil {
		t.Fatal(err)
	}
	sha := headSHA(t, r.root)

	if _, err := runGit(r.ctx, r.root, "checkout", "main"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}
	if err := os.WriteFile(full, []byte("main change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(r.ctx, r.root, r.trackedPath); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(r.ctx, r.root, "main change", "", nil); err != nil {
		t.Fatal(err)
	}

	// Conflicting cherry-pick, no --no-commit: git pauses with
	// CHERRY_PICK_HEAD set. Expected to fail — that failure IS the
	// precondition being reproduced, not a test-setup error.
	if _, err := runGit(r.ctx, r.root, "cherry-pick", sha); err == nil {
		t.Fatal("precondition: expected the cherry-pick to conflict and pause")
	}
	cherryHead := filepath.Join(r.root, ".git", "CHERRY_PICK_HEAD")
	if _, statErr := os.Stat(cherryHead); statErr != nil {
		t.Fatalf("precondition: CHERRY_PICK_HEAD must exist after a conflicting cherry-pick: %v", statErr)
	}

	_, err := verb.Apply(r.ctx, r.root, unrelatedPlan())
	if err == nil {
		t.Fatal("expected Apply to refuse while a cherry-pick is in progress; got nil")
	}
	if !strings.Contains(err.Error(), "cherry-pick") {
		t.Errorf("error should name the in-progress cherry-pick: %v", err)
	}
	if _, statErr := os.Stat(cherryHead); statErr != nil {
		t.Errorf("CHERRY_PICK_HEAD was disturbed by the refused Apply call: %v", statErr)
	}
}

// TestApply_RefusesDuringPendingRevert mirrors the merge case for
// `git revert --no-commit`, which leaves REVERT_HEAD behind.
func TestApply_RefusesDuringPendingRevert(t *testing.T) {
	t.Parallel()
	r := newApplyTestRepo(t)

	revertPath := "revert-me.md"
	if err := os.WriteFile(filepath.Join(r.root, revertPath), []byte("content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(r.ctx, r.root, revertPath); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(r.ctx, r.root, "add revert-me", "", nil); err != nil {
		t.Fatal(err)
	}
	targetSHA := headSHA(t, r.root)

	if _, err := runGit(r.ctx, r.root, "revert", "--no-commit", targetSHA); err != nil {
		t.Fatalf("revert --no-commit: %v", err)
	}
	revertHead := filepath.Join(r.root, ".git", "REVERT_HEAD")
	if _, statErr := os.Stat(revertHead); statErr != nil {
		t.Fatalf("precondition: REVERT_HEAD must exist after revert --no-commit: %v", statErr)
	}

	_, err := verb.Apply(r.ctx, r.root, unrelatedPlan())
	if err == nil {
		t.Fatal("expected Apply to refuse while a revert is in progress; got nil")
	}
	if !strings.Contains(err.Error(), "revert") {
		t.Errorf("error should name the in-progress revert: %v", err)
	}
	if _, statErr := os.Stat(revertHead); statErr != nil {
		t.Errorf("REVERT_HEAD was disturbed by the refused Apply call: %v", statErr)
	}
}

// TestApply_RefusesDuringPendingRebase simulates a paused rebase by
// creating the rebase-merge marker directory git itself creates —
// mirroring the stale-lock synthetic fixture in apply_lock_test.go,
// since orchestrating a real conflicted rebase adds setup weight
// without exercising any different code path (the guard only ever
// stats the directory).
func TestApply_RefusesDuringPendingRebase(t *testing.T) {
	t.Parallel()
	r := newApplyTestRepo(t)

	rebaseDir := filepath.Join(r.root, ".git", "rebase-merge")
	if err := os.Mkdir(rebaseDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := verb.Apply(r.ctx, r.root, unrelatedPlan())
	if err == nil {
		t.Fatal("expected Apply to refuse while a rebase is in progress; got nil")
	}
	if !strings.Contains(err.Error(), "rebase") {
		t.Errorf("error should name the in-progress rebase: %v", err)
	}
	if _, statErr := os.Stat(rebaseDir); statErr != nil {
		t.Errorf("rebase-merge marker was disturbed by the refused Apply call: %v", statErr)
	}
}

// TestApply_RefusesDuringPendingRebase_ApplyBackend covers the
// non-interactive rebase backend's marker directory (`rebase-apply`,
// as opposed to the interactive backend's `rebase-merge` covered
// above) — the guard checks both names, and each is its own loop
// iteration / branch.
func TestApply_RefusesDuringPendingRebase_ApplyBackend(t *testing.T) {
	t.Parallel()
	r := newApplyTestRepo(t)

	rebaseDir := filepath.Join(r.root, ".git", "rebase-apply")
	if err := os.Mkdir(rebaseDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := verb.Apply(r.ctx, r.root, unrelatedPlan())
	if err == nil {
		t.Fatal("expected Apply to refuse while a rebase is in progress; got nil")
	}
	if !strings.Contains(err.Error(), "rebase") {
		t.Errorf("error should name the in-progress rebase: %v", err)
	}
	if _, statErr := os.Stat(rebaseDir); statErr != nil {
		t.Errorf("rebase-apply marker was disturbed by the refused Apply call: %v", statErr)
	}
}

// TestApply_PendingMergeInMainCheckoutDoesNotBlockLinkedWorktree pins
// the worktree-aware half of the fix: MERGE_HEAD lives in the
// per-worktree gitdir, not the shared common dir, so a merge pending
// in the main checkout must not block an unrelated apply-routed verb
// running against a linked worktree with no operation of its own.
func TestApply_PendingMergeInMainCheckoutDoesNotBlockLinkedWorktree(t *testing.T) {
	t.Parallel()
	r := newApplyTestRepo(t)

	wtPath := filepath.Join(t.TempDir(), "wt")
	if err := gitops.WorktreeAddNewBranch(r.ctx, r.root, wtPath, "feature-wt", "main"); err != nil {
		t.Fatalf("WorktreeAddNewBranch: %v", err)
	}

	commitFeatureBranch(t, r, "feature-src")
	if _, err := runGit(r.ctx, r.root, "merge", "--no-ff", "--no-commit", "feature-src"); err != nil {
		t.Fatalf("merge --no-ff --no-commit: %v", err)
	}

	// The main checkout itself is still blocked.
	if _, err := verb.Apply(r.ctx, r.root, unrelatedPlan()); err == nil {
		t.Fatal("expected Apply to refuse in the main checkout with a pending merge; got nil")
	}

	// The linked worktree has no operation of its own and must proceed.
	wtPlan := &verb.Plan{
		Subject:  "test write in unrelated linked worktree",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpWrite, Path: "work/epics/E-0098-worktree/epic.md", Content: []byte("---\nid: E-0098\n---\n")},
		},
	}
	if _, err := verb.Apply(r.ctx, wtPath, wtPlan); err != nil {
		t.Errorf("Apply in unrelated linked worktree should succeed despite the main checkout's pending merge: %v", err)
	}
}
