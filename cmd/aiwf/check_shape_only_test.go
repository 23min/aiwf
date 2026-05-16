package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestRun_CheckShapeOnly_CleanTree exits 0 with no findings on a
// repo that only contains recognized entity files.
func TestRun_CheckShapeOnly_CleanTree(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := run([]string{"add", "gap", "--title", "Real gap", "--root", root, "--actor", "human/test"}); rc != cliutil.ExitOK {
		t.Fatalf("add gap: %d", rc)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"check", "--shape-only", "--root", root}); rc != cliutil.ExitOK {
			t.Errorf("got rc=%d, want %d (clean)", rc, cliutil.ExitOK)
		}
	})
	if strings.Contains(string(captured), "unexpected-tree-file") {
		t.Errorf("clean tree should not produce findings:\n%s", captured)
	}
}

// TestRun_CheckShapeOnly_StrayWarning_ExitOK: with tree.strict
// unset (the default), a stray under work/ is a warning. The verb
// prints the finding but exits 0 — the pre-commit hook proceeds.
func TestRun_CheckShapeOnly_StrayWarning_ExitOK(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	if err := os.MkdirAll(filepath.Join(root, "work", "gaps"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "work", "gaps", "scratch.md"), []byte("not an entity\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"check", "--shape-only", "--root", root}); rc != cliutil.ExitOK {
			t.Errorf("got rc=%d, want %d (warning, not blocking)", rc, cliutil.ExitOK)
		}
	})
	out := string(captured)
	if !strings.Contains(out, "unexpected-tree-file") {
		t.Errorf("expected `unexpected-tree-file` finding:\n%s", out)
	}
	if !strings.Contains(out, "work/gaps/scratch.md") {
		t.Errorf("expected stray path in output:\n%s", out)
	}
}

// TestRun_CheckShapeOnly_StrayStrict_ExitFindings: with
// tree.strict: true, the same stray promotes to error and the verb
// exits with findings — the pre-commit hook blocks the commit.
func TestRun_CheckShapeOnly_StrayStrict_ExitFindings(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	// Append tree.strict: true to the aiwf.yaml init wrote.
	yamlPath := filepath.Join(root, "aiwf.yaml")
	existing, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(yamlPath, append(existing, []byte("\ntree:\n  strict: true\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "work", "epics"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "work", "epics", "stray.md"), []byte("not an entity\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"check", "--shape-only", "--root", root}); rc != cliutil.ExitFindings {
			t.Errorf("got rc=%d, want %d (strict mode must block)", rc, cliutil.ExitFindings)
		}
	})
	if !strings.Contains(string(captured), "unexpected-tree-file") {
		t.Errorf("expected `unexpected-tree-file` finding under strict mode:\n%s", captured)
	}
}

// TestRun_CheckShapeOnly_AllowPathsExempt: a stray that matches
// aiwf.yaml: tree.allow_paths is exempt — no finding even with
// strict mode on.
func TestRun_CheckShapeOnly_AllowPathsExempt(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	yamlPath := filepath.Join(root, "aiwf.yaml")
	existing, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatal(err)
	}
	yaml := string(existing) + "\ntree:\n  strict: true\n  allow_paths:\n    - work/templates/*.md\n"
	if err := os.WriteFile(yamlPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "work", "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "work", "templates", "epic.md"), []byte("template body\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	captured := captureStdout(t, func() {
		if rc := run([]string{"check", "--shape-only", "--root", root}); rc != cliutil.ExitOK {
			t.Errorf("got rc=%d, want %d (allow_paths must exempt)", rc, cliutil.ExitOK)
		}
	})
	if strings.Contains(string(captured), "unexpected-tree-file") {
		t.Errorf("allow_paths exemption did not apply:\n%s", captured)
	}
}
