package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/initrepo"
)

// TestRun_PlainUpdateNeverCreatesStatusline asserts G-0344's guardrail:
// the upgrade-only auto-refresh only touches an *already-installed* copy;
// a plain `aiwf update` (no `--statusline`) must never scaffold a
// statusline where none exists. Initial install stays behind the explicit
// `--statusline` opt-in (ADR-0015 consent unchanged). Scoped to the
// tempdir project path, which is deterministic regardless of machine state.
func TestRun_PlainUpdateNeverCreatesStatusline(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	statuslinePath := filepath.Join(root, ".claude", "statusline.sh")
	if _, err := os.Stat(statuslinePath); err == nil {
		t.Fatalf("precondition: init must not scaffold a statusline without --statusline")
	}
	if rc := cli.Execute([]string{"update", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("update: %d", rc)
	}
	if _, err := os.Stat(statuslinePath); err == nil {
		t.Errorf("plain `aiwf update` must not create a statusline (found %s)", statuslinePath)
	}
}

// TestRun_UpdateMaterializes wipes a tampered skill file and verifies
// `aiwf update` restores the embedded content byte-for-byte.
func TestRun_UpdateMaterializes(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	skillPath := filepath.Join(root, ".claude", "skills", "aiwf-add", "SKILL.md")
	if err := os.WriteFile(skillPath, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := cli.Execute([]string{"update", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("update: %d", rc)
	}
	got, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "name: aiwf-add") {
		t.Errorf("aiwf-add not restored: %s", got)
	}
}

// TestRun_UpdateRefreshesPrePushHook removes a previously-installed
// pre-push hook and confirms `aiwf update` reinstalls it. Without
// the broadened update verb (step 5), this would fail because
// update only re-materialised skills.
func TestRun_UpdateRefreshesPrePushHook(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	// No --skip-hook: this test exercises update's hook-refresh
	// behavior and needs init to land a real hook first. The test
	// triggers no commits, so the embedded test-binary path never
	// fires as a hook.
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	hookPath := filepath.Join(root, ".git", "hooks", "pre-push")
	if err := os.Remove(hookPath); err != nil {
		t.Fatalf("removing pre-push hook: %v", err)
	}
	if rc := cli.Execute([]string{"update", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("update: %d", rc)
	}
	body, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("pre-push hook missing after update: %v", err)
	}
	if !strings.Contains(string(body), initrepo.HookMarker()) {
		t.Errorf("pre-push hook missing marker after update:\n%s", body)
	}
}

// TestRun_UpdateRefreshesPreCommitHook is the same property for the
// new pre-commit hook (default-on per status_md.auto_update).
func TestRun_UpdateRefreshesPreCommitHook(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	// No --skip-hook: same rationale as TestRun_UpdateRefreshesPrePushHook.
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	hookPath := filepath.Join(root, ".git", "hooks", "pre-commit")
	if err := os.Remove(hookPath); err != nil {
		t.Fatalf("removing pre-commit hook: %v", err)
	}
	if rc := cli.Execute([]string{"update", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("update: %d", rc)
	}
	body, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("pre-commit hook missing after update: %v", err)
	}
	if !strings.Contains(string(body), initrepo.PreCommitHookMarker()) {
		t.Errorf("pre-commit hook missing marker after update:\n%s", body)
	}
}

// TestRun_UpdateOptOutRemovesPostCommitKeepsGate (G42 + G-0112): run
// init (default install lays down both pre-commit and post-commit),
// flip status_md.auto_update: false, run update → the pre-commit
// hook stays installed (tree-discipline gate is enforcement, not
// opt-out-able) and never carries a regen step in any mode; the
// post-commit hook is removed (G-0112: that's where the regen toggle
// lives now).
func TestRun_UpdateOptOutRemovesPostCommitKeepsGate(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	// No --skip-hook: this test verifies G42 + G-0112 round-trip
	// behavior (install → opt-out → re-install) which needs real
	// hook installation through init. No commits triggered.
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	preCommit := filepath.Join(root, ".git", "hooks", "pre-commit")
	postCommit := filepath.Join(root, ".git", "hooks", "post-commit")
	if _, err := os.Stat(preCommit); err != nil {
		t.Fatalf("pre-commit hook not installed by default Init: %v", err)
	}
	if _, err := os.Stat(postCommit); err != nil {
		t.Fatalf("post-commit hook not installed by default Init (G-0112): %v", err)
	}

	// Flip the opt-out flag.
	yamlPath := filepath.Join(root, "aiwf.yaml")
	updated := []byte(`aiwf_version: 0.1.0
actor: human/test
status_md:
  auto_update: false
`)
	if err := os.WriteFile(yamlPath, updated, 0o644); err != nil {
		t.Fatalf("rewriting aiwf.yaml: %v", err)
	}

	if rc := cli.Execute([]string{"update", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("update: %d", rc)
	}
	body, err := os.ReadFile(preCommit)
	if err != nil {
		t.Fatalf("pre-commit hook missing after opt-out (G42 violation): %v", err)
	}
	if !strings.Contains(string(body), "check --shape-only") {
		t.Errorf("pre-commit hook missing tree-discipline gate after opt-out:\n%s", body)
	}
	if strings.Contains(string(body), "status --root") {
		t.Errorf("pre-commit hook still includes STATUS.md regen step (G-0112: regen lives in post-commit):\n%s", body)
	}
	if _, err := os.Stat(postCommit); !os.IsNotExist(err) {
		t.Errorf("post-commit hook should be removed under opt-out (G-0112) (stat err=%v)", err)
	}
}

// TestRun_UpdateMissingConfig: update against a directory with no
// aiwf.yaml is an internal error (config.Load returns ErrNotFound,
// which `aiwf update` cannot continue past — the StatusMdAutoUpdate
// flag has nowhere to come from). The user is expected to run
// `aiwf init` first.
func TestRun_UpdateMissingConfig(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	// No init: aiwf.yaml is absent.
	if rc := cli.Execute([]string{"update", "--root", root}); rc != cliutil.ExitInternal {
		t.Errorf("rc = %d, want cliutil.ExitInternal (%d)", rc, cliutil.ExitInternal)
	}
}

// TestRun_UpdateFromWorktree_WritesSharedHooks (G-0136 / M-0133 /
// AC-2): when `aiwf update` runs from a linked git worktree, the
// hook write goes to the shared `<main>/.git/hooks/` (which git
// actually fires) and NOT the per-worktree `.git/worktrees/<id>/hooks/`
// (inert — pre-fix behavior). Output names the affects-all-worktrees
// scope so the operator isn't surprised.
//
// Uses testutil.CaptureRun (cannot t.Parallel — os.Stdout mutation).
func TestRun_UpdateFromWorktree_WritesSharedHooks(t *testing.T) {
	main := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", main, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	// Commit init's artifacts (aiwf.yaml, CLAUDE.md, .gitignore) so
	// the worktree's checkout sees them.
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "aiwf init artifacts"},
	} {
		c := exec.Command("git", args...)
		c.Dir = main
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	// Create a linked worktree off the main checkout.
	worktreePath := filepath.Join(t.TempDir(), "wt")
	wtCmd := exec.Command("git", "worktree", "add", "-b", "feat", worktreePath)
	wtCmd.Dir = main
	if out, err := wtCmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, out)
	}
	// Remove the pre-push hook so update has visible work to do.
	prePushSharedPath := filepath.Join(main, ".git", "hooks", "pre-push")
	if err := os.Remove(prePushSharedPath); err != nil {
		t.Fatalf("remove pre-push hook: %v", err)
	}
	// Run aiwf update from the worktree.
	rc, stdout, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"update", "--root", worktreePath})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("update from worktree: rc=%d\nstdout: %s\nstderr: %s", rc, stdout, stderr)
	}
	// (a) Shared hooks dir was touched — pre-push hook reinstalled.
	if _, err := os.Stat(prePushSharedPath); err != nil {
		t.Errorf("shared pre-push hook not present after update from worktree: %v", err)
	}
	// (b) Per-worktree hooks dir was NOT touched. git's worktree
	// metadata lives at <main>/.git/worktrees/<id>/; the hooks/
	// subdir under it is what HooksDir incorrectly returned pre-fix.
	perWorktreeHooks := filepath.Join(main, ".git", "worktrees", "wt", "hooks")
	if _, err := os.Stat(filepath.Join(perWorktreeHooks, "pre-push")); !os.IsNotExist(err) {
		t.Errorf("per-worktree pre-push hook should not be created (stat err=%v); update from a worktree must write only to the shared dir per G-0136", err)
	}
	// (c) Output names the affects-all-worktrees scope.
	combined := stdout + stderr
	if !strings.Contains(combined, "affects all worktrees") {
		t.Errorf("update output should mention 'affects all worktrees' notice when run from a worktree:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
}

// TestRun_UpdateRemoveDeletesAiwfAuthoredWiring is the end-to-end
// happy path for G-0354: `--statusline --wire-settings` installs the
// project-scope script + settings key, and a follow-up
// `--scope project --remove` deletes both without needing --force
// because they look aiwf-authored.
func TestRun_UpdateRemoveDeletesAiwfAuthoredWiring(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := cli.Execute([]string{"update", "--root", root, "--statusline", "--scope", "project", "--wire-settings"}); rc != cliutil.ExitOK {
		t.Fatalf("update --statusline: %d", rc)
	}

	scriptPath := filepath.Join(root, ".claude", "statusline.sh")
	settingsPath := filepath.Join(root, ".claude", "settings.local.json")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("precondition: statusline script must exist after --statusline: %v", err)
	}
	settingsBefore, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("precondition: settings file must exist after --wire-settings: %v", err)
	}
	if !strings.Contains(string(settingsBefore), `"statusLine"`) {
		t.Fatalf("precondition: settings file must contain statusLine key:\n%s", settingsBefore)
	}

	if rc := cli.Execute([]string{"update", "--root", root, "--scope", "project", "--remove"}); rc != cliutil.ExitOK {
		t.Fatalf("update --remove: %d", rc)
	}

	if _, statErr := os.Stat(scriptPath); !os.IsNotExist(statErr) {
		t.Errorf("statusline script must be deleted after --remove, stat err=%v", statErr)
	}
	settingsAfter, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings file must still exist (only the key is stripped): %v", err)
	}
	if strings.Contains(string(settingsAfter), `"statusLine"`) {
		t.Errorf("statusLine key must be stripped after --remove:\n%s", settingsAfter)
	}
}

