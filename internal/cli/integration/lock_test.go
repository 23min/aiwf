package integration

import (
	"sync"
	"testing"
	"time"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/repolock"
)

// TestRun_ConcurrentMutations_ContenderWaitsForAReleasedLock pins the
// half of the lock contract a busy refusal cannot show: a mutating
// verb whose lock is contended waits for it, and proceeds once the
// holder releases. Without this, a zero wait — refuse the moment the
// lock is held — satisfies every other test in this package, including
// TestRun_ConcurrentMutations_OneWinsOneBusy below, which holds the
// lock for the whole invocation and so cannot tell a verb that waited
// two seconds from one that never waited at all.
//
// This arm pins the wiring: that a real verb blocks and then resumes
// on the shared helper's terms. A verb that took the lock itself with
// its own zero timeout would still refuse under OneWinsOneBusy, and
// would leave the helper the cliutil arm exercises untouched, so this
// is the only place that defect surfaces. The wait itself is pinned one
// layer down by TestAcquireRepoLock_WaitsForAReleasedLock, where the
// window between launching the contender and its lock attempt is a
// single function call rather than a whole cobra dispatch, so the
// observation stays sharp under load the way this one cannot.
//
// The claim is deliberately narrow: one contender, one release, one
// success. How many concurrent actors get through, and how fast, is a
// property of the machine rather than of aiwf (G-0468).
func TestRun_ConcurrentMutations_ContenderWaitsForAReleasedLock(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}

	// heldFor is how long the lock stays held once the contender is
	// running, and doubles as a floor under the wait: a verb that waits
	// longer than this cannot return inside the window, so returning
	// inside it is the defect. The margin the window has to cover is
	// the contender's prelude — cobra dispatch, root and actor
	// resolution — before it reaches the lock at all; a machine loaded
	// enough to stretch that prelude past the window leaves the test
	// passing without exercising the property, which is why the sharp
	// pin lives in cliutil rather than here.
	const heldFor = 500 * time.Millisecond

	release := testutil.HoldRepoLock(t, root)
	// The release func is idempotent, so this frees the lock on every
	// early exit and is a no-op after the deliberate release below.
	defer release()

	// started fires from inside the goroutine, so the window below is
	// measured from when the contender is actually running rather than
	// from when it was queued.
	started := make(chan struct{})
	done := make(chan int, 1)
	go func() {
		close(started)
		done <- cli.Execute([]string{"add", "epic", "--title", "Contender", "--root", root, "--actor", "human/test"})
	}()
	<-started

	select {
	case rc := <-done:
		if rc == cliutil.ExitOK {
			t.Fatal("add succeeded while another process held the lock; the verb never took the lock at all")
		}
		t.Fatalf("add returned rc=%d while the lock was held; a mutating verb must wait for a contended lock, not refuse at once — this also fires if cliutil's lockTimeout has dropped below the %v this test holds for", rc, heldFor)
	case <-time.After(heldFor):
	}

	release()

	select {
	case rc := <-done:
		if rc != cliutil.ExitOK {
			t.Errorf("contender returned rc=%d, want %d; it should have taken the lock once the holder released it", rc, cliutil.ExitOK)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("add did not return within 15s of the lock being released")
	}
}

// TestRun_ConcurrentMutations_OneWinsOneBusy is the load-bearing
// test for G4: two `aiwf add` invocations against the same repo
// must not both succeed in allocating the next id. With the
// repolock guard, exactly one wins and one returns cliutil.ExitUsage with
// a busy message.
func TestRun_ConcurrentMutations_OneWinsOneBusy(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}

	// Pre-acquire the lock to make the test deterministic: the in-process
	// `aiwf add` invocation blocks on Acquire and times out, returning
	// cliutil.ExitUsage. Without the guard, it would proceed and produce a
	// successful add. The lock is held for the whole invocation, which is
	// what makes the refusal certain — and what makes the 5s bound below
	// the ceiling on the wait, the two tests above pinning its floor.
	preLock, err := repolock.Acquire(root, 0)
	if err != nil {
		t.Fatalf("pre-acquire: %v", err)
	}

	var wg sync.WaitGroup
	var rc int
	wg.Add(1)
	go func() {
		defer wg.Done()
		rc = cli.Execute([]string{"add", "epic", "--title", "Test", "--root", root, "--actor", "human/test"})
	}()

	// Wait for the goroutine to finish (it should time out and return).
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("aiwf add did not return within 5s; lock acquisition seems unbounded")
	}

	if rc != cliutil.ExitUsage {
		t.Errorf("locked-out add returned rc=%d, want %d (cliutil.ExitUsage); the lock guard is missing", rc, cliutil.ExitUsage)
	}

	if err := preLock.Release(); err != nil {
		t.Fatal(err)
	}

	// After release, a fresh add should succeed.
	if rc := cli.Execute([]string{"add", "epic", "--title", "After", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Errorf("post-release add returned rc=%d, want %d", rc, cliutil.ExitOK)
	}
}

// TestRun_Check_DoesNotAcquireLock: read-only check must work even
// while a mutation lock is held — concurrent reads/writes are fine.
func TestRun_Check_DoesNotAcquireLock(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}

	preLock, err := repolock.Acquire(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer preLock.Release()

	// check should return promptly with cliutil.ExitOK regardless of the lock.
	done := make(chan int, 1)
	go func() {
		done <- cli.Execute([]string{"check", "--root", root})
	}()
	select {
	case rc := <-done:
		if rc != cliutil.ExitOK {
			t.Errorf("check rc=%d, want cliutil.ExitOK; check should not acquire the mutation lock", rc)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("check blocked on the mutation lock; should be lock-free")
	}
}
