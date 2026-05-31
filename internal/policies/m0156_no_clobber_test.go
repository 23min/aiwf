package policies

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/skills"
)

// TestM0156_AC4_PreExistingKeyBlocksWrite asserts M-0156/AC-4:
// when a settings file already contains a statusLine key pointing at
// a different command, WireStatuslineSettings refuses the write and
// returns ExistingValue for merge guidance.
func TestM0156_AC4_PreExistingKeyBlocksWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	original := []byte(`{"statusLine":{"type":"command","command":"/other/script.sh"}}` + "\n")
	if err := os.WriteFile(settingsPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := skills.WireStatuslineSettings(settingsPath, ".claude/statusline.sh")
	if err != nil {
		t.Fatalf("WireStatuslineSettings: %v", err)
	}
	if res.Wrote {
		t.Error("AC-4: Wrote must be false when a pre-existing statusLine key exists")
	}
	if res.Idempotent {
		t.Error("AC-4: Idempotent must be false when the existing value differs")
	}
	if res.ExistingValue == "" {
		t.Error("AC-4: ExistingValue must be non-empty so the caller can print merge guidance")
	}

	// File must be unchanged — no .bak, no edit.
	after, _ := os.ReadFile(settingsPath)
	if !bytes.Equal(after, original) {
		t.Errorf("AC-4: settings file was modified despite no-clobber\nbefore: %s\nafter:  %s", original, after)
	}
	if res.BackupPath != "" {
		t.Errorf("AC-4: BackupPath should be empty on no-clobber, got %q", res.BackupPath)
	}
}

// TestM0156_AC4_IdempotentWhenKeyMatches asserts M-0156/AC-4:
// when the statusLine key already points at the same command path,
// the call is an idempotent no-op — no write, no .bak.
func TestM0156_AC4_IdempotentWhenKeyMatches(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	original := []byte(`{"statusLine":{"type":"command","command":".claude/statusline.sh"}}` + "\n")
	if err := os.WriteFile(settingsPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := skills.WireStatuslineSettings(settingsPath, ".claude/statusline.sh")
	if err != nil {
		t.Fatalf("WireStatuslineSettings: %v", err)
	}
	if res.Wrote {
		t.Error("AC-4: Wrote must be false on idempotent re-run")
	}
	if !res.Idempotent {
		t.Error("AC-4: Idempotent must be true when the existing value matches")
	}
	if res.BackupPath != "" {
		t.Errorf("AC-4: BackupPath should be empty on idempotent no-op, got %q", res.BackupPath)
	}
}
