//go:build !windows

package repolock

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// initRepo makes a directory look enough like a git repo for
// repolock to find a place for the lockfile (.git/ subdir).
func initRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestAcquire_Once_OK(t *testing.T) {
	root := initRepo(t)
	l, err := Acquire(root, 0)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if err := l.Release(); err != nil {
		t.Errorf("Release: %v", err)
	}
}

func TestAcquire_Twice_SecondGetsErrBusy(t *testing.T) {
	root := initRepo(t)
	l1, err := Acquire(root, 0)
	if err != nil {
		t.Fatalf("Acquire 1: %v", err)
	}
	defer l1.Release()

	_, err = Acquire(root, 0)
	if !errors.Is(err, ErrBusy) {
		t.Errorf("Acquire 2: got %v, want ErrBusy", err)
	}
}

func TestAcquire_Twice_SecondWaitsThenSucceeds(t *testing.T) {
	root := initRepo(t)
	l1, err := Acquire(root, 0)
	if err != nil {
		t.Fatal(err)
	}

	type res struct {
		l   *Lock
		err error
	}
	out := make(chan res, 1)
	go func() {
		l2, err := Acquire(root, 2*time.Second)
		out <- res{l2, err}
	}()
	// Give goroutine a moment to start polling, then release.
	time.Sleep(100 * time.Millisecond)
	if err := l1.Release(); err != nil {
		t.Fatal(err)
	}
	r := <-out
	if r.err != nil {
		t.Errorf("Acquire 2 should have succeeded after release; got %v", r.err)
	}
	if r.l != nil {
		_ = r.l.Release()
	}
}

func TestAcquire_DifferentRoots_NoConflict(t *testing.T) {
	a := initRepo(t)
	b := initRepo(t)
	la, err := Acquire(a, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer la.Release()
	lb, err := Acquire(b, 0)
	if err != nil {
		t.Errorf("Acquire on different root should succeed; got %v", err)
	}
	if lb != nil {
		_ = lb.Release()
	}
}

func TestAcquire_AfterRelease_ReAcquireOK(t *testing.T) {
	root := initRepo(t)
	l1, err := Acquire(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rerr := l1.Release(); rerr != nil {
		t.Fatal(rerr)
	}
	l2, err := Acquire(root, 0)
	if err != nil {
		t.Errorf("re-Acquire after Release should succeed; got %v", err)
	}
	if l2 != nil {
		_ = l2.Release()
	}
}

func TestAcquire_NonExistentRoot_Error(t *testing.T) {
	_, err := Acquire(filepath.Join(t.TempDir(), "does", "not", "exist"), 0)
	if err == nil {
		t.Error("Acquire on missing root should error")
	}
	if errors.Is(err, ErrBusy) {
		t.Error("missing-root error must not be ErrBusy")
	}
}

func TestAcquire_TimeoutRespected(t *testing.T) {
	root := initRepo(t)
	l1, err := Acquire(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer l1.Release()

	start := time.Now()
	_, err = Acquire(root, 250*time.Millisecond)
	elapsed := time.Since(start)
	if !errors.Is(err, ErrBusy) {
		t.Errorf("expected ErrBusy after timeout; got %v", err)
	}
	if elapsed < 200*time.Millisecond {
		t.Errorf("returned too fast (%v); should have waited ~250ms", elapsed)
	}
	if elapsed > 1*time.Second {
		t.Errorf("returned too slow (%v); timeout not respected", elapsed)
	}
}

// TestAcquire_DotGitMissing_FallsBackToRoot: when .git/ doesn't exist
// (e.g. running aiwf in a directory that's not yet a git repo, or
// in tests), the lockfile lives at <root>/.aiwf.lock instead.
func TestAcquire_DotGitMissing_FallsBackToRoot(t *testing.T) {
	root := t.TempDir() // no .git
	l, err := Acquire(root, 0)
	if err != nil {
		t.Fatalf("Acquire on non-git root should succeed (fallback): %v", err)
	}
	defer l.Release()
	// And a second Acquire should still see it.
	if _, err := Acquire(root, 0); !errors.Is(err, ErrBusy) {
		t.Errorf("second Acquire should see ErrBusy; got %v", err)
	}
}

// TestRelease_Idempotent: calling Release a second time is a no-op
// rather than panicking on a closed fd.
func TestRelease_Idempotent(t *testing.T) {
	root := initRepo(t)
	l, err := Acquire(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Release(); err != nil {
		t.Errorf("first Release: %v", err)
	}
	if err := l.Release(); err != nil {
		t.Errorf("second Release should be no-op; got %v", err)
	}
}

// TestRelease_NilLockSafe: Release on a zero-value Lock is a no-op
// (used by deferred cleanup paths where Acquire may have failed).
func TestRelease_NilLockSafe(t *testing.T) {
	var l *Lock
	if err := l.Release(); err != nil {
		t.Errorf("nil Release should be no-op; got %v", err)
	}
}

// TestAcquire_OpenFailure_PropagatesError: when the lockfile dir is
// not writable, OpenFile fails and we get a wrapped error (not
// ErrBusy).
func TestAcquire_OpenFailure_PropagatesError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	root := initRepo(t)
	// Make .git/ read-only so OpenFile of .git/aiwf.lock fails.
	gitDir := filepath.Join(root, ".git")
	if err := os.Chmod(gitDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(gitDir, 0o755) })

	_, err := Acquire(root, 0)
	if err == nil {
		t.Fatal("expected error opening unwritable lockfile")
	}
	if errors.Is(err, ErrBusy) {
		t.Errorf("open failure must not be ErrBusy; got %v", err)
	}
}

// TestParallelAcquireRace: 20 goroutines race to acquire the same
// lock; exactly one should succeed at a time.
func TestParallelAcquireRace(t *testing.T) {
	root := initRepo(t)

	var (
		wg            sync.WaitGroup
		concurrent    int
		maxConcurrent int
		mu            sync.Mutex
	)
	const N = 20
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l, err := Acquire(root, 5*time.Second)
			if err != nil {
				t.Errorf("Acquire: %v", err)
				return
			}
			mu.Lock()
			concurrent++
			if concurrent > maxConcurrent {
				maxConcurrent = concurrent
			}
			mu.Unlock()
			time.Sleep(5 * time.Millisecond)
			mu.Lock()
			concurrent--
			mu.Unlock()
			_ = l.Release()
		}()
	}
	wg.Wait()
	if maxConcurrent != 1 {
		t.Errorf("max concurrent holders = %d, want 1", maxConcurrent)
	}
}
