package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/skills"
)

// TestM0156_AC5_NonTTYWithoutWireSettingsSkipsWrite asserts M-0156/AC-5:
// when --wire-settings is false and we are not on a TTY (the default
// under `go test`), RunStatuslineScaffold must NOT write to the
// settings file. It should print the activation snippet instead.
//
// This test sets up a fresh temp dir with the statusline script
// pre-scaffolded (so the scaffold step is a no-op) and verifies that
// the settings file does not exist after the call.
func TestM0156_AC5_NonTTYWithoutWireSettingsSkipsWrite(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	// Pre-scaffold the statusline so the scaffold step doesn't fail.
	_, err := skills.ScaffoldStatuslineWithHome(root, t.TempDir(), skills.StatuslineScopeProject)
	if err != nil {
		t.Fatal(err)
	}

	rc := cliutil.RunStatuslineScaffold(cliutil.StatuslineOpts{
		RootDir:      root,
		Scope:        "project",
		WireSettings: false,
		FormatJSON:   false,
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("AC-5: RunStatuslineScaffold returned %d, want %d", rc, cliutil.ExitOK)
	}

	// The settings file must NOT exist — non-TTY without --wire-settings
	// means no consent was given.
	settingsPath := filepath.Join(root, ".claude", "settings.local.json")
	if _, err := os.Stat(settingsPath); err == nil {
		t.Errorf("AC-5: settings file %s must not exist without consent (non-TTY, no --wire-settings)", settingsPath)
	}
}

// TestM0156_AC5_WireSettingsWritesWithoutPrompt asserts M-0156/AC-5:
// when --wire-settings is true, the write proceeds without any TTY
// prompt, even under `go test` (no TTY). The settings file must be
// created with the statusLine key.
func TestM0156_AC5_WireSettingsWritesWithoutPrompt(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	// Pre-scaffold the statusline.
	_, err := skills.ScaffoldStatuslineWithHome(root, t.TempDir(), skills.StatuslineScopeProject)
	if err != nil {
		t.Fatal(err)
	}

	rc := cliutil.RunStatuslineScaffold(cliutil.StatuslineOpts{
		RootDir:      root,
		Scope:        "project",
		WireSettings: true,
		FormatJSON:   false,
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("AC-5: RunStatuslineScaffold returned %d, want %d", rc, cliutil.ExitOK)
	}

	settingsPath := filepath.Join(root, ".claude", "settings.local.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("AC-5: settings file must exist when --wire-settings is true: %v", err)
	}
	if !strings.Contains(string(data), `"statusLine"`) {
		t.Errorf("AC-5: settings file must contain statusLine key\ncontent: %s", data)
	}
}

// TestM0156_AC5_FormatJSONWithoutWireSettingsSkipsWrite asserts
// M-0156/AC-5: when --format=json is active without --wire-settings,
// the function must skip the settings write (JSON mode is always
// non-interactive per ADR-0015).
func TestM0156_AC5_FormatJSONWithoutWireSettingsSkipsWrite(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	// Pre-scaffold the statusline.
	_, err := skills.ScaffoldStatuslineWithHome(root, t.TempDir(), skills.StatuslineScopeProject)
	if err != nil {
		t.Fatal(err)
	}

	rc := cliutil.RunStatuslineScaffold(cliutil.StatuslineOpts{
		RootDir:      root,
		Scope:        "project",
		WireSettings: false,
		FormatJSON:   true,
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("AC-5: RunStatuslineScaffold returned %d, want %d", rc, cliutil.ExitOK)
	}

	settingsPath := filepath.Join(root, ".claude", "settings.local.json")
	if _, err := os.Stat(settingsPath); err == nil {
		t.Errorf("AC-5: settings file must not exist in --format=json mode without --wire-settings")
	}
}
