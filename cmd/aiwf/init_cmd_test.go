package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRun_InitThroughDispatcher confirms `aiwf init` wires through the
// dispatcher: scaffolds dirs, writes aiwf.yaml, materializes skills.
//
// The dispatcher test passes --skip-hook because the in-process
// dispatcher bakes os.Executable() (= the test binary) into any
// hook it installs; firing the test binary as a hook is unsafe
// (see setupCLITestRepo's discipline note). End-to-end hook
// installation is covered by the runBin-style binary integration
// tests, which build a real aiwf and exercise consumer parity.
func TestRun_InitThroughDispatcher(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)

	rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"})
	if rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	if _, err := os.Stat(filepath.Join(root, "aiwf.yaml")); err != nil {
		t.Errorf("aiwf.yaml missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "skills", "aiwf-add", "SKILL.md")); err != nil {
		t.Errorf("aiwf-add skill missing: %v", err)
	}

	// Re-run to confirm idempotency through the dispatcher.
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Errorf("re-run init: %d", rc)
	}
}

// TestRun_InitDryRun confirms `aiwf init --dry-run` reports the
// would-be ledger, prefixes the output with a dry-run banner, and
// writes nothing to disk.
func TestRun_InitDryRun(t *testing.T) {
	root := setupCLITestRepo(t)

	captured := captureStdout(t, func() {
		if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook", "--dry-run"}); rc != exitOK {
			t.Errorf("got rc=%d, want %d", rc, exitOK)
		}
	})
	out := string(captured)

	for _, want := range []string{
		"dry-run",
		"created    aiwf.yaml",
		"created    work/epics",
		"updated    .claude/skills/aiwf-*",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}
	// Nothing on disk.
	for _, p := range []string{
		"aiwf.yaml",
		filepath.Join(".claude", "skills", "aiwf-add", "SKILL.md"),
		filepath.Join(".git", "hooks", "pre-push"),
	} {
		if _, err := os.Stat(filepath.Join(root, p)); !os.IsNotExist(err) {
			t.Errorf("dry-run wrote %s (stat err=%v); should be untouched", p, err)
		}
	}
}

// TestRun_InitSkipHook confirms `aiwf init --skip-hook` lands every
// step except the hook installation. Exit is OK (skip is requested,
// not a conflict).
func TestRun_InitSkipHook(t *testing.T) {
	root := setupCLITestRepo(t)

	captured := captureStdout(t, func() {
		if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
			t.Errorf("got rc=%d, want %d", rc, exitOK)
		}
	})
	out := string(captured)

	for _, want := range []string{
		"skipped    .git/hooks/pre-push",
		"--skip-hook",
		"pre-push hook skipped",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "aiwf.yaml")); err != nil {
		t.Errorf("aiwf.yaml missing after --skip-hook init: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-push")); !os.IsNotExist(err) {
		t.Errorf("hook installed despite --skip-hook (stat err=%v)", err)
	}
}

// TestRun_InitMigratesAlienHook (G45): when a non-aiwf pre-push hook
// is in place, init auto-migrates it to pre-push.local, installs
// aiwf's chain-aware hook, and exits exitOK. The migrated content
// is preserved byte-for-byte.
func TestRun_InitMigratesAlienHook(t *testing.T) {
	root := setupCLITestRepo(t)
	hookDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alien := []byte("#!/bin/sh\nexit 0\n")
	if err := os.WriteFile(filepath.Join(hookDir, "pre-push"), alien, 0o755); err != nil {
		t.Fatal(err)
	}

	captured := captureStdout(t, func() {
		// No --skip-hook here: this test exercises the G45 hook
		// migration path and needs init to actually install (and
		// migrate) the hook. The test does not trigger any commits,
		// so the test binary won't be invoked as a hook.
		if rc := run([]string{"init", "--root", root, "--actor", "human/test"}); rc != exitOK {
			t.Errorf("got %d, want %d (G45 auto-migrates, no conflict)", rc, exitOK)
		}
	})
	out := string(captured)

	for _, want := range []string{
		"created    aiwf.yaml",
		"created    work/epics",
		"updated    .claude/skills/aiwf-*",
		"migrated   .git/hooks/pre-push",
		"pre-push.local",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}

	// Migrated content lives at .local byte-for-byte.
	migrated, err := os.ReadFile(filepath.Join(hookDir, "pre-push.local"))
	if err != nil {
		t.Fatalf("reading pre-push.local: %v", err)
	}
	if !bytes.Equal(migrated, alien) {
		t.Errorf("migrated content drifted:\n got  %s\n want %s", migrated, alien)
	}
	// pre-push itself is now aiwf's chain-aware hook.
	installed, err := os.ReadFile(filepath.Join(hookDir, "pre-push"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(installed, []byte("# aiwf:pre-push")) {
		t.Errorf("post-migration pre-push lacks aiwf marker")
	}
	if !bytes.Contains(installed, []byte("pre-push.local")) {
		t.Errorf("post-migration pre-push lacks chain reference")
	}
}
