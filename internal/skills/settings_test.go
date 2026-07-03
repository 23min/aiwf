package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestStatuslineSettingsKeyStatus_NoFile asserts a missing settings
// file reports existed=false, matches=false, no error, no existing
// value (G-0354).
func TestStatuslineSettingsKeyStatus_NoFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	existed, matches, existingValue, err := StatuslineSettingsKeyStatus(settingsPath, "/x/statusline.sh")
	if err != nil {
		t.Fatalf("StatuslineSettingsKeyStatus: %v", err)
	}
	if existed || matches || existingValue != "" {
		t.Errorf("missing file must report existed=false, matches=false, existingValue=\"\"; got existed=%v matches=%v existingValue=%q", existed, matches, existingValue)
	}
}

// TestStatuslineSettingsKeyStatus_NoKey asserts a settings file with
// other keys but no statusLine reports existed=false.
func TestStatuslineSettingsKeyStatus_NoKey(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"hooks": {}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	existed, matches, existingValue, err := StatuslineSettingsKeyStatus(settingsPath, "/x/statusline.sh")
	if err != nil {
		t.Fatalf("StatuslineSettingsKeyStatus: %v", err)
	}
	if existed || matches || existingValue != "" {
		t.Errorf("no statusLine key must report existed=false, matches=false, existingValue=\"\"; got existed=%v matches=%v existingValue=%q", existed, matches, existingValue)
	}
}

// TestStatuslineSettingsKeyStatus_Matching asserts a statusLine key
// whose command equals cmdPath reports existed=true, matches=true, no
// existingValue.
func TestStatuslineSettingsKeyStatus_Matching(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"statusLine": {"type": "command", "command": "/x/statusline.sh"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	existed, matches, existingValue, err := StatuslineSettingsKeyStatus(settingsPath, "/x/statusline.sh")
	if err != nil {
		t.Fatalf("StatuslineSettingsKeyStatus: %v", err)
	}
	if !existed || !matches || existingValue != "" {
		t.Errorf("matching key must report existed=true, matches=true, existingValue=\"\"; got existed=%v matches=%v existingValue=%q", existed, matches, existingValue)
	}
}

// TestStatuslineSettingsKeyStatus_Mismatch asserts a statusLine key
// whose command does NOT equal cmdPath reports existed=true,
// matches=false, and a non-empty existingValue for the caller's
// refusal message.
func TestStatuslineSettingsKeyStatus_Mismatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"statusLine": {"type": "command", "command": "/hand/written.sh"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	existed, matches, existingValue, err := StatuslineSettingsKeyStatus(settingsPath, "/x/statusline.sh")
	if err != nil {
		t.Fatalf("StatuslineSettingsKeyStatus: %v", err)
	}
	if !existed || matches || existingValue == "" {
		t.Errorf("mismatched key must report existed=true, matches=false, existingValue non-empty; got existed=%v matches=%v existingValue=%q", existed, matches, existingValue)
	}
}

// TestStatuslineSettingsKeyStatus_UnparsableValueTreatedAsMismatch
// asserts a statusLine value that isn't a {type,command} object
// (json.Unmarshal fails) is treated as a mismatch, not a match.
func TestStatuslineSettingsKeyStatus_UnparsableValueTreatedAsMismatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"statusLine": "not-an-object"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	existed, matches, existingValue, err := StatuslineSettingsKeyStatus(settingsPath, "/x/statusline.sh")
	if err != nil {
		t.Fatalf("StatuslineSettingsKeyStatus: %v", err)
	}
	if !existed || matches || existingValue == "" {
		t.Errorf("an unparsable statusLine value must report existed=true, matches=false, existingValue non-empty; got existed=%v matches=%v existingValue=%q", existed, matches, existingValue)
	}
}

// TestStatuslineSettingsKeyStatus_ReadError asserts a non-ENOENT read
// failure (settingsPath is a directory) surfaces as an error rather
// than being treated as "no key".
func TestStatuslineSettingsKeyStatus_ReadError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.Mkdir(settingsPath, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, _, _, err := StatuslineSettingsKeyStatus(settingsPath, "/x/statusline.sh"); err == nil {
		t.Error("expected an error when settingsPath is a directory")
	}
}

// TestStatuslineSettingsKeyStatus_ParseError asserts malformed JSON
// content surfaces as an error.
func TestStatuslineSettingsKeyStatus_ParseError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, _, err := StatuslineSettingsKeyStatus(settingsPath, "/x/statusline.sh"); err == nil {
		t.Error("expected an error for malformed settings JSON")
	}
}

// TestRemoveStatuslineSettingsKey_NoFile asserts a missing settings
// file is a no-op: removed=false, no error.
func TestRemoveStatuslineSettingsKey_NoFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	removed, err := RemoveStatuslineSettingsKey(settingsPath)
	if err != nil {
		t.Fatalf("RemoveStatuslineSettingsKey: %v", err)
	}
	if removed {
		t.Error("missing file must report removed=false")
	}
}

// TestRemoveStatuslineSettingsKey_NoKey asserts a settings file with
// other keys but no statusLine is a no-op, left untouched.
func TestRemoveStatuslineSettingsKey_NoKey(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	original := []byte(`{"hooks": {}}` + "\n")
	if err := os.WriteFile(settingsPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	removed, err := RemoveStatuslineSettingsKey(settingsPath)
	if err != nil {
		t.Fatalf("RemoveStatuslineSettingsKey: %v", err)
	}
	if removed {
		t.Error("no statusLine key must report removed=false")
	}
	after, _ := os.ReadFile(settingsPath)
	if !bytes.Equal(after, original) {
		t.Errorf("file must be left untouched when there is no statusLine key\nbefore: %s\nafter:  %s", original, after)
	}
}

// TestRemoveStatuslineSettingsKey_RemovesKeyAndPreservesOthers asserts
// the key is stripped unconditionally (the caller has already decided
// this is authorized) and unrelated keys are preserved.
func TestRemoveStatuslineSettingsKey_RemovesKeyAndPreservesOthers(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	original := []byte(`{"hooks": {}, "statusLine": {"type": "command", "command": "/x/statusline.sh"}}` + "\n")
	if err := os.WriteFile(settingsPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	removed, err := RemoveStatuslineSettingsKey(settingsPath)
	if err != nil {
		t.Fatalf("RemoveStatuslineSettingsKey: %v", err)
	}
	if !removed {
		t.Error("existing key must report removed=true")
	}

	obj, err := parseSettingsJSON(mustRead(t, settingsPath))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := obj["statusLine"]; ok {
		t.Error("statusLine key must be gone after removal")
	}
	if _, ok := obj["hooks"]; !ok {
		t.Error("unrelated 'hooks' key must be preserved")
	}
}

// TestRemoveStatuslineSettingsKey_ReadError asserts a non-ENOENT read
// failure (settingsPath is a directory) surfaces as an error.
func TestRemoveStatuslineSettingsKey_ReadError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.Mkdir(settingsPath, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := RemoveStatuslineSettingsKey(settingsPath); err == nil {
		t.Error("expected an error when settingsPath is a directory")
	}
}

// TestRemoveStatuslineSettingsKey_ParseError asserts malformed JSON
// content surfaces as an error rather than being treated as "no key".
func TestRemoveStatuslineSettingsKey_ParseError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := RemoveStatuslineSettingsKey(settingsPath); err == nil {
		t.Error("expected an error for malformed settings JSON")
	}
}

// mustRead reads path or fails the test.
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return b
}
