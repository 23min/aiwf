package verb

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// TestRollback_NoOpOutsideGitRepo: rollback is pure filesystem (M-0186)
// — a touched path with no captured pre-Apply state and nothing on
// disk at that path is simply absent both before and after, even
// outside a git repo entirely. Mirrors the days-gone git-restore
// failure this scenario used to trigger: rollback no longer calls git
// at all, so a non-repo root is no longer a failure mode.
func TestRollback_NoOpOutsideGitRepo(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tx := &applyTx{
		root:    root,
		ctx:     context.Background(),
		journal: []undoStep{writeUndo{path: "some/path.md", existed: false}},
	}
	if err := tx.rollback(); err != nil {
		t.Errorf("rollback outside a git repo = %v, want nil (rollback is pure filesystem)", err)
	}
}

// TestRollback_RemoveError: a created file whose parent has been
// chmod'd to 0500 fails os.Remove with EACCES; the error is
// captured (and not swallowed as ErrNotExist).
func TestRollback_RemoveError(t *testing.T) {
	t.Parallel()
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

	rel := filepath.Join("locked", "file.md")
	tx := &applyTx{
		root:    root,
		ctx:     context.Background(),
		journal: []undoStep{writeUndo{path: rel, existed: false}},
	}
	if err := tx.rollback(); err == nil {
		t.Error("expected rollback to capture remove error")
	}
}

// TestRollback_NoOpWhenCommitted: a committed transaction's
// rollback does nothing.
func TestRollback_NoOpWhenCommitted(t *testing.T) {
	t.Parallel()
	tx := &applyTx{committed: true, journal: []undoStep{writeUndo{path: "x", existed: false}}}
	if err := tx.rollback(); err != nil {
		t.Errorf("committed rollback should be no-op; got %v", err)
	}
}

// TestRollback_NoOpWhenNothingTouched: an empty transaction's
// rollback does nothing.
func TestRollback_NoOpWhenNothingTouched(t *testing.T) {
	t.Parallel()
	tx := &applyTx{ctx: context.Background()}
	if err := tx.rollback(); err != nil {
		t.Errorf("empty rollback should be no-op; got %v", err)
	}
}

// TestApply_WrapsErrorWhenRollbackAlsoFails: when the primary error
// triggers rollback, and rollback itself fails, the user sees the
// composite error string mentioning manual cleanup.
func TestApply_WrapsErrorWhenRollbackAlsoFails(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	// GIT_{AUTHOR,COMMITTER}_{NAME,EMAIL} are seeded once in TestMain
	// (setup_test.go) — using t.Setenv here would panic under t.Parallel.

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
	// non-git dir AND whose preApply records an absent entry at a
	// locked-parent path, so both restore and remove fail.
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

	rel := filepath.Join("locked", "x.md")
	tx := &applyTx{
		root:    bareDir,
		ctx:     context.Background(),
		journal: []undoStep{writeUndo{path: rel, existed: false}}, // absent pre-Apply → rollback removes (and fails)
	}
	if err := tx.rollback(); err == nil {
		t.Error("expected rollback to fail (both restore and remove broken)")
	}
}

