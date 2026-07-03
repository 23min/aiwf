package cliutil

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/skills"
)

// TestRunStatuslineRemove_NothingToRemove asserts an empty project
// scope (no script, no settings key) is a no-op that reports exit 0
// (G-0354).
func TestRunStatuslineRemove_NothingToRemove(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	rc := RunStatuslineRemove(StatuslineRemoveOpts{RootDir: root, Scope: "project"})
	if rc != ExitOK {
		t.Fatalf("rc = %d, want ExitOK", rc)
	}
}

// TestRunStatuslineRemove_AiwfAuthoredRemovesBoth asserts the happy
// path: a scaffolded script + a matching wired settings key are both
// removed without needing --force.
func TestRunStatuslineRemove_AiwfAuthoredRemovesBoth(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()

	scaffold, err := skills.ScaffoldStatuslineWithHome(root, home, skills.StatuslineScopeProject)
	if err != nil {
		t.Fatal(err)
	}
	settingsPath, err := skills.SettingsPathForScope(root, home, skills.StatuslineScopeProject)
	if err != nil {
		t.Fatal(err)
	}
	if _, wireErr := skills.WireStatuslineSettings(settingsPath, scaffold.Command); wireErr != nil {
		t.Fatal(wireErr)
	}

	rc := RunStatuslineRemove(StatuslineRemoveOpts{RootDir: root, Scope: "project"})
	if rc != ExitOK {
		t.Fatalf("rc = %d, want ExitOK", rc)
	}
	if _, statErr := os.Stat(scaffold.Path); !os.IsNotExist(statErr) {
		t.Errorf("script must be deleted, stat err=%v", statErr)
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings file must still exist: %v", err)
	}
	if strings.Contains(string(data), `"statusLine"`) {
		t.Errorf("statusLine key must be stripped from settings:\n%s", data)
	}
}

