package initcmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/initcmd"
	"github.com/23min/aiwf/internal/skills"
)

// M-0256/AC-1 backfill (folded in after initcmd wasn't assigned to any
// milestone during planning): Run's resolveInitRoot guard is
// `//coverage:ignore`d in initcmd.go itself — unlike cliutil.ResolveRoot,
// it only wraps filepath.Abs (explicit --root) or os.Getwd (no --root),
// neither portably triggerable. Every other flagged branch below is
// genuinely triggerable.
//
// Serial: TestRun_InitFailure uses testutil.BrokenGitIdentity, which
// uses t.Setenv, which panics under t.Parallel.

// TestRun_InitFailure covers Run's bare initrepo.Init guard: no local
// git identity configured, combined with BrokenGitIdentity's broken
// global git config, makes ensureConfig's deriveActor call fail.
func TestRun_InitFailure(t *testing.T) {
	testutil.BrokenGitIdentity(t)
	root := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	rc := initcmd.Run(root, "", false, false, false, "", false, false, nil, nil)
	if rc != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", rc)
	}
}

// TestRun_HookMigrationCollision covers the res.HookConflict branch
// (G45): an alien pre-existing commit-msg hook AND its `.local`
// sibling both present makes the migration refuse to pick a side,
// mirroring internal/initrepo/commitmsg_test.go's own
// TestEnsureCommitMsgHook_MigrationCollision fixture at the initcmd
// entry point.
func TestRun_HookMigrationCollision(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alien := []byte("#!/bin/sh\n# alien commit-msg\nexit 0\n")
	if err := os.WriteFile(filepath.Join(hooksDir, "commit-msg"), alien, 0o755); err != nil {
		t.Fatal(err)
	}
	prior := []byte("#!/bin/sh\n# prior commit-msg.local\nexit 0\n")
	if err := os.WriteFile(filepath.Join(hooksDir, "commit-msg.local"), prior, 0o755); err != nil {
		t.Fatal(err)
	}
	rc := initcmd.Run(root, "", false, false, false, "", false, false, nil, skills.ShippedHooks)
	if rc != cliutil.ExitFindings {
		t.Errorf("rc = %d, want ExitFindings", rc)
	}
}

// TestRun_StatuslineDryRun covers the statusline-scaffold dry-run
// branch: --statusline combined with --dry-run reports without
// scaffolding.
func TestRun_StatuslineDryRun(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	rc := initcmd.Run(root, "", true, false, true, string(skills.StatuslineScopeUser), false, false, nil, nil)
	if rc != cliutil.ExitOK {
		t.Errorf("rc = %d, want ExitOK", rc)
	}
}