// TestCheckStagedConflict_DirectoryMoveNestedPaths pins G-0377: the
// guard must catch a staged edit nested inside a directory an OpMove
// is about to relocate — not just an exact match on the directory's
// own Path/NewPath — since gatherCommitOps walks the destination
// recursively and would otherwise silently absorb that staged content
// into the verb's commit with no refusal ever firing.
func TestCheckStagedConflict_DirectoryMoveNestedPaths(t *testing.T) {
	t.Parallel()
	moveOps := []FileOp{{Type: OpMove, Path: "work/epics/E-0001-src", NewPath: "work/epics/E-0002-dst"}}

	tests := []struct {
		name         string
		staged       []string
		ops          []FileOp
		wantConflict bool
	}{
		{
			name:         "exact match on OpMove source",
			staged:       []string{"work/epics/E-0001-src"},
			ops:          moveOps,
			wantConflict: true,
		},
		{
			name:         "exact match on OpMove destination",
			staged:       []string{"work/epics/E-0002-dst"},
			ops:          moveOps,
			wantConflict: true,
		},
		{
			name:         "nested under the moved directory's source",
			staged:       []string{"work/epics/E-0001-src/M-0001-nested.md"},
			ops:          moveOps,
			wantConflict: true,
		},
		{
			name:         "nested under the moved directory's destination",
			staged:       []string{"work/epics/E-0002-dst/M-0001-nested.md"},
			ops:          moveOps,
			wantConflict: true,
		},
		{
			name:         "unrelated staged path outside the move",
			staged:       []string{"work/epics/E-0099-unrelated/epic.md"},
			ops:          moveOps,
			wantConflict: false,
		},
		{
			name:         "sibling directory with the source as a string prefix, not a path prefix",
			staged:       []string{"work/epics/E-0001-src-other/epic.md"},
			ops:          moveOps,
			wantConflict: false,
		},
		{
			name:         "OpWrite exact match",
			staged:       []string{"work/epics/E-0003/epic.md"},
			ops:          []FileOp{{Type: OpWrite, Path: "work/epics/E-0003/epic.md"}},
			wantConflict: true,
		},
		{
			name:         "OpWrite does not match a nested path (no directory semantics for writes)",
			staged:       []string{"work/epics/E-0003/epic.md/nested"},
			ops:          []FileOp{{Type: OpWrite, Path: "work/epics/E-0003/epic.md"}},
			wantConflict: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := checkStagedConflict(tt.staged, tt.ops)
			if tt.wantConflict && err == nil {
				t.Errorf("checkStagedConflict(%v, ops) = nil, want a conflict error", tt.staged)
			}
			if !tt.wantConflict && err != nil {
				t.Errorf("checkStagedConflict(%v, ops) = %v, want nil (unrelated path)", tt.staged, err)
			}
		})
	}
}

// TestGatherCommitOps_MoveDestinationMissing drives gatherCommitOps
// directly (the friend-assembly technique also used for
// commitTreeFromParent): an OpMove whose NewPath was never actually
// created on disk fails os.Lstat before any read is attempted. Not
// reachable through a real Apply() call (Phase 1's own os.Rename would
// already have failed), but a real branch in gatherCommitOps' own
// contract worth pinning directly.
func TestGatherCommitOps_MoveDestinationMissing(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	plan := &Plan{Ops: []FileOp{
		{Type: OpMove, Path: "old.md", NewPath: "missing.md"},
	}}

	_, _, err := gatherCommitOps(root, plan)
	if err == nil {
		t.Fatal("want error for a move destination that doesn't exist, got nil")
	}
	if !strings.Contains(err.Error(), "stat missing.md") {
		t.Errorf("error %q should mention stat missing.md", err.Error())
	}
}

// TestGatherCommitOps_MoveDestinationUnreadable: the move destination
// exists as a regular file but isn't readable, failing addFile's
// os.ReadFile — the flat-file (non-directory) path through addFile.
func TestGatherCommitOps_MoveDestinationUnreadable(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	root := t.TempDir()
	full := filepath.Join(root, "moved.md")
	if err := os.WriteFile(full, []byte("content"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(full, 0o644) })

	plan := &Plan{Ops: []FileOp{
		{Type: OpMove, Path: "old.md", NewPath: "moved.md"},
	}}
	_, _, err := gatherCommitOps(root, plan)
	if err == nil {
		t.Fatal("want error for an unreadable move destination, got nil")
	}
	if !strings.Contains(err.Error(), "reading moved.md for commit") {
		t.Errorf("error %q should mention reading moved.md for commit", err.Error())
	}
}

// TestGatherCommitOps_WriteDestinationIsDirectory: an OpWrite whose
// Path is a directory (not a regular file) fails addFile's
// os.ReadFile with "is a directory" — a scenario gatherCommitOps can
// hit directly but a real Apply() call cannot (Phase 2's own
// AtomicWriteFile would already have failed trying to rename a file
// over an existing directory).
func TestGatherCommitOps_WriteDestinationIsDirectory(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "adir"), 0o755); err != nil {
		t.Fatal(err)
	}

	plan := &Plan{Ops: []FileOp{
		{Type: OpWrite, Path: "adir", Content: []byte("x")},
	}}
	_, _, err := gatherCommitOps(root, plan)
	if err == nil {
		t.Fatal("want error for a write destination that's a directory, got nil")
	}
	if !strings.Contains(err.Error(), "reading adir for commit") {
		t.Errorf("error %q should mention reading adir for commit", err.Error())
	}
}

