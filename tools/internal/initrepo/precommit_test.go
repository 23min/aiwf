package initrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnsurePreCommitHook_InstallFresh: install=true on a repo with
// no existing pre-commit hook → ActionCreated, hook lands with the
// marker and the embedded template body, no conflict.
func TestEnsurePreCommitHook_InstallFresh(t *testing.T) {
	root := freshGitRepo(t)
	step, conflict, err := ensurePreCommitHook(context.Background(), root, true, false)
	if err != nil {
		t.Fatalf("ensurePreCommitHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false (no prior hook)")
	}
	if step.Action != ActionCreated {
		t.Errorf("Action = %q, want %q", step.Action, ActionCreated)
	}
	body, err := os.ReadFile(filepath.Join(root, ".git", "hooks", "pre-commit"))
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	if !strings.Contains(string(body), PreCommitHookMarker()) {
		t.Errorf("hook body missing marker:\n%s", body)
	}
	if !strings.Contains(string(body), "status --root") {
		t.Errorf("hook body missing status invocation:\n%s", body)
	}
}

// TestEnsurePreCommitHook_RefreshOurOwn: install=true when our own
// marker-managed hook is already there → ActionUpdated, body
// rewritten from the embedded template.
func TestEnsurePreCommitHook_RefreshOurOwn(t *testing.T) {
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := []byte("#!/bin/sh\n" + PreCommitHookMarker() + "\n# stale body\nexit 1\n")
	if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), stale, 0o755); err != nil {
		t.Fatal(err)
	}

	step, conflict, err := ensurePreCommitHook(context.Background(), root, true, false)
	if err != nil {
		t.Fatalf("ensurePreCommitHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false (own hook)")
	}
	if step.Action != ActionUpdated {
		t.Errorf("Action = %q, want %q", step.Action, ActionUpdated)
	}
	got, err := os.ReadFile(filepath.Join(hooksDir, "pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "stale body") {
		t.Errorf("stale content survived refresh:\n%s", got)
	}
}

// TestEnsurePreCommitHook_SkipsAlien: install=true with a non-marker
// hook in place → ActionSkipped, conflict=true, alien hook left
// byte-for-byte alone.
func TestEnsurePreCommitHook_SkipsAlien(t *testing.T) {
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alien := []byte("#!/bin/sh\n# user's own hook\nexit 0\n")
	hookPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(hookPath, alien, 0o755); err != nil {
		t.Fatal(err)
	}

	step, conflict, err := ensurePreCommitHook(context.Background(), root, true, false)
	if err != nil {
		t.Fatalf("ensurePreCommitHook: %v", err)
	}
	if !conflict {
		t.Error("conflict = false, want true (alien hook should signal)")
	}
	if step.Action != ActionSkipped {
		t.Errorf("Action = %q, want %q", step.Action, ActionSkipped)
	}
	got, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytesEqual(got, alien) {
		t.Errorf("alien hook clobbered:\nwant %q\ngot  %q", alien, got)
	}
}

// TestEnsurePreCommitHook_UninstallOurOwn: install=false with a
// marker-managed hook in place → ActionRemoved, hook file gone.
func TestEnsurePreCommitHook_UninstallOurOwn(t *testing.T) {
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte("#!/bin/sh\n" + PreCommitHookMarker() + "\nexit 0\n")
	hookPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(hookPath, body, 0o755); err != nil {
		t.Fatal(err)
	}

	step, conflict, err := ensurePreCommitHook(context.Background(), root, false, false)
	if err != nil {
		t.Fatalf("ensurePreCommitHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false (own hook)")
	}
	if step.Action != ActionRemoved {
		t.Errorf("Action = %q, want %q", step.Action, ActionRemoved)
	}
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Errorf("hook still exists after uninstall (stat err=%v)", err)
	}
}

// TestEnsurePreCommitHook_UninstallNoHook: install=false with no
// hook present → ActionPreserved with a "disabled by config"
// detail; nothing on disk.
func TestEnsurePreCommitHook_UninstallNoHook(t *testing.T) {
	root := freshGitRepo(t)
	step, conflict, err := ensurePreCommitHook(context.Background(), root, false, false)
	if err != nil {
		t.Fatalf("ensurePreCommitHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false")
	}
	if step.Action != ActionPreserved {
		t.Errorf("Action = %q, want %q", step.Action, ActionPreserved)
	}
	if !strings.Contains(step.Detail, "disabled by config") {
		t.Errorf("Detail = %q, want it to mention the opt-out", step.Detail)
	}
}

// TestEnsurePreCommitHook_UninstallSkipsAlien: install=false with a
// non-marker hook in place → ActionSkipped, conflict=true, alien
// hook left alone. Critical: opt-out must never delete user
// content.
func TestEnsurePreCommitHook_UninstallSkipsAlien(t *testing.T) {
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alien := []byte("#!/bin/sh\n# user's own hook\nexit 0\n")
	hookPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(hookPath, alien, 0o755); err != nil {
		t.Fatal(err)
	}

	step, conflict, err := ensurePreCommitHook(context.Background(), root, false, false)
	if err != nil {
		t.Fatalf("ensurePreCommitHook: %v", err)
	}
	if !conflict {
		t.Error("conflict = false, want true (alien hook should signal)")
	}
	if step.Action != ActionSkipped {
		t.Errorf("Action = %q, want %q", step.Action, ActionSkipped)
	}
	got, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytesEqual(got, alien) {
		t.Errorf("alien hook clobbered on uninstall:\nwant %q\ngot  %q", alien, got)
	}
}

