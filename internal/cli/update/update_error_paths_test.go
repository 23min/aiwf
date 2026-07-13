package update_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/update"
)

// M-0253/AC-1 backfill: update.Run's ResolveRoot fatal-IO branch is
// `//coverage:ignore`d in update.go itself, mirroring the established
// internal/cli/archive and wave-1/wave-2 precedent. update.Run's
// signature carries no actor/principal params at all — it never
// resolves an actor, so M-0252's BrokenGitIdentity fixture doesn't
// apply to this verb. The one remaining flagged branch is update.go's
// own hook-chain-collision guard (G45, `if conflict`), reached past a
// successful root/lock/config sequence, which gets a real test below.

// TestRun_HookChainCollisionReturnsFindings covers the `if conflict`
// branch (G45): a non-aiwf pre-push hook AND a pre-existing
// pre-push.local sibling makes RefreshArtifacts refuse to migrate the
// alien hook (it would clobber the .local file), and Run must surface
// that as cliutil.ExitFindings rather than silently proceeding.
// Mirrors internal/initrepo's own
// TestInit_RefusesPreHookMigrationOnCollision fixture shape, driven
// through update.Run instead of initrepo.Init directly. Reuses
// hooks_test.go's freshInitializedRepo fixture (same package).
func TestRun_HookChainCollisionReturnsFindings(t *testing.T) {
	t.Parallel()
	root := freshInitializedRepo(t)
	hookDir := filepath.Join(root, ".git", "hooks")
	alien := []byte("#!/bin/sh\n# alien\nexit 0\n")
	prior := []byte("#!/bin/sh\n# prior local\nexit 0\n")
	if err := os.WriteFile(filepath.Join(hookDir, "pre-push"), alien, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hookDir, "pre-push.local"), prior, 0o755); err != nil {
		t.Fatal(err)
	}

	rc := update.Run(root, false, "", false, false, false, false, nil, nil)
	if rc != cliutil.ExitFindings {
		t.Errorf("rc = %d, want ExitFindings", rc)
	}
}