// TestGatherCommitOps_MoveDirectoryWalkPermissionDenied covers both the
// WalkDir callback's own error-propagation branch and gatherCommitOps'
// outer walk-error wrap: a directory OpMove whose destination contains
// a permission-denied subdirectory fails the recursive walk.
func TestGatherCommitOps_MoveDirectoryWalkPermissionDenied(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	root := t.TempDir()
	destDir := filepath.Join(root, "epic-new")
	blockedDir := filepath.Join(destDir, "blocked")
	if err := os.MkdirAll(blockedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blockedDir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blockedDir, 0o755) })

	plan := &Plan{Ops: []FileOp{
		{Type: OpMove, Path: "epic-old", NewPath: "epic-new"},
	}}
	_, _, err := gatherCommitOps(root, plan)
	if err == nil {
		t.Fatal("want error for a permission-denied subdirectory, got nil")
	}
	if !strings.Contains(err.Error(), "walking epic-new for commit") {
		t.Errorf("error %q should mention walking epic-new for commit", err.Error())
	}
}

// TestRollback_WriteBackErrorIsCaptured: the rollback write-back arm
// (captured pre-Apply bytes exist, so rollback restores them) fails
// when the parent directory can't be written to. Mirrors
// TestRollback_RemoveErrorIsCapturedWhenRestoreSucceeds, which pins the
// sibling os.Remove arm; this is the AtomicWriteFile arm.
func TestRollback_WriteBackErrorIsCaptured(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	root := t.TempDir()
	parent := filepath.Join(root, "locked")
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	full := filepath.Join(parent, "file.md")
	if err := os.WriteFile(full, []byte("current"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })

	rel := filepath.Join("locked", "file.md")
	tx := &applyTx{
		root:    root,
		ctx:     context.Background(),
		journal: []undoStep{writeUndo{path: rel, existed: true, content: []byte("pre-apply content")}},
	}
	err := tx.rollback()
	if err == nil {
		t.Fatal("expected rollback to capture the write-back error")
	}
	if !strings.Contains(err.Error(), "restoring") {
		t.Errorf("expected error to mention 'restoring'; got: %v", err)
	}
}

// TestLockContentionHint_GitDirFailsFallsBackToDotGit: a non-repo
// workdir makes gitops.GitDir fail; lockContentionHint falls back to
// workdir/.git, finds no lock there, and returns "".
func TestLockContentionHint_GitDirFailsFallsBackToDotGit(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if got := lockContentionHint(context.Background(), root); got != "" {
		t.Errorf("lockContentionHint on a non-repo dir = %q, want \"\"", got)
	}
}

// TestLockContentionHint_NoLockPresent: a real repo with no
// .git/index.lock file returns "" — the lock cleared, nothing to
// report.
func TestLockContentionHint_NoLockPresent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatal(err)
	}
	if got := lockContentionHint(ctx, root); got != "" {
		t.Errorf("lockContentionHint with no lock present = %q, want \"\"", got)
	}
}

// TestLockContentionHint_LsofMissing: an empty PATH makes
// exec.LookPath("lsof") fail, falling back to the bare "" hint even
// though a lock file is present.
func TestLockContentionHint_LsofMissing(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatal(err)
	}
	gitDir, err := gitops.GitDir(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(gitDir, "index.lock")
	if err := os.WriteFile(lockPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(lockPath) })

	t.Setenv("PATH", "")
	if got := lockContentionHint(ctx, root); got != "" {
		t.Errorf("lockContentionHint with lsof missing = %q, want \"\"", got)
	}
}

// fakeLsofOnPath installs a fake `lsof` script on PATH (in a
// t.TempDir prepended ahead of the real PATH) that prints the given
// stdout and exits 0. Returns nothing; callers just need it on PATH
// for the duration of the test (t.Setenv already restores PATH after).
func fakeLsofOnPath(t *testing.T, stdout string) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\nprintf '%s'\n"
	script = strings.Replace(script, "%s", strings.ReplaceAll(stdout, "'", `'\''`), 1)
	path := filepath.Join(dir, "lsof")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// TestLockContentionHint_LsofOutputDoesNotParse: lsof runs and exits
