package pathutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAtomicWriteFile_CreatesNewFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.md")
	if err := AtomicWriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("AtomicWriteFile: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "hello\n" {
		t.Errorf("content = %q, want %q", got, "hello\n")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("perm = %v, want 0644", info.Mode().Perm())
	}
}

func TestAtomicWriteFile_OverwritesExistingFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.md")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatalf("seeding: %v", err)
	}
	if err := AtomicWriteFile(path, []byte("new"), 0o644); err != nil {
		t.Fatalf("AtomicWriteFile: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "new" {
		t.Errorf("content = %q, want %q", got, "new")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("perm = %v, want 0644 (rename replaces mode)", info.Mode().Perm())
	}
}

func TestAtomicWriteFile_AppliesExecutablePerm(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "hook")
	if err := AtomicWriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("AtomicWriteFile: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("perm = %v, want 0755", info.Mode().Perm())
	}
}

func TestAtomicWriteFile_MissingParentDirErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "nope", "out.md")
	err := AtomicWriteFile(path, []byte("x"), 0o644)
	if err == nil {
		t.Fatal("expected error for missing parent dir, got nil")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not name the target path %q", err, path)
	}
}

// TestAtomicWriteFile_RenameOntoDirectoryFailsAndCleansTemp pins the
// rename-failure branch and the "temp file is removed on every error
// path" contract: a directory occupying path makes os.Rename fail
// after the temp file was written and synced.
func TestAtomicWriteFile_RenameOntoDirectoryFailsAndCleansTemp(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "occupied")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("seeding directory: %v", err)
	}
	err := AtomicWriteFile(path, []byte("x"), 0o644)
	if err == nil {
		t.Fatal("expected error renaming onto an existing directory, got nil")
	}
	entries, rdErr := os.ReadDir(dir)
	if rdErr != nil {
		t.Fatalf("ReadDir: %v", rdErr)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".aiwf-tmp-") {
			t.Errorf("stray temp file left behind after rename failure: %s", e.Name())
		}
	}
}

func TestAtomicWriteFile_LeavesNoTempFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.md")
	if err := AtomicWriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatalf("AtomicWriteFile: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".aiwf-tmp-") {
			t.Errorf("stray temp file left behind: %s", e.Name())
		}
	}
	if len(entries) != 1 {
		t.Errorf("dir has %d entries, want exactly the target file", len(entries))
	}
}
