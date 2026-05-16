package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/initrepo"
)

// TestRun_UpdateMaterializes wipes a tampered skill file and verifies
// `aiwf update` restores the embedded content byte-for-byte.
func TestRun_UpdateMaterializes(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	skillPath := filepath.Join(root, ".claude", "skills", "aiwf-add", "SKILL.md")
	if err := os.WriteFile(skillPath, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rc := run([]string{"update", "--root", root}); rc != cliutil.ExitOK {
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
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	hookPath := filepath.Join(root, ".git", "hooks", "pre-push")
	if err := os.Remove(hookPath); err != nil {
		t.Fatalf("removing pre-push hook: %v", err)
	}
	if rc := run([]string{"update", "--root", root}); rc != cliutil.ExitOK {
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
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	hookPath := filepath.Join(root, ".git", "hooks", "pre-commit")
	if err := os.Remove(hookPath); err != nil {
		t.Fatalf("removing pre-commit hook: %v", err)
	}
	if rc := run([]string{"update", "--root", root}); rc != cliutil.ExitOK {
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
	if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
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

	if rc := run([]string{"update", "--root", root}); rc != cliutil.ExitOK {
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
	if rc := run([]string{"update", "--root", root}); rc != cliutil.ExitInternal {
		t.Errorf("rc = %d, want cliutil.ExitInternal (%d)", rc, cliutil.ExitInternal)
	}
}
