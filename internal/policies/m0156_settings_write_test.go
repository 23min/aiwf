package policies

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/skills"
)

// TestM0156_AC3_ConsentGatedWriteCreatesBakAndInsertsKey asserts
// M-0156/AC-3: when the settings file exists with other keys but no
// statusLine, WireStatuslineSettings writes a .bak of the original
// and inserts the statusLine key.
func TestM0156_AC3_ConsentGatedWriteCreatesBakAndInsertsKey(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, ".claude", "settings.local.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"hooks": {}}` + "\n")
	if err := os.WriteFile(settingsPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := skills.WireStatuslineSettings(settingsPath, ".claude/statusline.sh")
	if err != nil {
		t.Fatalf("WireStatuslineSettings: %v", err)
	}
	if !res.Wrote {
		t.Error("AC-3: Wrote must be true when statusLine was inserted")
	}

	// .bak must exist and match original content.
	bakData, err := os.ReadFile(res.BackupPath)
	if err != nil {
		t.Fatalf("AC-3: .bak file must exist at %s: %v", res.BackupPath, err)
	}
	if !bytes.Equal(bakData, original) {
		t.Errorf("AC-3: .bak content mismatch\ngot:  %q\nwant: %q", bakData, original)
	}

	// Settings file must now contain statusLine key.
	written, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(written, &obj); err != nil {
		t.Fatalf("AC-3: written settings not valid JSON: %v", err)
	}
	raw, ok := obj["statusLine"]
	if !ok {
		t.Fatal("AC-3: written settings must contain statusLine key")
	}
	var sl struct {
		Type    string `json:"type"`
		Command string `json:"command"`
	}
	if err := json.Unmarshal(raw, &sl); err != nil {
		t.Fatalf("AC-3: statusLine value not valid: %v", err)
	}
	if sl.Type != "command" {
		t.Errorf("AC-3: statusLine.type = %q, want %q", sl.Type, "command")
	}
	if sl.Command != ".claude/statusline.sh" {
		t.Errorf("AC-3: statusLine.command = %q, want %q", sl.Command, ".claude/statusline.sh")
	}

	// Original keys must be preserved.
	if _, ok := obj["hooks"]; !ok {
		t.Error("AC-3: original 'hooks' key must be preserved after insert")
	}
}

// TestM0156_AC3_CreatesNewFileWhenAbsent asserts that
// WireStatuslineSettings creates the settings file from scratch when
// it does not exist (no .bak needed since there was no original).
func TestM0156_AC3_CreatesNewFileWhenAbsent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, ".claude", "settings.local.json")

	res, err := skills.WireStatuslineSettings(settingsPath, ".claude/statusline.sh")
	if err != nil {
		t.Fatalf("WireStatuslineSettings: %v", err)
	}
	if !res.Wrote {
		t.Error("AC-3: Wrote must be true when file was created")
	}
	if res.BackupPath != "" {
		t.Errorf("AC-3: BackupPath should be empty when file did not exist, got %q", res.BackupPath)
	}

	written, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(written, &obj); err != nil {
		t.Fatalf("AC-3: written settings not valid JSON: %v", err)
	}
	if _, ok := obj["statusLine"]; !ok {
		t.Fatal("AC-3: written settings must contain statusLine key")
	}
}

// TestM0156_AC3_SettingsPathForScope asserts SettingsPathForScope
// resolves to the correct file per scope.
func TestM0156_AC3_SettingsPathForScope(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()
	cases := []struct {
		name  string
		scope skills.StatuslineScope
		want  string
	}{
		{
			"project",
			skills.StatuslineScopeProject,
			filepath.Join(root, ".claude", "settings.local.json"),
		},
		{
			"user",
			skills.StatuslineScopeUser,
			filepath.Join(home, ".claude", "settings.json"),
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := skills.SettingsPathForScope(root, home, tc.scope)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("SettingsPathForScope(%q) = %q, want %q", tc.scope, got, tc.want)
			}
		})
	}
}
