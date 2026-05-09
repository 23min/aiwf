package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRun_RenderHTML_WarnsWhenOutDirNotGitignored: when --out points
// at a path inside the repo that isn't covered by .gitignore, the
// render verb emits a defense-in-depth stderr warning. This is the
// case the init/update reconciliation cannot catch — the operator
// passed an ad-hoc --out that aiwf init never saw. Closes G-056.
func TestRun_RenderHTML_WarnsWhenOutDirNotGitignored(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)

	stderr := captureStderr(t, func() {
		_ = captureStdout(t, func() {
			mustRun(t, "render", "--root", root, "--format", "html", "--out", "custom-render-dir")
		})
	})

	want := "is not gitignored"
	if !strings.Contains(string(stderr), want) {
		t.Errorf("expected stderr to contain %q; got:\n%s", want, stderr)
	}
}

// TestRun_RenderHTML_SilentWhenOutDirGitignored: when the resolved
// output dir is covered by .gitignore (the steady-state for any
// consumer that has run aiwf init/update with the default
// commit_output: false), the verb stays silent on stderr.
func TestRun_RenderHTML_SilentWhenOutDirGitignored(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)

	// Sanity: aiwf init should have written `site/` into .gitignore.
	gi, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(gi), "site/") {
		t.Fatalf("expected aiwf init to add site/ to .gitignore; got:\n%s", gi)
	}

	stderr := captureStderr(t, func() {
		_ = captureStdout(t, func() {
			mustRun(t, "render", "--root", root, "--format", "html")
		})
	})

	if strings.Contains(string(stderr), "is not gitignored") {
		t.Errorf("expected silent stderr when out_dir is gitignored; got:\n%s", stderr)
	}
}

// TestRun_RenderHTML_SilentWhenOutDirOutsideRoot: when --out is an
// absolute path that escapes the repo root, .gitignore semantics
// don't apply (the file lives in a different working tree, possibly
// none). The warning must stay silent so a deliberate "render to
// /tmp/" doesn't spam stderr.
func TestRun_RenderHTML_SilentWhenOutDirOutsideRoot(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)

	out := filepath.Join(t.TempDir(), "site-elsewhere")
	stderr := captureStderr(t, func() {
		_ = captureStdout(t, func() {
			mustRun(t, "render", "--root", root, "--format", "html", "--out", out)
		})
	})

	if strings.Contains(string(stderr), "is not gitignored") {
		t.Errorf("expected silent stderr when out is outside repo root; got:\n%s", stderr)
	}
}

// TestRun_RenderHTML_SilentWhenCommitOutputTrue: when aiwf.yaml has
// html.commit_output: true the operator opted into tracking the
// rendered files; .gitignore by design does not cover them. The
// warning must be silent regardless of gitignore status, because the
// operator's intent is explicit.
func TestRun_RenderHTML_SilentWhenCommitOutputTrue(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)

	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "html:\n  commit_output: true\n"
	if err := os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	// Reconcile the gitignore so site/ is removed (matches the
	// shape a real consumer reaches by running `aiwf update`).
	mustRun(t, "update", "--root", root)

	stderr := captureStderr(t, func() {
		_ = captureStdout(t, func() {
			mustRun(t, "render", "--root", root, "--format", "html")
		})
	})

	if strings.Contains(string(stderr), "is not gitignored") {
		t.Errorf("expected silent stderr when commit_output:true; got:\n%s", stderr)
	}
}