// TestRunStatuslineRemove_ForeignScriptRefusedWithoutForce asserts a
// hand-authored script (no aiwf marker) is left alone and the call
// reports ExitFindings when --force is not given.
func TestRunStatuslineRemove_ForeignScriptRefusedWithoutForce(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	dest := filepath.Join(root, ".claude", "statusline.sh")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("#!/usr/bin/env bash\necho hand-written\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	rc := RunStatuslineRemove(StatuslineRemoveOpts{RootDir: root, Scope: "project"})
	if rc != ExitFindings {
		t.Fatalf("rc = %d, want ExitFindings", rc)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("foreign script must be left on disk without --force: %v", err)
	}
}

// TestRunStatuslineRemove_ForceRemovesForeignScript asserts --force
// deletes a foreign script that would otherwise be refused.
func TestRunStatuslineRemove_ForceRemovesForeignScript(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	dest := filepath.Join(root, ".claude", "statusline.sh")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("#!/usr/bin/env bash\necho hand-written\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	rc := RunStatuslineRemove(StatuslineRemoveOpts{RootDir: root, Scope: "project", Force: true})
	if rc != ExitOK {
		t.Fatalf("rc = %d, want ExitOK", rc)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("script must be deleted under --force, stat err=%v", err)
	}
}

// TestRunStatuslineRemove_ForeignSettingsKeyRefusedWithoutForce asserts
// a statusLine key whose command doesn't match aiwf's own wiring is
// left alone (ExitFindings) even when there's no script on disk.
func TestRunStatuslineRemove_ForeignSettingsKeyRefusedWithoutForce(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()

	settingsPath, err := skills.SettingsPathForScope(root, home, skills.StatuslineScopeProject)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"statusLine": {"type": "command", "command": "/hand/written.sh"}}` + "\n")
	if err := os.WriteFile(settingsPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	rc := RunStatuslineRemove(StatuslineRemoveOpts{RootDir: root, Scope: "project"})
	if rc != ExitFindings {
		t.Fatalf("rc = %d, want ExitFindings", rc)
	}
	after, _ := os.ReadFile(settingsPath)
	if !bytes.Equal(after, original) {
		t.Errorf("settings file must be left untouched on refusal\nbefore: %s\nafter:  %s", original, after)
	}
}

// TestRunStatuslineRemove_MixedAiwfScriptForeignSettingsRefusesBoth is
// the G-0354 review-blocking regression test: an aiwf-authored script
// alongside a FOREIGN (mismatched) settings key must refuse the WHOLE
// call and mutate NEITHER artifact — the aiwf-owned script must not be
// silently deleted just because the other artifact was refused.
func TestRunStatuslineRemove_MixedAiwfScriptForeignSettingsRefusesBoth(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()

	scaffold, err := skills.ScaffoldStatuslineWithHome(root, home, skills.StatuslineScopeProject)
	if err != nil {
		t.Fatal(err)
	}
	settingsPath, err := skills.SettingsPathForScope(root, home, skills.StatuslineScopeProject)
	if err != nil {
		t.Fatal(err)
	}
	if mkErr := os.MkdirAll(filepath.Dir(settingsPath), 0o755); mkErr != nil {
		t.Fatal(mkErr)
	}
	originalSettings := []byte(`{"statusLine": {"type": "command", "command": "/hand/written.sh"}}` + "\n")
	if writeErr := os.WriteFile(settingsPath, originalSettings, 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}
	originalScript, err := os.ReadFile(scaffold.Path)
	if err != nil {
		t.Fatal(err)
	}

	rc := RunStatuslineRemove(StatuslineRemoveOpts{RootDir: root, Scope: "project"})
	if rc != ExitFindings {
		t.Fatalf("rc = %d, want ExitFindings", rc)
	}

	// The aiwf-owned script must NOT have been deleted just because the
	// settings key was foreign.
	afterScript, err := os.ReadFile(scaffold.Path)
	if err != nil {
		t.Fatalf("aiwf-authored script must still be present after refusal: %v", err)
	}
	if !bytes.Equal(afterScript, originalScript) {
		t.Errorf("aiwf-authored script content must be unchanged after refusal\nbefore: %s\nafter:  %s", originalScript, afterScript)
	}
	afterSettings, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings file must still exist after refusal: %v", err)
	}
	if !bytes.Equal(afterSettings, originalSettings) {
		t.Errorf("foreign settings key must be unchanged after refusal\nbefore: %s\nafter:  %s", originalSettings, afterSettings)
	}
}

// TestRunStatuslineRemove_MixedForeignScriptAiwfSettingsRefusesBoth is
// the mirror-image G-0354 regression test: a FOREIGN script alongside
// an aiwf-authored (matching) settings key must also refuse the whole
// call and mutate NEITHER artifact.
func TestRunStatuslineRemove_MixedForeignScriptAiwfSettingsRefusesBoth(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()

	dest := filepath.Join(root, ".claude", "statusline.sh")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	originalScript := []byte("#!/usr/bin/env bash\necho hand-written\n")
	if err := os.WriteFile(dest, originalScript, 0o755); err != nil {
		t.Fatal(err)
	}

	cmdPath := skills.ProjectStatuslineCommand(root)
	settingsPath, err := skills.SettingsPathForScope(root, home, skills.StatuslineScopeProject)
	if err != nil {
		t.Fatal(err)
	}
	if _, wireErr := skills.WireStatuslineSettings(settingsPath, cmdPath); wireErr != nil {
		t.Fatal(wireErr)
	}
	originalSettings, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	rc := RunStatuslineRemove(StatuslineRemoveOpts{RootDir: root, Scope: "project"})
	if rc != ExitFindings {
		t.Fatalf("rc = %d, want ExitFindings", rc)
	}

	// The aiwf-owned settings key must NOT have been stripped just
	// because the script was foreign.
	afterScript, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("foreign script must still be present after refusal: %v", err)
	}
	if !bytes.Equal(afterScript, originalScript) {
		t.Errorf("foreign script content must be unchanged after refusal\nbefore: %s\nafter:  %s", originalScript, afterScript)
	}
	afterSettings, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("aiwf-authored settings key must still be present after refusal: %v", err)
	}
	if !bytes.Equal(afterSettings, originalSettings) {
		t.Errorf("aiwf-authored settings key must be unchanged after refusal\nbefore: %s\nafter:  %s", originalSettings, afterSettings)
	}
}

// TestRunStatuslineRemove_OnlySettingsKeyPresent covers the
// scriptExisted=false / settingsExisted=true asymmetry: no script on
// disk, only a matching settings key. Exercises the "skip script
// removal, still strip the key" branch pairing.
func TestRunStatuslineRemove_OnlySettingsKeyPresent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()

	cmdPath := skills.ProjectStatuslineCommand(root)
	settingsPath, err := skills.SettingsPathForScope(root, home, skills.StatuslineScopeProject)
	if err != nil {
		t.Fatal(err)
	}
	if _, wireErr := skills.WireStatuslineSettings(settingsPath, cmdPath); wireErr != nil {
		t.Fatal(wireErr)
	}

	rc := RunStatuslineRemove(StatuslineRemoveOpts{RootDir: root, Scope: "project"})
	if rc != ExitOK {
		t.Fatalf("rc = %d, want ExitOK", rc)
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings file must still exist: %v", err)
	}
	if strings.Contains(string(data), `"statusLine"`) {
		t.Errorf("statusLine key must be stripped from settings:\n%s", data)
	}
}

// TestRunStatuslineRemove_OnlyScriptPresent covers the
// scriptExisted=true / settingsExisted=false asymmetry: a matching
// script on disk, no settings key at all.
func TestRunStatuslineRemove_OnlyScriptPresent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()

	scaffold, err := skills.ScaffoldStatuslineWithHome(root, home, skills.StatuslineScopeProject)
	if err != nil {
		t.Fatal(err)
	}

	rc := RunStatuslineRemove(StatuslineRemoveOpts{RootDir: root, Scope: "project"})
	if rc != ExitOK {
		t.Fatalf("rc = %d, want ExitOK", rc)
	}
	if _, statErr := os.Stat(scaffold.Path); !os.IsNotExist(statErr) {
		t.Errorf("script must be deleted, stat err=%v", statErr)
	}
}

// TestRunStatuslineRemove_UnknownScope asserts an invalid --scope value
// surfaces as a usage error.
func TestRunStatuslineRemove_UnknownScope(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	rc := RunStatuslineRemove(StatuslineRemoveOpts{RootDir: root, Scope: "bogus"})
	if rc != ExitUsage {
		t.Fatalf("rc = %d, want ExitUsage", rc)
	}
}

// TestRunStatuslineRemove_ScriptReadErrorIsInternal asserts a
// non-ENOENT failure reading the script (dest is a directory) is
// reported as ExitInternal, not silently treated as "nothing to
// remove".
func TestRunStatuslineRemove_ScriptReadErrorIsInternal(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dest := filepath.Join(root, ".claude", "statusline.sh")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	rc := RunStatuslineRemove(StatuslineRemoveOpts{RootDir: root, Scope: "project"})
	if rc != ExitInternal {
		t.Fatalf("rc = %d, want ExitInternal", rc)
	}
}

// TestRunStatuslineRemove_SettingsReadErrorIsInternal asserts a
// non-ENOENT failure reading the settings file (settingsPath is a
// directory) is reported as ExitInternal.
func TestRunStatuslineRemove_SettingsReadErrorIsInternal(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()
	settingsPath, err := skills.SettingsPathForScope(root, home, skills.StatuslineScopeProject)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(settingsPath, 0o755); err != nil {
		t.Fatal(err)
	}

	rc := RunStatuslineRemove(StatuslineRemoveOpts{RootDir: root, Scope: "project"})
	if rc != ExitInternal {
		t.Fatalf("rc = %d, want ExitInternal", rc)
	}
}
