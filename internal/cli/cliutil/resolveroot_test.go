package cliutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TestResolveRoot_ExplicitWins covers the happy path where --root is
// passed: the explicit value is resolved to an absolute path and
// returned unchanged regardless of cwd's surroundings.
func TestResolveRoot_ExplicitWins(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	got, err := ResolveRoot(tmp)
	if err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(tmp)
	if got != abs {
		t.Errorf("got %q, want %q", got, abs)
	}
}

// TestWalkUpFor is the internal-package unit test for the unexported
// walkUpFor helper. ResolveRoot's happy path (find aiwf.yaml by
// walking up) and the negative path (not found) are exercised here.
func TestWalkUpFor(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "marker.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, ok := walkUpFor(deep, "marker.txt")
	if !ok {
		t.Fatal("not found")
	}
	if got != root {
		t.Errorf("got %q, want %q", got, root)
	}
	if _, ok := walkUpFor(deep, "nonsuch.txt"); ok {
		t.Errorf("nonsuch.txt should not be found")
	}
}
