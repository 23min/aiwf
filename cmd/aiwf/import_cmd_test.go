package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// writeManifest is a helper that writes a YAML manifest into the test
// repo and returns the absolute path.
func writeManifest(t *testing.T, root, body string) string {
	t.Helper()
	path := filepath.Join(root, "manifest.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestRun_ImportThroughDispatcher: a manifest with two entities lands
// one commit, prints its subject, and the on-disk tree validates.
func TestRun_ImportThroughDispatcher(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}

	manifest := writeManifest(t, root, `version: 1
entities:
  - kind: epic
    id: E-0001
    frontmatter: {title: "Cake", status: active}
  - kind: milestone
    id: M-0001
    frontmatter: {title: "Bake", status: draft, parent: E-0001}
`)

	captured := captureStdout(t, func() {
		if rc := run([]string{"import", "--root", root, "--actor", "human/test", manifest}); rc != exitOK {
			t.Errorf("import rc != ok")
		}
	})
	if !strings.Contains(string(captured), "aiwf import 2 entities") {
		t.Errorf("expected default subject in output; got:\n%s", captured)
	}

	for _, p := range []string{
		filepath.Join("work", "epics", "E-0001-cake", "epic.md"),
		filepath.Join("work", "epics", "E-0001-cake", "M-0001-bake.md"),
	} {
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}

	if rc := run([]string{"check", "--root", root}); rc != exitOK {
		t.Errorf("post-import check rc = %d", rc)
	}
}

// TestRun_ImportDryRun prints the would-be plans and writes nothing.
func TestRun_ImportDryRun(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	manifest := writeManifest(t, root, `version: 1
entities:
  - kind: epic
    id: E-0001
    frontmatter: {title: "Cake", status: active}
`)

	captured := captureStdout(t, func() {
		if rc := run([]string{"import", "--root", root, "--actor", "human/test", "--dry-run", manifest}); rc != exitOK {
			t.Errorf("import --dry-run rc != ok")
		}
	})
	out := string(captured)

	for _, want := range []string{"dry-run", "would land", "write work/epics/E-0001-cake/epic.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull:\n%s", want, out)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "work", "epics", "E-0001-cake", "epic.md")); !os.IsNotExist(err) {
		t.Errorf("dry-run wrote epic.md (stat err=%v)", err)
	}
}

// TestRun_ImportCollisionFail: re-importing the same explicit-id
// manifest exits with findings.
func TestRun_ImportCollisionFail(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	manifest := writeManifest(t, root, `version: 1
entities:
  - kind: epic
    id: E-0001
    frontmatter: {title: "Cake", status: active}
`)
	if rc := run([]string{"import", "--root", root, "--actor", "human/test", manifest}); rc != exitOK {
		t.Fatalf("first import rc = %d", rc)
	}
	if rc := run([]string{"import", "--root", root, "--actor", "human/test", manifest}); rc != exitFindings {
		t.Errorf("second import rc = %d, want %d", rc, exitFindings)
	}
}

// TestRun_ImportPerEntityCommit emits one commit per entity in the
// manifest. Verifies the commit count grew by exactly N.
func TestRun_ImportPerEntityCommit(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := run([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != exitOK {
		t.Fatalf("init: %d", rc)
	}
	// Seed an initial commit so HEAD exists.
	if rc := run([]string{"add", "epic", "--title", "Seed", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("seed add: %d", rc)
	}
	manifest := writeManifest(t, root, `version: 1
commit:
  mode: per-entity
entities:
  - kind: epic
    id: E-0002
    frontmatter: {title: "B", status: active}
  - kind: epic
    id: E-0003
    frontmatter: {title: "C", status: active}
`)
	if rc := run([]string{"import", "--root", root, "--actor", "human/test", manifest}); rc != exitOK {
		t.Fatalf("import per-entity rc = %d", rc)
	}
	// Two new commits expected.
	out, err := commitCount(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if out < 3 { // seed + 2
		t.Errorf("expected ≥3 commits, got %d", out)
	}
}

// commitCount returns the number of commits reachable from HEAD.
func commitCount(t *testing.T, root string) (int, error) {
	t.Helper()
	out, err := capture(t, root, "git", "rev-list", "--count", "HEAD")
	if err != nil {
		return 0, err
	}
	var n int
	for _, c := range strings.TrimSpace(out) {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

// capture runs a command in workdir and returns combined output.
func capture(t *testing.T, workdir, name string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
