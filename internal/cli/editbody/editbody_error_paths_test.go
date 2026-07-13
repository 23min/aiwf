package editbody_test

import (
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/editbody"
)

// M-0253/AC-1 backfill: editbody.Run's ResolveRoot and
// LoadTreeWithTrunk fatal-IO branches are `//coverage:ignore`d in
// editbody.go itself, mirroring the established internal/cli/archive,
// internal/cli/renamearea, and internal/cli/setarea precedent. The
// two remaining flagged branches — the --body-file read guard and the
// actor-resolution guard — get real tests below.

// TestRun_BodyFileReadFailure covers Run's --body-file read guard: a
// nonexistent --body-file path makes cliutil.ReadBodyFile fail before
// any root/tree work runs.
func TestRun_BodyFileReadFailure(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "does-not-exist.md")
	var out cliutil.OutputFormat
	rc := editbody.Run("G-0001", "", "", "", "", missing, out)
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_ResolveActorFailure covers Run's cliutil.ResolveActor guard
// using M-0252's BrokenGitIdentity fixture. Serial: BrokenGitIdentity
// uses t.Setenv, which panics under t.Parallel.
func TestRun_ResolveActorFailure(t *testing.T) {
	testutil.BrokenGitIdentity(t)
	root := t.TempDir()
	var out cliutil.OutputFormat
	rc := editbody.Run("G-0001", "", "", root, "", "", out)
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}
