package main

import (
	"sync"
	"testing"
	"time"

	"github.com/23min/ai-workflow-v2/internal/repolock"
)

// TestRun_ConcurrentMutations_OneWinsOneBusy is the load-bearing
// test for G4: two `aiwf add` invocations against the same repo
// must not both succeed in allocating the next id. With the
// repolock guard, exactly one wins and one returns exitUsage with
// a busy message.
func TestRun_ConcurrentMutations_OneWinsOneBusy(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}

	// Pre-acquire the lock to make the test deterministic: the in-process
	// `aiwf add` invocation will block on Acquire and time out after
	// lockTimeout (2s), returning exitUsage. Without the guard, it would
	// proceed and produce a successful add. We hold the lock for slightly
	// longer than lockTimeout to ensure timeout fires.
	preLock, err := repolock.Acquire(root, 0)
	if err != nil {
		t.Fatalf("pre-acquire: %v", err)
	}

	var wg sync.WaitGroup
	var rc int
	wg.Add(1)
	go func() {
		defer wg.Done()
		rc = run([]string{"add", "epic", "--title", "Test", "--root", root, "--actor", "human/test"})
	}()

	// Wait for the goroutine to finish (it should time out and return).
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("aiwf add did not return within 5s; lock acquisition seems unbounded")
	}

	if rc != exitUsage {
		t.Errorf("locked-out add returned rc=%d, want %d (exitUsage); the lock guard is missing", rc, exitUsage)
	}

	if err := preLock.Release(); err != nil {
		t.Fatal(err)
	}

	// After release, a fresh add should succeed.
	if rc := run([]string{"add", "epic", "--title", "After", "--root", root, "--actor", "human/test"}); rc != exitOK {
		t.Errorf("post-release add returned rc=%d, want %d", rc, exitOK)
	}
}

// TestRun_Check_DoesNotAcquireLock: read-only check must work even
// while a mutation lock is held — concurrent reads/writes are fine.
func TestRun_Check_DoesNotAcquireLock(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}

	preLock, err := repolock.Acquire(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer preLock.Release()

	// check should return promptly with exitOK regardless of the lock.
	done := make(chan int, 1)
	go func() {
		done <- run([]string{"check", "--root", root})
	}()
	select {
	case rc := <-done:
		if rc != exitOK {
			t.Errorf("check rc=%d, want exitOK; check should not acquire the mutation lock", rc)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("check blocked on the mutation lock; should be lock-free")
	}
}
