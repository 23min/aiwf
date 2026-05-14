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
	t.Parallel()
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
// silently run the tree-discipline gate on a repo that hasn't
// adopted aiwf.
func TestPreCommitHookScript_HasBrownfieldGuard(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
// same brownfield guard on the pre-commit hook. The baked binary is
// /bin/false; if the guard fails the gate runs and /bin/false's
// non-zero exit aborts the hook.
func TestPreCommitHookScript_NoAiwfYamlExitsSilently(t *testing.T) {
	t.Parallel()
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
}

// TestPreCommitHookScript_AiwfYamlPresentRunsGate asserts the guard
// passes through to the gate when aiwf.yaml exists. Per G-0112 the
// pre-commit hook no longer regenerates STATUS.md, so the only body
// reach we test is the gate (a succeed-shim exits 0; if the gate
// runs, the hook itself exits 0).
func TestPreCommitHookScript_AiwfYamlPresentRunsGate(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"),
		[]byte("aiwf_version: 0.1.0\nactor: human/peter\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	shim := filepath.Join(t.TempDir(), "succeed-shim")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(t.TempDir(), "pre-commit.sh")
	if err := os.WriteFile(hookPath, []byte(preCommitHookScript(shim)), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("/bin/sh", hookPath)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pre-commit hook exited non-zero with aiwf.yaml present:\n%s\nerr: %v", out, err)
	}
	// Per G-0112 the pre-commit hook must never produce STATUS.md.
	if _, err := os.Stat(filepath.Join(root, "STATUS.md")); !os.IsNotExist(err) {
		t.Errorf("pre-commit hook produced STATUS.md (G-0112 regression — regen lives in post-commit now):\n%v", err)
	}
}

// TestPreCommitHookScript_CheckFailureBlocksCommit pins G41's
// load-bearing behavior: when `aiwf check --shape-only` fails (here
// simulated with a fail-shim that always exits 1), the pre-commit
// hook exits non-zero and blocks the commit. Without this, the
// kernel's tree-discipline guarantee would still only fire at
// pre-push time — the LLM-loop signal G41 added would be silently
// dropped.
func TestPreCommitHookScript_CheckFailureBlocksCommit(t *testing.T) {
	t.Parallel()
	root := freshGitRepo(t)
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"),
		[]byte("aiwf_version: 0.1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	shim := filepath.Join(t.TempDir(), "fail-shim")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(t.TempDir(), "pre-commit.sh")
	if err := os.WriteFile(hookPath, []byte(preCommitHookScript(shim)), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("/bin/sh", hookPath)
	cmd.Dir = root
	if err := cmd.Run(); err == nil {
		t.Fatal("pre-commit hook exited 0; expected non-zero (tree-discipline gate must block)")
	}
}

// TestPreCommitHookScript_InvokesShapeOnly pins the verb invocation
// shape — drift here means a future refactor of `aiwf check`'s flags
// could silently break the pre-commit gate. Asserts the script body
// contains `check --shape-only`, the load-bearing verb call.
func TestPreCommitHookScript_InvokesShapeOnly(t *testing.T) {
	t.Parallel()
	body := preCommitHookScript("/some/aiwf")
	if !strings.Contains(body, "check --shape-only") {
		t.Errorf("pre-commit hook missing `check --shape-only` invocation; tree-discipline gate would not fire:\n%s", body)
	}
}
