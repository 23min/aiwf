package reallocate_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/reallocate"
)

// M-0253/AC-1 backfill: reallocate.Run's ResolveRoot and
// LoadTreeWithTrunk fatal-IO branches are `//coverage:ignore`d in
// reallocate.go itself, mirroring the established internal/cli/archive
// and wave-1 internal/cli/add/internal/cli/editbody precedent. The
// one remaining flagged branch — the actor-resolution guard — gets a
// real test below.

// TestRun_ResolveActorFailure covers Run's cliutil.ResolveActor guard
// using M-0252's BrokenGitIdentity fixture. Serial: BrokenGitIdentity
// uses t.Setenv, which panics under t.Parallel.
func TestRun_ResolveActorFailure(t *testing.T) {
	testutil.BrokenGitIdentity(t)
	root := t.TempDir()
	var out cliutil.OutputFormat
	rc := reallocate.Run("E-0001", "", "", root, out)
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}
