package initcmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
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
	rc := initcmd.Run(root, "", false, false, false, "", false, false, false, nil, nil)
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
	rc := initcmd.Run(root, "", false, false, false, "", false, false, false, nil, skills.ShippedHooks)
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
	rc := initcmd.Run(root, "", true, false, true, string(skills.StatuslineScopeUser), false, false, false, nil, nil)
	if rc != cliutil.ExitOK {
		t.Errorf("rc = %d, want ExitOK", rc)
	}
}

// TestRun_StatuslineFailureSetsExitCode pins that a failed statusline
// scaffold is init's exit code while the guidance report still ends the run.
// An unknown scope fails before anything is written, and disabled guidance
// maintenance keeps the run away from personal configuration.
//
// Serial: captures process stdout.
func TestRun_StatuslineFailureSetsExitCode(t *testing.T) {
	root := freshGitRepo(t)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("guidance:\n  enabled: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rc := cliutil.ExitOK
	output := string(testutil.CaptureStdout(t, func() {
		rc = initcmd.Run(root, "", false, true, true, "no-such-scope", false, true, true, nil, nil)
	}))
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	heading := slices.Index(lines, "Guidance:")
	if heading < 0 || !strings.HasPrefix(lines[len(lines)-1], "  ") || slices.ContainsFunc(lines[heading+1:], func(line string) bool { return !strings.HasPrefix(line, "  ") }) {
		t.Errorf("output does not end with the guidance section:\n%s", output)
	}
}

// TestRun_LaterFailureStillReportsGuidance makes the hook step fail after
// guidance ran: init exits with an internal error and still prints the
// guidance section. Disabled maintenance keeps the run hermetic.
//
// Serial: captures process stdout.
func TestRun_LaterFailureStillReportsGuidance(t *testing.T) {
	root := freshGitRepo(t)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("hosts: [codex]\nguidance:\n  enabled: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hooks := filepath.Join(root, ".git", "hooks")
	if err := os.RemoveAll(hooks); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hooks, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	rc := cliutil.ExitOK
	output := string(testutil.CaptureStdout(t, func() {
		rc = initcmd.Run(root, "", false, false, false, "", false, false, true, nil, nil)
	}))
	if rc != cliutil.ExitInternal || !strings.Contains(output, "\nGuidance:\n") {
		t.Fatalf("rc = %d, want ExitInternal with the guidance section:\n%s", rc, output)
	}
}