// TestRun_UpdateRemoveRefusesForeignScriptWithoutForce asserts a
// hand-authored (unmarked) statusline script at the target scope is
// left in place and `--remove` reports findings, not success, unless
// --force is also given.
//
// Serial (no t.Parallel): uses testutil.CaptureRun, which mutates
// os.Stdout/os.Stderr (process-level fds); see setup_test.go.
func TestRun_UpdateRemoveRefusesForeignScriptWithoutForce(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	scriptPath := filepath.Join(root, ".claude", "statusline.sh")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scriptPath, []byte("#!/usr/bin/env bash\necho hand-written\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"update", "--root", root, "--scope", "project", "--remove"})
	})
	if rc != cliutil.ExitFindings {
		t.Fatalf("rc = %d, want cliutil.ExitFindings; stderr: %s", rc, stderr)
	}
	if _, err := os.Stat(scriptPath); err != nil {
		t.Errorf("foreign script must be left on disk without --force: %v", err)
	}
	if !strings.Contains(stderr, "--force") {
		t.Errorf("refusal message should mention --force:\n%s", stderr)
	}

	// --force overrides the refusal.
	if rc := cli.Execute([]string{"update", "--root", root, "--scope", "project", "--remove", "--force"}); rc != cliutil.ExitOK {
		t.Fatalf("update --remove --force: %d", rc)
	}
	if _, err := os.Stat(scriptPath); !os.IsNotExist(err) {
		t.Errorf("foreign script must be deleted with --force, stat err=%v", err)
	}
}