// 0, but its output doesn't parse to a PID (parseLsof returns ""),
// falling back to the bare "" hint.
func TestLockContentionHint_LsofOutputDoesNotParse(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatal(err)
	}
	gitDir, err := gitops.GitDir(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(gitDir, "index.lock")
	if err := os.WriteFile(lockPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(lockPath) })

	fakeLsofOnPath(t, "COMMAND   PID  USER ...\n")
	if got := lockContentionHint(ctx, root); got != "" {
		t.Errorf("lockContentionHint with unparseable lsof output = %q, want \"\"", got)
	}
}

// TestLockContentionHint_LsofFails: lsof is on PATH but exits non-zero
// (e.g. permission denied querying open files), falling back to the
// bare "" hint.
func TestLockContentionHint_LsofFails(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatal(err)
	}
	gitDir, err := gitops.GitDir(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(gitDir, "index.lock")
	if err := os.WriteFile(lockPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(lockPath) })

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "lsof"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if got := lockContentionHint(ctx, root); got != "" {
		t.Errorf("lockContentionHint with a failing lsof = %q, want \"\"", got)
	}
}

// TestRollback_DirMoveSkippedWhenDestinationGone: a tracked directory
// move whose destination no longer exists at rollback time (removed by
// something else in between) is silently skipped — nothing to reverse.
func TestRollback_DirMoveSkippedWhenDestinationGone(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tx := &applyTx{
		root:    root,
		journal: []undoStep{moveUndo{from: "old-dir", to: "never-existed-dir"}},
	}
	if err := tx.rollback(); err != nil {
		t.Errorf("rollback with a missing move destination = %v, want nil", err)
	}
}

// TestRollback_DirMoveReverseFails drives applyTx.rollback directly
// (friend assembly, mirroring the sibling capture/remove-error tests):
// the directory move's destination exists, but the source's parent
// directory denies write access, so the reversing os.Rename fails.
func TestRollback_DirMoveReverseFails(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	root := t.TempDir()
	toDir := filepath.Join(root, "moved")
	if err := os.Mkdir(toDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lockedParent := filepath.Join(root, "locked")
	if err := os.Mkdir(lockedParent, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(lockedParent, 0o755) })

	tx := &applyTx{
		root:    root,
		journal: []undoStep{moveUndo{from: filepath.Join("locked", "orig"), to: "moved"}},
	}
	err := tx.rollback()
	if err == nil {
		t.Fatal("expected rollback to capture the directory-move reversal error")
	}
	if !strings.Contains(err.Error(), "reversing move") {
		t.Errorf("expected error to mention 'reversing move'; got: %v", err)
	}
}

// TestCheckNoGitOperationInProgress_GitDirResolutionFails: outside a
// git repo, gitops.GitDir itself fails, and checkNoGitOperationInProgress
// must surface that failure rather than silently treating "no gitdir"
// as "no operation in progress" (G-0329's guard is meaningless if it
// degrades to a no-op the moment its own precondition can't resolve).
func TestCheckNoGitOperationInProgress_GitDirResolutionFails(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	err := checkNoGitOperationInProgress(context.Background(), root)
	if err == nil {
		t.Fatal("expected an error resolving the git dir outside a repo")
	}
	if !strings.Contains(err.Error(), "checking for an in-progress git operation") {
		t.Errorf("error %q should mention checking for an in-progress git operation", err.Error())
	}
}

// TestCheckNoGitOperationInProgress_CleanRepoPasses: a freshly
// initialized repo with no pending operation must not be flagged —
// the guard's true baseline, distinct from every "it fires" test.
func TestCheckNoGitOperationInProgress_CleanRepoPasses(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ctx := context.Background()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatal(err)
	}
	if err := checkNoGitOperationInProgress(ctx, root); err != nil {
		t.Errorf("clean repo flagged as having an operation in progress: %v", err)
	}
}
