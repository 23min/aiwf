package testutil_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/tree"
)

// TestBrokenGitIdentity_TriggersResolveActorFailure proves the fixture
// actually forces cliutil.ResolveActor's git-config-derivation branch
// to fail, rather than merely asserting the fixture "looks right".
func TestBrokenGitIdentity_TriggersResolveActorFailure(t *testing.T) {
	testutil.BrokenGitIdentity(t)

	_, err := cliutil.ResolveActor("", "")
	if err == nil {
		t.Fatal("ResolveActor(\"\", \"\") = nil error, want an error under BrokenGitIdentity")
	}
}

// TestHoldRepoLock_TriggersAcquireRepoLockBusy proves the fixture
// forces cliutil.AcquireRepoLock's busy-contention branch: while the
// lock is held, a second acquisition attempt on the same root fails
// with cliutil.ExitUsage and a nil release func.
func TestHoldRepoLock_TriggersAcquireRepoLockBusy(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	release := testutil.HoldRepoLock(t, root)
	defer release()

	got, rc := cliutil.AcquireRepoLock(root, "aiwf test", cliutil.OutputFormat{})
	if got != nil {
		t.Errorf("AcquireRepoLock returned a non-nil release func while the lock was held")
	}
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want cliutil.ExitUsage", rc)
	}
}

// TestWriteMalformedEntity_TriggersLoadError proves the fixture forces
// tree.Load's per-file parse-error branch: the malformed file produces
// exactly one LoadError naming its path, and the fatal error return
// stays nil (parse errors are non-fatal per tree.Load's contract).
func TestWriteMalformedEntity_TriggersLoadError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	rel := filepath.Join("work", "epics", "E-0099-broken", "epic.md")
	testutil.WriteMalformedEntity(t, root, rel)

	_, loadErrs, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	if len(loadErrs) != 1 {
		t.Fatalf("loadErrs count = %d, want 1: %+v", len(loadErrs), loadErrs)
	}
	if loadErrs[0].Path != rel {
		t.Errorf("loadErrs[0].Path = %q, want %q", loadErrs[0].Path, rel)
	}
}

// TestInvalidFormat_TriggersReadVerbUsageError proves testutil.InvalidFormat
// forces a read-style verb's up-front format guard, using check.Run as
// the representative read-verb call site.
func TestInvalidFormat_TriggersReadVerbUsageError(t *testing.T) {
	t.Parallel()
	rc := check.Run("", testutil.InvalidFormat, false, "", false, false, false, nil, "")
	if rc != cliutil.ExitUsage {
		t.Errorf("check.Run with testutil.InvalidFormat: rc = %d, want cliutil.ExitUsage", rc)
	}
}

// TestHoldRepoLock_FatalsOnUnacquirableRoot proves HoldRepoLock's own
// defensive branch (fixtures.go:72): a root whose lockfile can't even
// be opened — the "does-not-exist" one-line pattern the fixture's own
// doc comment already documents for cliutil.AcquireRepoLock's "other
// error" branch — makes the underlying repolock.Acquire fail, and
// HoldRepoLock reports that via t.Fatalf rather than returning it.
//
// HoldRepoLock calls t.Fatalf on failure, so it can't be called
// directly against the real *testing.T and expect a normal return; see
// runAndCaptureFatal's doc comment for why a t.Run-based "assert the
// bool" wrapper is the wrong tool here (it would permanently fail this
// whole package's test suite, not just one subtest).
func TestHoldRepoLock_FatalsOnUnacquirableRoot(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "does-not-exist")

	failed := runAndCaptureFatal(func(t *testing.T) {
		t.Helper()
		testutil.HoldRepoLock(t, root)
	})
	if !failed {
		t.Fatal("expected HoldRepoLock to Fatalf on an unacquirable root")
	}
}

// TestWriteMalformedEntity_FatalsOnMkdirFailure proves
// WriteMalformedEntity's os.MkdirAll error branch (fixtures.go:95): a
// regular file pre-staged at the exact path component the fixture
// needs to MkdirAll makes the directory creation fail (a path
// component exists as a non-directory).
func TestWriteMalformedEntity_FatalsOnMkdirFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// relPath's first component ("work") is pre-created as a regular
	// file, so MkdirAll(".../work/epics/E-0099-broken") fails.
	if err := os.WriteFile(filepath.Join(root, "work"), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("pre-staging work as a file: %v", err)
	}
	relPath := filepath.Join("work", "epics", "E-0099-broken", "epic.md")

	failed := runAndCaptureFatal(func(t *testing.T) {
		t.Helper()
		testutil.WriteMalformedEntity(t, root, relPath)
	})
	if !failed {
		t.Fatal("expected WriteMalformedEntity to Fatalf when a path component is a non-directory")
	}
}

// TestWriteMalformedEntity_FatalsOnWriteFailure proves
// WriteMalformedEntity's os.WriteFile error branch (fixtures.go:104):
// pre-staging relPath itself as a directory makes MkdirAll(parent)
// succeed trivially but the subsequent WriteFile(relPath) fail because
// relPath already exists as a directory, not a file.
func TestWriteMalformedEntity_FatalsOnWriteFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	relPath := filepath.Join("work", "epics", "E-0099-broken", "epic.md")
	if err := os.MkdirAll(filepath.Join(root, relPath), 0o755); err != nil {
		t.Fatalf("pre-staging %s as a directory: %v", relPath, err)
	}

	failed := runAndCaptureFatal(func(t *testing.T) {
		t.Helper()
		testutil.WriteMalformedEntity(t, root, relPath)
	})
	if !failed {
		t.Fatal("expected WriteMalformedEntity to Fatalf when relPath already exists as a directory")
	}
}

// runAndCaptureFatal runs fn against a throwaway *testing.T in its own
// goroutine and reports whether that T ended up Failed(). This is the
// portable way to prove a t.Fatalf-guarded line inside a test-only
// fixture actually fires, without permanently failing the real test:
// t.Fatalf's FailNow calls common.Fail (which — per testing's own
// source, common.Fail recurses into c.parent.Fail() — marks every
// t.Run-linked ancestor failed too) and then runtime.Goexit, which
// unwinds only the calling goroutine. A *testing.T obtained via t.Run
// has its parent wired up, so a naive `ok := t.Run("inner", fn); if ok
// {...}` wrapper would still permanently fail this package's test
// suite the moment fn's Fatalf fires — not just the inner subtest. The
// throwaway T here was never obtained via t.Run, so its own Fail() has
// no parent to propagate to, and running it in a dedicated goroutine
// means the Goexit terminates only that goroutine, not the real
// test's.
func runAndCaptureFatal(fn func(t *testing.T)) (failed bool) {
	done := make(chan *testing.T, 1)
	go func() {
		fake := &testing.T{}
		defer func() { done <- fake }()
		fn(fake)
	}()
	fake := <-done
	return fake.Failed()
}
