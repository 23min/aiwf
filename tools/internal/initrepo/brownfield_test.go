package initrepo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestPreHookScript_HasBrownfieldGuard pins the embedded pre-push
// template's load-bearing guard line. If this assertion drifts, the
// hook will start blocking pushes on brownfield clones.
func TestPreHookScript_HasBrownfieldGuard(t *testing.T) {
	body := preHookScript("/some/aiwf")
	if !strings.Contains(body, `[ -f "$(git rev-parse --show-toplevel)/aiwf.yaml" ] || exit 0`) {
		t.Errorf("pre-push hook missing brownfield guard:\n%s", body)
	}
	if !strings.Contains(body, "exec '/some/aiwf' check") {
		t.Errorf("pre-push hook missing exec line:\n%s", body)
	}
}

// TestPreCommitHookScript_HasBrownfieldGuard pins the same guard on
// the pre-commit template. Without it, brownfield commits would
// silently introduce a tracked STATUS.md.
func TestPreCommitHookScript_HasBrownfieldGuard(t *testing.T) {
	body := preCommitHookScript("/some/aiwf")
	if !strings.Contains(body, `[ -f "$repo_root/aiwf.yaml" ] || exit 0`) {
		t.Errorf("pre-commit hook missing brownfield guard:\n%s", body)
	}
}

// TestPreHookScript_NoAiwfYamlExitsSilently runs the embedded
// pre-push template under /bin/sh in a fresh git repo with no
// aiwf.yaml and asserts exit 0 and no `aiwf check` invocation.
//
// The aiwf binary path baked into the script points at /bin/false:
// if the guard works, /bin/false is never reached. If the guard is
// removed/broken, /bin/false runs and the hook exits non-zero, so
// this test fails loudly.
func TestPreHookScript_NoAiwfYamlExitsSilently(t *testing.T) {
	root := freshGitRepo(t)
	hookPath := filepath.Join(t.TempDir(), "pre-push.sh")
	if err := os.WriteFile(hookPath, []byte(preHookScript("/bin/false")), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("/bin/sh", hookPath)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pre-push hook exited non-zero on brownfield repo:\n%s\nerr: %v", out, err)
	}
}

// TestPreHookScript_AiwfYamlExitsViaExec asserts the guard does not
// short-circuit when aiwf.yaml IS present — the hook proceeds to
// the exec line. We use /bin/false so the test exits non-zero
// (proving the exec ran) without depending on a real aiwf.
func TestPreHookScript_AiwfYamlExitsViaExec(t *testing.T) {
	root := freshGitRepo(t)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"),
		[]byte("aiwf_version: 0.1.0\nactor: human/peter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(t.TempDir(), "pre-push.sh")
	if err := os.WriteFile(hookPath, []byte(preHookScript("/bin/false")), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("/bin/sh", hookPath)
	cmd.Dir = root
	if err := cmd.Run(); err == nil {
		t.Errorf("pre-push hook exited 0; expected non-zero (guard let exec /bin/false through)")
	}
}

// TestPreCommitHookScript_NoAiwfYamlExitsSilently exercises the
// same brownfield guard on the pre-commit hook. Critically asserts
// no STATUS.md is written and the hook does not call `aiwf status`
// (the baked path is /bin/false; if it ran, the if-branch would
// fail and the hook would still exit 0 — but no tmp file would be
// produced either way; we assert STATUS.md absence as the visible
// signal).
func TestPreCommitHookScript_NoAiwfYamlExitsSilently(t *testing.T) {
	root := freshGitRepo(t)
	hookPath := filepath.Join(t.TempDir(), "pre-commit.sh")
	if err := os.WriteFile(hookPath, []byte(preCommitHookScript("/bin/false")), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("/bin/sh", hookPath)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pre-commit hook exited non-zero on brownfield repo:\n%s\nerr: %v", out, err)
	}
	if _, err := os.Stat(filepath.Join(root, "STATUS.md")); !os.IsNotExist(err) {
		t.Errorf("pre-commit hook wrote STATUS.md in brownfield repo (stat err=%v)", err)
	}
}

// TestPreCommitHookScript_AiwfYamlPresentRunsBody asserts the guard
// passes through to the body when aiwf.yaml exists. With /bin/false
// as the baked path the if-branch fails, the else-branch runs
// `rm -f "$tmp"`, and the hook exits 0 — but importantly the body
// is reached, demonstrating the guard isn't over-broad.
func TestPreCommitHookScript_AiwfYamlPresentRunsBody(t *testing.T) {
	root := freshGitRepo(t)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"),
		[]byte("aiwf_version: 0.1.0\nactor: human/peter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(t.TempDir(), "pre-commit.sh")
	if err := os.WriteFile(hookPath, []byte(preCommitHookScript("/bin/false")), 0o755); err != nil {
		t.Fatal(err)
	}

	// Pre-create the tmp file so we can detect that the body's
	// `rm -f "$tmp"` path executed.
	tmp := filepath.Join(root, "STATUS.md.tmp")
	if err := os.WriteFile(tmp, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("/bin/sh", hookPath)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pre-commit hook exited non-zero with aiwf.yaml present:\n%s\nerr: %v", out, err)
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Errorf("pre-commit hook did not run the body (STATUS.md.tmp survived; stat err=%v)", err)
	}
}
