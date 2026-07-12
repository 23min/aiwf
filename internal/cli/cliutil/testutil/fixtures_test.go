package testutil_test

import (
	"context"
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
