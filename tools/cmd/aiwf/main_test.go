package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRun_NoArgs_UsageError(t *testing.T) {
	if got := run(nil); got != exitUsage {
		t.Errorf("run(nil) = %d, want %d", got, exitUsage)
	}
}

func TestRun_UnknownVerb_UsageError(t *testing.T) {
	if got := run([]string{"yodel"}); got != exitUsage {
		t.Errorf("run(yodel) = %d, want %d", got, exitUsage)
	}
}

func TestRun_HelpVariants(t *testing.T) {
	for _, arg := range []string{"help", "--help", "-h"} {
		t.Run(arg, func(t *testing.T) {
			if got := run([]string{arg}); got != exitOK {
				t.Errorf("run(%q) = %d, want %d", arg, got, exitOK)
			}
		})
	}
}

func TestRun_VersionVariants(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-v"} {
		t.Run(arg, func(t *testing.T) {
			if got := run([]string{arg}); got != exitOK {
				t.Errorf("run(%q) = %d, want %d", arg, got, exitOK)
			}
		})
	}
}

func TestRun_CheckEmptyRepo_OK(t *testing.T) {
	root := t.TempDir()
	if got := run([]string{"check", "--root=" + root}); got != exitOK {
		t.Errorf("run(check on empty) = %d, want %d", got, exitOK)
	}
}

func TestRun_CheckBadFormat_UsageError(t *testing.T) {
	root := t.TempDir()
	if got := run([]string{"check", "--root=" + root, "--format=xml"}); got != exitUsage {
		t.Errorf("got %d, want %d", got, exitUsage)
	}
}

func TestRun_CheckFindsErrors(t *testing.T) {
	root := t.TempDir()
	// Create a milestone with a bad parent reference and a bad status.
	dir := filepath.Join(root, "work", "epics", "E-01-foo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "epic.md"), []byte(`---
id: E-01
title: Foo
status: active
---
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "M-001-bar.md"), []byte(`---
id: M-001
title: Bar
status: bogus
parent: E-99
---
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := run([]string{"check", "--root=" + root}); got != exitFindings {
		t.Errorf("got %d, want %d (findings)", got, exitFindings)
	}
}

func TestResolveRoot_ExplicitWins(t *testing.T) {
	tmp := t.TempDir()
	got, err := resolveRoot(tmp)
	if err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(tmp)
	if got != abs {
		t.Errorf("got %q, want %q", got, abs)
	}
}

func TestWalkUpFor(t *testing.T) {
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
