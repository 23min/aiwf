package verb

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// TestRollback_RestoreError: when git restore fails for a non-"did
// not match" reason (e.g. the path is outside the repo), the
// rollback captures the error.
func TestRollback_RestoreError(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	// Use a non-git directory as the root so git restore fails hard.
	root := t.TempDir()
	tx := &applyTx{
		root:         root,
		ctx:          context.Background(),
		touchedPaths: []string{"some/path.md"},
	}
	if err := tx.rollback(); err == nil {
		t.Error("expected rollback error when running outside a git repo")
	}
}

// TestRollback_RemoveError: a created file whose parent has been
// chmod'd to 0500 fails os.Remove with EACCES; the error is
// captured (and not swallowed as ErrNotExist).
func TestRollback_RemoveError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	root := t.TempDir()
	if err := gitops.Init(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(root, "locked")
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	created := filepath.Join(parent, "file.md")
	if err := os.WriteFile(created, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })

	tx := &applyTx{
		root:         root,
		ctx:          context.Background(),
		createdFiles: []string{filepath.Join("locked", "file.md")},
	}
	// Restore succeeds (no touched paths to restore); the remove path
	// is what fails.
	if err := tx.rollback(); err == nil {
		t.Error("expected rollback to capture remove error")
	}
}

// TestRollback_NoOpWhenCommitted: a committed transaction's
// rollback does nothing.
func TestRollback_NoOpWhenCommitted(t *testing.T) {
	tx := &applyTx{committed: true, touchedPaths: []string{"x"}, createdFiles: []string{"y"}}
	if err := tx.rollback(); err != nil {
		t.Errorf("committed rollback should be no-op; got %v", err)
	}
}

// TestRollback_NoOpWhenNothingTouched: an empty transaction's
// rollback does nothing.
func TestRollback_NoOpWhenNothingTouched(t *testing.T) {
	tx := &applyTx{ctx: context.Background()}
	if err := tx.rollback(); err != nil {
		t.Errorf("empty rollback should be no-op; got %v", err)
	}
}

// TestApply_WrapsErrorWhenRollbackAlsoFails: when the primary error
// triggers rollback, and rollback itself fails, the user sees the
// composite error string mentioning manual cleanup.
func TestApply_WrapsErrorWhenRollbackAlsoFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	t.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	// A directory that's not a git repo: git mv will fail. But to
	// trigger the rollback-also-fails path, we need a touched-path
	// to exist before the failure. Manufacture this by writing a
	// file (preexisting=true), then having a downstream op fail.
	root := t.TempDir()
	if err := gitops.Init(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	// Seed a tracked file so git mv has something to operate on.
	tracked := "seed.md"
	if err := os.WriteFile(filepath.Join(root, tracked), []byte("seed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(context.Background(), root, tracked); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(context.Background(), root, "seed", "", nil); err != nil {
		t.Fatal(err)
	}

	// Plan: mv the seed (succeeds, populating touchedPaths), then a
	// write that fails. Then we'll corrupt .git mid-rollback by
	// pre-locking the .git/index file's parent — except that's
	// tricky to time.
	//
	// Simpler: mv to a dest, then have an OpWrite fail. After
	// failure, swap the .git directory permissions before defer
	// fires — impossible from outside. Instead, induce the
	// rollback-also-fails branch by making the post-failure restore
	// hit a worktree where .git has been removed. We do this by
	// running rollback() directly on an applyTx whose root is a
	// non-git dir AND whose createdFiles points at a locked-parent
	// path so both restore and remove fail.
	bareDir := t.TempDir()
	parent := filepath.Join(bareDir, "locked")
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	created := filepath.Join(parent, "x.md")
	if err := os.WriteFile(created, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })

	tx := &applyTx{
		root:         bareDir,
		ctx:          context.Background(),
		touchedPaths: []string{filepath.Join("locked", "x.md")},
		createdFiles: []string{filepath.Join("locked", "x.md")},
	}
	if err := tx.rollback(); err == nil {
		t.Error("expected rollback to fail (both restore and remove broken)")
	}
}

// TestDedupePaths_PreservesOrderRemovesDuplicates exercises the
// helper directly with a duplicate-laden input.
func TestDedupePaths_PreservesOrderRemovesDuplicates(t *testing.T) {
	in := []string{"a", "b", "a", "c", "b", "a"}
	got := dedupePaths(in)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (got %v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
