package skills

import (
	"os"
	"path/filepath"
	"testing"
)

// TestStatuslineScriptStatus_NoFile asserts a missing script reports
// existed=false, aiwfAuthored=false, no error (G-0354).
func TestStatuslineScriptStatus_NoFile(t *testing.T) {
	t.Parallel()
	dest := filepath.Join(t.TempDir(), "statusline.sh")

	existed, aiwfAuthored, err := StatuslineScriptStatus(dest)
	if err != nil {
		t.Fatalf("StatuslineScriptStatus: %v", err)
	}
	if existed || aiwfAuthored {
		t.Errorf("missing script must report existed=false, aiwfAuthored=false; got existed=%v aiwfAuthored=%v", existed, aiwfAuthored)
	}
}

// TestStatuslineScriptStatus_AiwfAuthored asserts a script carrying the
// aiwf version marker reports aiwfAuthored=true.
func TestStatuslineScriptStatus_AiwfAuthored(t *testing.T) {
	t.Parallel()
	dest := filepath.Join(t.TempDir(), "statusline.sh")
	if err := os.WriteFile(dest, RenderStatusline("v1.2.3"), 0o755); err != nil {
		t.Fatal(err)
	}

	existed, aiwfAuthored, err := StatuslineScriptStatus(dest)
	if err != nil {
		t.Fatalf("StatuslineScriptStatus: %v", err)
	}
	if !existed || !aiwfAuthored {
		t.Errorf("aiwf-marked script must report existed=true, aiwfAuthored=true; got existed=%v aiwfAuthored=%v", existed, aiwfAuthored)
	}
}

// TestStatuslineScriptStatus_Foreign asserts a script without the aiwf
// marker (hand-authored / foreign) reports aiwfAuthored=false.
func TestStatuslineScriptStatus_Foreign(t *testing.T) {
	t.Parallel()
	dest := filepath.Join(t.TempDir(), "statusline.sh")
	content := []byte("#!/usr/bin/env bash\necho hand-written\n")
	if err := os.WriteFile(dest, content, 0o755); err != nil {
		t.Fatal(err)
	}

	existed, aiwfAuthored, err := StatuslineScriptStatus(dest)
	if err != nil {
		t.Fatalf("StatuslineScriptStatus: %v", err)
	}
	if !existed || aiwfAuthored {
		t.Errorf("foreign script must report existed=true, aiwfAuthored=false; got existed=%v aiwfAuthored=%v", existed, aiwfAuthored)
	}
}

// TestStatuslineScriptStatus_ReadError asserts a non-ENOENT read
// failure (dest is a directory) surfaces as an error.
func TestStatuslineScriptStatus_ReadError(t *testing.T) {
	t.Parallel()
	dest := filepath.Join(t.TempDir(), "statusline.sh")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, _, err := StatuslineScriptStatus(dest); err == nil {
		t.Error("expected an error when dest is a directory")
	}
}

// TestRemoveStatuslineScriptFile_NoFile asserts a missing script is a
// no-op: removed=false, no error.
func TestRemoveStatuslineScriptFile_NoFile(t *testing.T) {
	t.Parallel()
	dest := filepath.Join(t.TempDir(), "statusline.sh")

	removed, err := RemoveStatuslineScriptFile(dest)
	if err != nil {
		t.Fatalf("RemoveStatuslineScriptFile: %v", err)
	}
	if removed {
		t.Error("missing script must report removed=false")
	}
}

// TestRemoveStatuslineScriptFile_DeletesUnconditionally asserts the
// file is deleted regardless of content — the caller has already
// decided this removal is authorized (aiwf-authored, or --force).
func TestRemoveStatuslineScriptFile_DeletesUnconditionally(t *testing.T) {
	t.Parallel()
	dest := filepath.Join(t.TempDir(), "statusline.sh")
	if err := os.WriteFile(dest, []byte("#!/usr/bin/env bash\necho hand-written\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	removed, err := RemoveStatuslineScriptFile(dest)
	if err != nil {
		t.Fatalf("RemoveStatuslineScriptFile: %v", err)
	}
	if !removed {
		t.Error("existing script must report removed=true")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("script must be deleted from disk, stat err=%v", err)
	}
}

// TestRemoveStatuslineScriptFile_RemoveError asserts a non-ENOENT
// os.Remove failure (dest is a non-empty directory, which os.Remove
// refuses) surfaces as an error.
func TestRemoveStatuslineScriptFile_RemoveError(t *testing.T) {
	t.Parallel()
	dest := filepath.Join(t.TempDir(), "statusline.sh")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "nested"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := RemoveStatuslineScriptFile(dest); err == nil {
		t.Error("expected an error when dest is a non-empty directory")
	}
}

// TestStatuslineDestForScope asserts the exported wrapper resolves the
// same destination + command the scaffold would write, for both scopes
// (the read-only counterpart used by `aiwf update --remove`).
func TestStatuslineDestForScope(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	home := t.TempDir()

	projDest, projCmd, err := StatuslineDestForScope(root, home, StatuslineScopeProject)
	if err != nil {
		t.Fatalf("StatuslineDestForScope(project): %v", err)
	}
	if want := filepath.Join(root, statuslineRelPath); projDest != want {
		t.Errorf("project dest = %q, want %q", projDest, want)
	}
	if projCmd != ProjectStatuslineCommand(root) {
		t.Errorf("project cmd = %q, want %q", projCmd, ProjectStatuslineCommand(root))
	}

	userDest, userCmd, err := StatuslineDestForScope(root, home, StatuslineScopeUser)
	if err != nil {
		t.Fatalf("StatuslineDestForScope(user): %v", err)
	}
	if want := filepath.Join(home, statuslineRelPath); userDest != want {
		t.Errorf("user dest = %q, want %q", userDest, want)
	}
	if userCmd != UserStatuslineCommand() {
		t.Errorf("user cmd = %q, want %q", userCmd, UserStatuslineCommand())
	}

	if _, _, err := StatuslineDestForScope(root, home, StatuslineScope("bogus")); err == nil {
		t.Error("expected an error for an unknown scope")
	}
}
