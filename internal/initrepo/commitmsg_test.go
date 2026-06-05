package initrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnsureCommitMsgHook_InstallFresh: fresh install lands the
// commit-msg hook with the marker AND the unique commit-msg exec
// line. The marker / migrate / dry-run paths are covered uniformly
// across all four hooks by precommit_test.go; only the commit-msg-
// specific exec line needs pinning here.
func TestEnsureCommitMsgHook_InstallFresh(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	step, conflict, err := ensureCommitMsgHook(context.Background(), root, false)
	if err != nil {
		t.Fatalf("ensureCommitMsgHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false (no prior hook)")
	}
	if step.Action != ActionCreated {
		t.Errorf("Action = %q, want %q", step.Action, ActionCreated)
	}
	body, err := os.ReadFile(filepath.Join(root, ".git", "hooks", "commit-msg"))
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	if !strings.Contains(string(body), CommitMsgHookMarker()) {
		t.Errorf("hook body missing marker:\n%s", body)
	}
	if !strings.Contains(string(body), `check --commit-msg "$1"`) {
		t.Errorf("hook body missing commit-msg validation exec:\n%s", body)
	}
}

// TestEnsureCommitMsgHook_MigratesAlien: a non-marker commit-msg
// hook → auto-migrates to commit-msg.local (G45). Pins the
// commit-msg-specific migration path so a future regression in
// ensureCommitMsgHook's migration branch doesn't silently swallow
// a consumer's own commit-msg hook (e.g. a Conventional Commits
// enforcer).
func TestEnsureCommitMsgHook_MigratesAlien(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alien := []byte("#!/bin/sh\n# user's own commit-msg hook\nexit 0\n")
	hookPath := filepath.Join(hooksDir, "commit-msg")
	if err := os.WriteFile(hookPath, alien, 0o755); err != nil {
		t.Fatal(err)
	}

	step, conflict, err := ensureCommitMsgHook(context.Background(), root, false)
	if err != nil {
		t.Fatalf("ensureCommitMsgHook: %v", err)
	}
	if conflict {
		t.Error("conflict = true, want false (G45 auto-migrates)")
	}
	if step.Action != ActionMigrated {
		t.Errorf("Action = %q, want %q", step.Action, ActionMigrated)
	}
	migrated, err := os.ReadFile(filepath.Join(hooksDir, "commit-msg.local"))
	if err != nil {
		t.Fatalf("reading commit-msg.local: %v", err)
	}
	if !bytesEqual(migrated, alien) {
		t.Errorf("migrated content drifted:\nwant %q\ngot  %q", alien, migrated)
	}
	installed, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(installed), CommitMsgHookMarker()) {
		t.Errorf("post-migration commit-msg lacks aiwf marker")
	}
	if !strings.Contains(string(installed), "commit-msg.local") {
		t.Errorf("post-migration commit-msg lacks chain prelude:\n%s", installed)
	}
}

// TestEnsureCommitMsgHook_RefreshOurOwn: marker present → refresh
// in place (ActionUpdated), stale body rewritten from the template.
// `ensureCommitMsgHook` is independently coded (no shared
// ensureMarkerHook helper exists yet), so each hook's update path
// needs its own pin.
func TestEnsureCommitMsgHook_RefreshOurOwn(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := []byte("#!/bin/sh\n" + CommitMsgHookMarker() + "\n# stale body\nexit 1\n")
	if err := os.WriteFile(filepath.Join(hooksDir, "commit-msg"), stale, 0o755); err != nil {
		t.Fatal(err)
	}
	step, conflict, err := ensureCommitMsgHook(context.Background(), root, false)
	if err != nil {
		t.Fatalf("ensureCommitMsgHook: %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false (our own hook)")
	}
	if step.Action != ActionUpdated {
		t.Errorf("Action = %q, want %q", step.Action, ActionUpdated)
	}
	got, err := os.ReadFile(filepath.Join(hooksDir, "commit-msg"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "stale body") {
		t.Errorf("stale content survived refresh:\n%s", got)
	}
}

// TestEnsureCommitMsgHook_MigrationCollision: alien hook in place
// AND commit-msg.local already exists → Skipped + conflict=true.
// Catches a regression in the collision branch unique to this hook.
func TestEnsureCommitMsgHook_MigrationCollision(t *testing.T) {
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
	step, conflict, err := ensureCommitMsgHook(context.Background(), root, false)
	if err != nil {
		t.Fatalf("ensureCommitMsgHook: %v", err)
	}
	if !conflict {
		t.Error("conflict = false, want true (alien + .local collision)")
	}
	if step.Action != ActionSkipped {
		t.Errorf("Action = %q, want %q", step.Action, ActionSkipped)
	}
	got, err := os.ReadFile(filepath.Join(hooksDir, "commit-msg"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytesEqual(got, alien) {
		t.Errorf("alien commit-msg was modified on conflict:\nwant %q\ngot  %q", alien, got)
	}
}

// TestEnsureCommitMsgHook_DryRunReports: dry-run reports the
// prospective action but writes nothing.
func TestEnsureCommitMsgHook_DryRunReports(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	step, conflict, err := ensureCommitMsgHook(context.Background(), root, true)
	if err != nil {
		t.Fatalf("ensureCommitMsgHook(dry): %v", err)
	}
	if conflict {
		t.Errorf("conflict = true, want false")
	}
	if step.Action != ActionCreated {
		t.Errorf("Action = %q, want %q (dry-run still reports the prospective action)", step.Action, ActionCreated)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "commit-msg")); !os.IsNotExist(err) {
		t.Errorf("dry-run wrote .git/hooks/commit-msg (err=%v); should be a no-op", err)
	}
}

// TestApply_InstallsCommitMsgHook: Apply() wires the commit-msg
// hook alongside pre-push/pre-commit/post-commit. The seam test —
// catches the case where ensureCommitMsgHook exists but isn't wired
// into the install pipeline.
func TestApply_InstallsCommitMsgHook(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	if _, err := Init(context.Background(), root, Options{}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(root, ".git", "hooks", "commit-msg"))
	if err != nil {
		t.Fatalf("commit-msg hook not installed by Init: %v", err)
	}
	if !strings.Contains(string(body), CommitMsgHookMarker()) {
		t.Errorf("Init installed commit-msg without marker:\n%s", body)
	}
}
