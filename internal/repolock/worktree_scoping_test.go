//go:build !windows

package repolock

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// worktree_scoping_test.go — M-0241/AC-4: confirms directly that a
// linked git worktree's lockfile is NOT the main checkout's
// <root>/.git/aiwf.lock. A linked worktree's .git is a regular FILE
// (a "gitdir: <path>" pointer), not a directory, so
// lockfilePath's os.Stat(...).IsDir() check is false there and it
// falls through to the <root>/.aiwf.lock fallback — a path entirely
// outside the main checkout's .git/. This is the exact mechanism
// behind M-0241/AC-3's cross-worktree id race: two worktrees hold two
// entirely separate locks, so repolock provides no cross-worktree
// serialization. Read-only confirmation — no production code changes.

// initRepoWithWorktree creates a real git repo with one commit at
// root/main, then adds a real linked worktree at root/wt via `git
// worktree add`. Returns (mainRoot, worktreeRoot).
func initRepoWithWorktree(t *testing.T) (mainRoot, worktreeRoot string) {
	t.Helper()
	root := t.TempDir()
	mainRoot = filepath.Join(root, "main")
	worktreeRoot = filepath.Join(root, "wt")
	if err := os.MkdirAll(mainRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	runGitForTest(t, mainRoot, "init", "-q")
	runGitForTest(t, mainRoot, "config", "user.email", "test@example.com")
	runGitForTest(t, mainRoot, "config", "user.name", "test")
	runGitForTest(t, mainRoot, "commit", "-q", "--allow-empty", "-m", "seed")
	runGitForTest(t, mainRoot, "worktree", "add", "-q", "-b", "feature", worktreeRoot)
	return mainRoot, worktreeRoot
}

func runGitForTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// TestLockfilePath_LinkedWorktreeDotGitIsAFileNotADirectory pins the
// structural fact lockfilePath's fallback depends on: a linked
// worktree's .git is a regular file, never a directory — unlike the
// main checkout's .git.
func TestLockfilePath_LinkedWorktreeDotGitIsAFileNotADirectory(t *testing.T) {
	t.Parallel()
	mainRoot, worktreeRoot := initRepoWithWorktree(t)

	mainInfo, err := os.Stat(filepath.Join(mainRoot, ".git"))
	if err != nil {
		t.Fatalf("stat main .git: %v", err)
	}
	if !mainInfo.IsDir() {
		t.Fatal("expected the main checkout's .git to be a directory")
	}

	wtInfo, err := os.Stat(filepath.Join(worktreeRoot, ".git"))
	if err != nil {
		t.Fatalf("stat worktree .git: %v", err)
	}
	if wtInfo.IsDir() {
		t.Fatal("expected the linked worktree's .git to be a regular file (a gitdir pointer), not a directory")
	}
}

// TestLockfilePath_LinkedWorktreeResolvesToItsOwnFallbackNotMains
// confirms the consequence directly: because .git is a file there,
// lockfilePath falls back to <worktreeRoot>/.aiwf.lock — never
// <mainRoot>/.git/aiwf.lock.
func TestLockfilePath_LinkedWorktreeResolvesToItsOwnFallbackNotMains(t *testing.T) {
	t.Parallel()
	mainRoot, worktreeRoot := initRepoWithWorktree(t)

	mainLock, err := lockfilePath(mainRoot)
	if err != nil {
		t.Fatalf("lockfilePath(main): %v", err)
	}
	wantMain := filepath.Join(mainRoot, ".git", "aiwf.lock")
	if mainLock != wantMain {
		t.Fatalf("lockfilePath(main) = %q, want %q", mainLock, wantMain)
	}

	wtLock, err := lockfilePath(worktreeRoot)
	if err != nil {
		t.Fatalf("lockfilePath(worktree): %v", err)
	}
	wantWt := filepath.Join(worktreeRoot, ".aiwf.lock")
	if wtLock != wantWt {
		t.Fatalf("lockfilePath(worktree) = %q, want %q (the fallback, not something under the main checkout's .git/)", wtLock, wantWt)
	}
	if wtLock == mainLock {
		t.Fatal("expected the worktree's lockfile path to differ from the main checkout's")
	}
}

// TestAcquire_MainCheckoutAndLinkedWorktreeDoNotContend is the
// behavioral confirmation: holding the main checkout's lock does NOT
// block a concurrent Acquire in the linked worktree — repolock
// provides no cross-worktree serialization, exactly the mechanism
// M-0241/AC-3's cross-worktree id race depends on. Contrast
// TestAcquire_Twice_SecondGetsErrBusy (repolock_test.go), which pins
// that the SAME root genuinely does contend — the positive control
// ruling out "repolock's flock is a no-op" as the explanation here.
func TestAcquire_MainCheckoutAndLinkedWorktreeDoNotContend(t *testing.T) {
	t.Parallel()
	mainRoot, worktreeRoot := initRepoWithWorktree(t)

	mainLock, err := Acquire(mainRoot, 0)
	if err != nil {
		t.Fatalf("Acquire(main): %v", err)
	}
	defer mainLock.Release()

	wtLock, err := Acquire(worktreeRoot, 0)
	if err != nil {
		t.Fatalf("Acquire(worktree) should succeed immediately while the main checkout's lock is held; got %v", err)
	}
	_ = wtLock.Release()
}