// TestEnsurePreCommitHook_DryRunInstall: dryRun=true must not write
// the hook even when install=true and no prior hook exists. The
// reported StepResult still says ActionCreated so a preview ledger
// reads as "this would be created".
func TestEnsurePreCommitHook_DryRunInstall(t *testing.T) {
	root := freshGitRepo(t)
	step, conflict, err := ensurePreCommitHook(context.Background(), root, true, true)
	if err != nil {
		t.Fatalf("ensurePreCommitHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false")
	}
	if step.Action != ActionCreated {
		t.Errorf("Action = %q, want %q", step.Action, ActionCreated)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit")); !os.IsNotExist(err) {
		t.Errorf("dry-run wrote the hook (stat err=%v)", err)
	}
}

// TestEnsurePreCommitHook_DryRunUninstall: dryRun=true must not
// remove a marker-managed hook even when install=false. The
// reported StepResult still says ActionRemoved.
func TestEnsurePreCommitHook_DryRunUninstall(t *testing.T) {
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte("#!/bin/sh\n" + PreCommitHookMarker() + "\nexit 0\n")
	hookPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(hookPath, body, 0o755); err != nil {
		t.Fatal(err)
	}

	step, _, err := ensurePreCommitHook(context.Background(), root, false, true)
	if err != nil {
		t.Fatalf("ensurePreCommitHook: %v", err)
	}
	if step.Action != ActionRemoved {
		t.Errorf("Action = %q, want %q", step.Action, ActionRemoved)
	}
	if _, err := os.Stat(hookPath); err != nil {
		t.Errorf("dry-run removed the hook: %v", err)
	}
}

// TestInit_InstallsPreCommitByDefault: a fresh `aiwf init` against a
// new repo lands the pre-commit hook with the marker, and the
// ledger reports it Created. Default-on is the framework's contract
// for STATUS.md auto-update.
func TestInit_InstallsPreCommitByDefault(t *testing.T) {
	root := freshGitRepo(t)
	res, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	step := findStep(t, res.Steps, ".git/hooks/pre-commit")
	if step.Action != ActionCreated {
		t.Errorf("pre-commit step.Action = %q, want %q", step.Action, ActionCreated)
	}
	body, err := os.ReadFile(filepath.Join(root, ".git", "hooks", "pre-commit"))
	if err != nil {
		t.Fatalf("read pre-commit hook: %v", err)
	}
	if !strings.Contains(string(body), PreCommitHookMarker()) {
		t.Errorf("pre-commit hook missing marker:\n%s", body)
	}
}

// TestInit_RespectsStatusMdAutoUpdateFalse: a repo whose pre-existing
// aiwf.yaml opts out of STATUS.md auto-update lands no pre-commit
// hook, even on a fresh init. The ledger row reports it Preserved
// with a "disabled by config" detail so the user understands why
// the step did nothing.
func TestInit_RespectsStatusMdAutoUpdateFalse(t *testing.T) {
	root := freshGitRepo(t)
	yaml := []byte(`aiwf_version: 0.1.0
actor: human/peter
status_md:
  auto_update: false
`)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	step := findStep(t, res.Steps, ".git/hooks/pre-commit")
	if step.Action != ActionPreserved {
		t.Errorf("pre-commit step.Action = %q, want %q (opt-out, no prior hook)", step.Action, ActionPreserved)
	}
	if !strings.Contains(step.Detail, "disabled by config") {
		t.Errorf("pre-commit step.Detail = %q, want a 'disabled by config' note", step.Detail)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit")); !os.IsNotExist(err) {
		t.Errorf("pre-commit hook installed despite opt-out (stat err=%v)", err)
	}
}

// TestRefreshArtifacts_FlipFlagUninstalls: simulate the canonical
// opt-out flow — install on default, then flip the flag and re-run
// the refresh. The hook is removed.
func TestRefreshArtifacts_FlipFlagUninstalls(t *testing.T) {
	root := freshGitRepo(t)
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	hookPath := filepath.Join(root, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); err != nil {
		t.Fatalf("pre-commit hook not installed by default Init: %v", err)
	}

	// Flip the flag in the typical hand-edit shape.
	yaml := []byte(`aiwf_version: 0.1.0
actor: human/peter
status_md:
  auto_update: false
`)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}

	steps, conflict, err := RefreshArtifacts(context.Background(), root, RefreshOptions{
		StatusMdAutoUpdate: false,
	})
	if err != nil {
		t.Fatalf("RefreshArtifacts: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true on opt-out, want false")
	}
	step := findStep(t, steps, ".git/hooks/pre-commit")
	if step.Action != ActionRemoved {
		t.Errorf("pre-commit step.Action = %q, want %q", step.Action, ActionRemoved)
	}
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Errorf("pre-commit hook still on disk after opt-out (stat err=%v)", err)
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