// TestRun_UpdateRemoveAndStatuslineMutuallyExclusive asserts the CLI
// rejects `--statusline` combined with `--remove` with a usage error
// (exit code 2), reachable through the full cobra dispatch path.
//
// Serial (no t.Parallel): uses testutil.CaptureRun, which mutates
// os.Stdout/os.Stderr (process-level fds); see setup_test.go.
func TestRun_UpdateRemoveAndStatuslineMutuallyExclusive(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"update", "--root", root, "--statusline", "--remove"})
	})
	if rc != cliutil.ExitUsage {
		t.Fatalf("rc = %d, want cliutil.ExitUsage; stderr: %s", rc, stderr)
	}
	if !strings.Contains(stderr, "mutually exclusive") {
		t.Errorf("expected a mutually-exclusive usage message, got: %s", stderr)
	}
}

// TestRun_UpdateRemoveNothingToRemoveIsANoOp asserts `--remove` at a
// scope with no script and no settings key succeeds (ExitOK) rather
// than erroring.
//
// Serial (no t.Parallel): uses testutil.CaptureRun, which mutates
// os.Stdout/os.Stderr (process-level fds); see setup_test.go.
func TestRun_UpdateRemoveNothingToRemoveIsANoOp(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}

	rc, stdout, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"update", "--root", root, "--scope", "project", "--remove"})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("rc = %d, want cliutil.ExitOK; stdout: %s stderr: %s", rc, stdout, stderr)
	}
	if !strings.Contains(stdout, "nothing to remove") {
		t.Errorf("expected a 'nothing to remove' message, got stdout: %s", stdout)
	}
}
