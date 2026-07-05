package gitops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReconcilePaths_StagesOnlyWrittenPaths pins M-0186/AC-2: after a
// CommitTree commit lands, ReconcilePaths stages exactly the paths it
// wrote into the live index — leaving every other staged path
// byte-for-byte untouched. Pre-stages path A with distinct content,
// commits a write to path B via CommitTree (which never touches the live
// index — AC-1), then reconciles only B. A's staged content must be
// unchanged; B must be clean in the live index (its staged content
// matches what CommitTree just committed).
func TestReconcilePaths_StagesOnlyWrittenPaths(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t, ctx) // base.md committed at HEAD

	// Pre-stage path A with distinct content — simulates the caller's own
	// unrelated pending work that must survive reconciliation untouched.
	err := os.WriteFile(filepath.Join(root, "a.md"), []byte("A content\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	err = Add(ctx, root, "a.md")
	if err != nil {
		t.Fatalf("add a.md: %v", err)
	}

	bWrite := PathWrite{Path: "b.md", Content: []byte("B content\n")}
	sha, err := CommitTree(ctx, root, []PathWrite{bWrite}, "write b", "", nil)
	if err != nil {
		t.Fatalf("CommitTree: %v", err)
	}

	err = ReconcilePaths(ctx, root, []PathWrite{bWrite})
	if err != nil {
		t.Fatalf("ReconcilePaths: %v", err)
	}

	// A's staged content is unchanged: the index blob at ":a.md" still
	// matches what was staged before ReconcilePaths ran.
	aStaged, err := output(ctx, root, "show", ":a.md")
	if err != nil {
		t.Fatalf("show :a.md: %v", err)
	}
	if aStaged != "A content\n" {
		t.Errorf("staged a.md = %q, want %q (ReconcilePaths must not touch unrelated staged paths)", aStaged, "A content\n")
	}

	// B is clean in the live index: no diff between the index and HEAD
	// for b.md.
	bDiff, err := output(ctx, root, "diff", "--cached", "--", "b.md")
	if err != nil {
		t.Fatalf("diff --cached -- b.md: %v", err)
	}
	if bDiff != "" {
		t.Errorf("b.md not clean in the live index after ReconcilePaths: %q", bDiff)
	}

	// b.md is staged at the exact blob CommitTree wrote into HEAD.
	bIndexShow, err := output(ctx, root, "show", ":b.md")
	if err != nil {
		t.Fatalf("show :b.md: %v", err)
	}
	if bIndexShow != "B content\n" {
		t.Errorf("staged b.md = %q, want %q", bIndexShow, "B content\n")
	}

	// Sanity: the commit itself landed (guards against a false pass if
	// CommitTree silently no-oped).
	if sha == "" {
		t.Fatal("CommitTree returned empty SHA")
	}
}

// TestReconcilePaths_HashObjectFails_ObjectsDirReadOnly exercises
// ReconcilePaths' own error branch: the object database can't be
// written to, so hashing the write's content fails before update-index
// ever runs.
func TestReconcilePaths_HashObjectFails_ObjectsDirReadOnly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t, ctx)

	gitDir, err := GitDir(ctx, root)
	if err != nil {
		t.Fatalf("GitDir: %v", err)
	}
	objectsDir := filepath.Join(gitDir, "objects")
	err = os.Chmod(objectsDir, 0o500)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(objectsDir, 0o755) })

	err = ReconcilePaths(ctx, root, []PathWrite{{Path: "a.md", Content: []byte("a\n")}})
	if err == nil {
		t.Fatal("want error with a read-only objects dir, got nil")
	}
	if !strings.Contains(err.Error(), "hashing blob") {
		t.Errorf("error %q should mention hashing blob", err.Error())
	}
}

// TestReconcilePaths_UpdateIndexFails_StaleLock reproduces a real,
// deterministically-triggerable failure at the update-index step
// specifically (not hash-object): a stale `.git/index.lock` left behind
// by a crashed or still-running git process. hash-object succeeds (it
// only writes to the object database, never touches the index lock);
// update-index fails because it can't acquire the lock.
func TestReconcilePaths_UpdateIndexFails_StaleLock(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t, ctx)

	gitDir, err := GitDir(ctx, root)
	if err != nil {
		t.Fatalf("GitDir: %v", err)
	}
	lockPath := filepath.Join(gitDir, "index.lock")
	err = os.WriteFile(lockPath, nil, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(lockPath) })

	err = ReconcilePaths(ctx, root, []PathWrite{{Path: "a.md", Content: []byte("a\n")}})
	if err == nil {
		t.Fatal("want error with a stale index.lock, got nil")
	}
	if !strings.Contains(err.Error(), "update-index") {
		t.Errorf("error %q should mention update-index", err.Error())
	}
}

// TestReconcilePaths_OverwritesExistingTrackedFile pins the primary
// real-world case per AC-3: most aiwf verbs (promote, edit-body, cancel)
// rewrite an EXISTING entity file, not add a new one. The path's live
// index entry pre-reconciliation still holds the OLD content (CommitTree
// never touches the live index — AC-1); update-index --add --cacheinfo
// must replace that stale entry rather than duplicate it.
func TestReconcilePaths_OverwritesExistingTrackedFile(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t, ctx) // base.md = "base\n", tracked and clean at HEAD

	write := PathWrite{Path: "base.md", Content: []byte("overwritten\n")}
	_, err := CommitTree(ctx, root, []PathWrite{write}, "overwrite base.md", "", nil)
	if err != nil {
		t.Fatalf("CommitTree: %v", err)
	}

	err = ReconcilePaths(ctx, root, []PathWrite{write})
	if err != nil {
		t.Fatalf("ReconcilePaths: %v", err)
	}

	diff, err := output(ctx, root, "diff", "--cached", "--", "base.md")
	if err != nil {
		t.Fatalf("diff --cached -- base.md: %v", err)
	}
	if diff != "" {
		t.Errorf("base.md not clean in the live index after ReconcilePaths: %q", diff)
	}

	staged, err := output(ctx, root, "ls-files", "--stage", "--", "base.md")
	if err != nil {
		t.Fatalf("ls-files --stage -- base.md: %v", err)
	}
	if got := strings.Count(strings.TrimSpace(staged), "\n") + 1; got != 1 {
		t.Errorf("base.md has %d index entries, want exactly 1: %q", got, staged)
	}
}

// TestReconcilePaths_EmptyWrites is a no-op: no paths to reconcile means
// no git subprocess runs and no error.
func TestReconcilePaths_EmptyWrites(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t, ctx)

	if err := ReconcilePaths(ctx, root, nil); err != nil {
		t.Errorf("ReconcilePaths(nil) = %v, want nil", err)
	}
}
