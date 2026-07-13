package editbody_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/editbody"
)

// M-0253/AC-1 backfill: editbody.Run's ResolveRoot fatal-IO branch is
// `//coverage:ignore`d in editbody.go itself, mirroring the
// established internal/cli/archive, internal/cli/renamearea, and
// internal/cli/setarea precedent. The three remaining flagged
// branches — the --body-file read guard, the actor-resolution guard,
// and the LoadTreeWithTrunk config-parse guard — get real tests below.

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

// TestRun_LoadTreeWithTrunkConfigParseFailure covers Run's
// cliutil.LoadTreeWithTrunk guard: a syntactically malformed
// aiwf.yaml makes config.Load return a parse error that
// LoadTreeWithTrunk propagates as-is (config.go's Load only swallows
// config.ErrNotFound — a missing file — not a parse failure). This is
// distinct from tree.Load's per-file LoadError case: a malformed
// aiwf.yaml is a fatal load error, not a findings-shaped one.
func TestRun_LoadTreeWithTrunkConfigParseFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"),
		[]byte("areas:\n  members: [unclosed\n"), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	var out cliutil.OutputFormat
	rc := editbody.Run("G-0001", "human/test", "", root, "", "", out)
	if rc != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", rc)
	}
}
