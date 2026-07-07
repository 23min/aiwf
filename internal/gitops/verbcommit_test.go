package gitops

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCommitVerbChange_HappyPath pins the full sequence M-0186/AC-5
// factors into one call: the commit lands (git ls-tree shows the write),
// and the written path is reconciled into the live index (StagedPaths —
// which reports paths differing from HEAD — comes back empty for it).
func TestCommitVerbChange_HappyPath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t, ctx)

	sha, err := CommitVerbChange(ctx, root, nil, []PathWrite{{Path: "a.md", Content: []byte("hi\n")}}, "add a.md", "", nil)
	if err != nil {
		t.Fatalf("CommitVerbChange: %v", err)
	}
	if sha == "" {
		t.Fatal("want a non-empty commit SHA")
	}

	out, lsErr := output(ctx, root, "ls-tree", "-r", "--name-only", sha)
	if lsErr != nil {
		t.Fatalf("ls-tree: %v", lsErr)
	}
	if !strings.Contains(out, "a.md") {
		t.Errorf("committed tree %q does not contain a.md", out)
	}

	staged, stagedErr := StagedPaths(ctx, root)
	if stagedErr != nil {
		t.Fatalf("StagedPaths: %v", stagedErr)
	}
	if len(staged) != 0 {
		t.Errorf("want a.md reconciled into the live index (no staged diff against HEAD), got staged=%v", staged)
	}
}

// TestCommitVerbChange_CommitFailurePropagatesPlainError pins the first
// failure shape: when CommitTree itself fails, CommitVerbChange returns
// before anything lands — sha is empty, and the error is NOT wrapped as
// a *ReconcileError (nothing was reconciled; there was nothing to
// reconcile against).
func TestCommitVerbChange_CommitFailurePropagatesPlainError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t, ctx)
	if err := run(ctx, root, "config", "commit.gpgsign", "banana"); err != nil {
		t.Fatalf("config commit.gpgsign: %v", err)
	}

	sha, err := CommitVerbChange(ctx, root, nil, []PathWrite{{Path: "a.md", Content: []byte("hi\n")}}, "should not land", "", nil)
	if err == nil {
		t.Fatal("expected an error from a malformed commit.gpgsign value, got nil")
	}
	if sha != "" {
		t.Errorf("want an empty SHA on commit failure, got %q", sha)
	}
	var reconcileErr *ReconcileError
	if errors.As(err, &reconcileErr) {
		t.Errorf("expected a plain commit error, got *ReconcileError: %v", reconcileErr)
	}
}

// TestCommitVerbChange_ReconcileFailureWrapsSHA pins the second failure
// shape: the commit lands, but reconciling the written path into the
// live index fails (a stale `.git/index.lock`, exactly like
// TestReconcilePaths_UpdateIndexFails_StaleLock). CommitTree itself
// never touches the live index, so the lock does not block it —
// CommitVerbChange must still return the landed SHA, wrapped in a
// *ReconcileError so a caller can tell "committed but not reconciled"
// apart from "nothing happened."
func TestCommitVerbChange_ReconcileFailureWrapsSHA(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t, ctx)

	gitDir, err := GitDir(ctx, root)
	if err != nil {
		t.Fatalf("GitDir: %v", err)
	}
	lockPath := filepath.Join(gitDir, "index.lock")
	if writeErr := os.WriteFile(lockPath, nil, 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}
	t.Cleanup(func() { _ = os.Remove(lockPath) })

	sha, commitErr := CommitVerbChange(ctx, root, nil, []PathWrite{{Path: "a.md", Content: []byte("hi\n")}}, "lands despite lock", "", nil)
	if sha == "" {
		t.Fatal("want a non-empty commit SHA even though reconciliation fails")
	}

	var reconcileErr *ReconcileError
	if !errors.As(commitErr, &reconcileErr) {
		t.Fatalf("want a *ReconcileError, got %T: %v", commitErr, commitErr)
	}
	if reconcileErr.SHA != sha {
		t.Errorf("ReconcileError.SHA = %q, want %q", reconcileErr.SHA, sha)
	}
	if !strings.Contains(reconcileErr.Err.Error(), "update-index") {
		t.Errorf("ReconcileError.Err = %q, want it to mention update-index", reconcileErr.Err)
	}
	if !strings.Contains(reconcileErr.Error(), sha) {
		t.Errorf("ReconcileError.Error() = %q, want it to name the landed SHA %q", reconcileErr.Error(), sha)
	}
	if !errors.Is(reconcileErr, reconcileErr.Err) {
		t.Errorf("errors.Is(reconcileErr, reconcileErr.Err) = false, want true (Unwrap must expose the wrapped error)")
	}

	// The commit landed (git history), verified independent of the
	// wrapper: rev-parse the SHA and confirm it's a real, reachable
	// commit object with the expected tree content.
	out, lsErr := output(ctx, root, "ls-tree", "-r", "--name-only", sha)
	if lsErr != nil {
		t.Fatalf("ls-tree on the landed commit: %v", lsErr)
	}
	if !strings.Contains(out, "a.md") {
		t.Errorf("landed commit tree %q does not contain a.md", out)
	}
}
