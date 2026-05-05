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
	body := preCommitHookScript("/some/aiwf", true)
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
	if err := os.WriteFile(hookPath, []byte(preCommitHookScript("/bin/false", true)), 0o755); err != nil {
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
// passes through to the body when aiwf.yaml exists. The baked
// "binary" is a shell-shim that exits 0 — both the tree-discipline
// check and the STATUS.md regen succeed; the body is reached and
// STATUS.md gets written by `mv`.
//
// (Pre-G41 the test used /bin/false to exercise the body's tolerant
// rm-the-tmp branch. With G41 the check step is now non-tolerant —
// a failing binary would block the commit before reaching status.
// A succeed-shim keeps the test focused on "the body runs.")
func TestPreCommitHookScript_AiwfYamlPresentRunsBody(t *testing.T) {
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
	if err := os.WriteFile(hookPath, []byte(preCommitHookScript(shim, true)), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("/bin/sh", hookPath)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pre-commit hook exited non-zero with aiwf.yaml present:\n%s\nerr: %v", out, err)
	}
	// The shim ignores its args and exits 0 with no stdout, so the
	// status step's `>"$tmp"` truncates tmp to empty and `mv` moves
	// it to STATUS.md. STATUS.md's existence proves the body ran
	// past the tree-discipline gate into the regen step.
	if _, err := os.Stat(filepath.Join(root, "STATUS.md")); err != nil {
		t.Errorf("pre-commit hook did not reach STATUS.md regen step: %v", err)
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
	if err := os.WriteFile(hookPath, []byte(preCommitHookScript(shim, true)), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("/bin/sh", hookPath)
	cmd.Dir = root
	if err := cmd.Run(); err == nil {
		t.Fatal("pre-commit hook exited 0; expected non-zero (tree-discipline gate must block)")
	}
	// STATUS.md must NOT have been written — the gate fails before
	// the regen step runs.
	if _, err := os.Stat(filepath.Join(root, "STATUS.md")); !os.IsNotExist(err) {
		t.Errorf("STATUS.md exists after a blocked commit; the gate did not short-circuit (stat err=%v)", err)
	}
}

// TestPreCommitHookScript_InvokesShapeOnly pins the verb invocation
// shape — drift here means a future refactor of `aiwf check`'s flags
// could silently break the pre-commit gate. Asserts the script body
// contains `check --shape-only`, the load-bearing verb call.
func TestPreCommitHookScript_InvokesShapeOnly(t *testing.T) {
	body := preCommitHookScript("/some/aiwf", true)
	if !strings.Contains(body, "check --shape-only") {
		t.Errorf("pre-commit hook missing `check --shape-only` invocation; tree-discipline gate would not fire:\n%s", body)
	}
}

// TestPreCommitHookScript_RegenStatus_Decoupling pins the G42
// contract: the tree-discipline gate is always present, and only
// the STATUS.md regen step toggles with regenStatus. Without this
// test, a refactor that re-coupled the two responsibilities would
// silently regress G42's enforcement guarantee.
func TestPreCommitHookScript_RegenStatus_Decoupling(t *testing.T) {
	withRegen := preCommitHookScript("/some/aiwf", true)
	withoutRegen := preCommitHookScript("/some/aiwf", false)

	for _, body := range []string{withRegen, withoutRegen} {
		if !strings.Contains(body, "check --shape-only") {
			t.Errorf("gate missing regardless of regenStatus:\n%s", body)
		}
	}
	if !strings.Contains(withRegen, "status --root") {
		t.Errorf("regenStatus=true should include status regen:\n%s", withRegen)
	}
	if strings.Contains(withoutRegen, "status --root") {
		t.Errorf("regenStatus=false must omit status regen (G42 decoupling):\n%s", withoutRegen)
	}
}
