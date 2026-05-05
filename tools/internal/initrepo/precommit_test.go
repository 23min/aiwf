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

// TestEnsurePreCommitHook_RegenOff_FreshInstall (G42): regenStatus=false
// on a repo with no prior hook still installs the hook (the
// tree-discipline gate is enforcement, not opt-out-able). The script
// body omits the STATUS.md regen block.
func TestEnsurePreCommitHook_RegenOff_FreshInstall(t *testing.T) {
	root := freshGitRepo(t)
	step, conflict, err := ensurePreCommitHook(context.Background(), root, false, false)
	if err != nil {
		t.Fatalf("ensurePreCommitHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false")
	}
	if step.Action != ActionCreated {
		t.Errorf("Action = %q, want %q (G42: hook always installs)", step.Action, ActionCreated)
	}
	body, err := os.ReadFile(filepath.Join(root, ".git", "hooks", "pre-commit"))
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	if !strings.Contains(string(body), "check --shape-only") {
		t.Errorf("regenStatus=false hook still must include the tree-discipline gate:\n%s", body)
	}
	if strings.Contains(string(body), "status --root") {
		t.Errorf("regenStatus=false must omit STATUS.md regen step:\n%s", body)
	}
}

// TestEnsurePreCommitHook_RegenOff_RefreshDropsRegen (G42): when our
// own hook is in place and the consumer flips status_md.auto_update
// to false, a refresh rewrites the script in place to drop the regen
// step. The gate stays. Action=Updated, conflict=false.
func TestEnsurePreCommitHook_RegenOff_RefreshDropsRegen(t *testing.T) {
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(hooksDir, "pre-commit")
	prior := []byte("#!/bin/sh\n" + PreCommitHookMarker() + "\n# stale body with status --root invocation\nexit 0\n")
	if err := os.WriteFile(hookPath, prior, 0o755); err != nil {
		t.Fatal(err)
	}

	step, conflict, err := ensurePreCommitHook(context.Background(), root, false, false)
	if err != nil {
		t.Fatalf("ensurePreCommitHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false (own hook)")
	}
	if step.Action != ActionUpdated {
		t.Errorf("Action = %q, want %q", step.Action, ActionUpdated)
	}
	got, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "check --shape-only") {
		t.Errorf("refreshed hook missing tree-discipline gate:\n%s", got)
	}
	if strings.Contains(string(got), "status --root") {
		t.Errorf("regenStatus=false refresh must drop the regen step:\n%s", got)
	}
}

// TestEnsurePreCommitHook_RegenOff_AlienHookPreserved (G42): regenStatus=false
// with a non-marker hook in place — the alien hook is left alone,
// same conflict-skip contract as the install path. The G42 change
// (always-install) does not weaken the alien-preservation guarantee.
func TestEnsurePreCommitHook_RegenOff_AlienHookPreserved(t *testing.T) {
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
		t.Errorf("alien hook clobbered:\nwant %q\ngot  %q", alien, got)
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

// TestEnsurePreCommitHook_DryRunRegenOff (G42): dryRun=true with
// regenStatus=false must not write the hook even though it would
// otherwise refresh in place. The StepResult reports ActionUpdated
// since a hook was already installed.
func TestEnsurePreCommitHook_DryRunRegenOff(t *testing.T) {
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	prior := []byte("#!/bin/sh\n" + PreCommitHookMarker() + "\n# untouched\nexit 0\n")
	hookPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(hookPath, prior, 0o755); err != nil {
		t.Fatal(err)
	}

	step, _, err := ensurePreCommitHook(context.Background(), root, false, true)
	if err != nil {
		t.Fatalf("ensurePreCommitHook: %v", err)
	}
	if step.Action != ActionUpdated {
		t.Errorf("Action = %q, want %q", step.Action, ActionUpdated)
	}
	got, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytesEqual(got, prior) {
		t.Errorf("dry-run rewrote the hook:\nwant %q\ngot  %q", prior, got)
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

// TestInit_StatusMdAutoUpdateFalse_StillInstallsGate (G42): a repo
// whose aiwf.yaml opts out of STATUS.md auto-update on fresh init
// still gets the pre-commit hook installed — the tree-discipline
// gate is enforcement and decoupled from the regen convenience.
// The ledger row reports it Created; the script body lacks the
// regen step.
func TestInit_StatusMdAutoUpdateFalse_StillInstallsGate(t *testing.T) {
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
	if step.Action != ActionCreated {
		t.Errorf("pre-commit step.Action = %q, want %q (G42: gate always installs)", step.Action, ActionCreated)
	}
	body, err := os.ReadFile(filepath.Join(root, ".git", "hooks", "pre-commit"))
	if err != nil {
		t.Fatalf("pre-commit hook not installed despite G42 contract: %v", err)
	}
	if !strings.Contains(string(body), "check --shape-only") {
		t.Errorf("hook missing tree-discipline gate:\n%s", body)
	}
	if strings.Contains(string(body), "status --root") {
		t.Errorf("status_md.auto_update: false must drop the regen step:\n%s", body)
	}
}

// TestRefreshArtifacts_FlipFlagDropsRegenKeepsGate (G42): canonical
// opt-out flow — install on default, then flip status_md.auto_update
// and re-refresh. The hook stays installed; only the regen block is
// dropped from the script body. Action=Updated.
func TestRefreshArtifacts_FlipFlagDropsRegenKeepsGate(t *testing.T) {
	root := freshGitRepo(t)
	if _, err := Init(context.Background(), root, Options{AiwfVersion: "0.1.0"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	hookPath := filepath.Join(root, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); err != nil {
		t.Fatalf("pre-commit hook not installed by default Init: %v", err)
	}

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
	if step.Action != ActionUpdated {
		t.Errorf("pre-commit step.Action = %q, want %q (G42: refresh in place)", step.Action, ActionUpdated)
	}
	body, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("pre-commit hook missing after refresh (G42 violation): %v", err)
	}
	if !strings.Contains(string(body), "check --shape-only") {
		t.Errorf("refreshed hook missing tree-discipline gate:\n%s", body)
	}
	if strings.Contains(string(body), "status --root") {
		t.Errorf("flip-flag refresh must drop status regen step:\n%s", body)
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
