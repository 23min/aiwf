package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// buildOnce caches the path to a freshly-built `aiwf` binary so the
// integration tests in this file share one compile.
var (
	buildOnce sync.Once
	builtPath string
	buildErr  error
)

// aiwfBinary returns the absolute path to a built `aiwf` binary,
// compiling on the first call. The binary lives in a per-process temp
// dir so concurrent `go test` runs don't fight over it.
func aiwfBinary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "aiwf-int-build-")
		if err != nil {
			buildErr = err
			return
		}
		bin := filepath.Join(dir, "aiwf")
		if runtime.GOOS == "windows" {
			bin += ".exe"
		}
		// Find the repo root by walking up from this file's package
		// dir to where go.mod lives. cmd/aiwf -> .. -> ..
		// The test binary's working dir is the package dir, so
		// `./...` is wrong; pass an absolute module path.
		cmd := exec.Command("go", "build", "-o", bin, "github.com/23min/ai-workflow-v2/cmd/aiwf")
		out, err := cmd.CombinedOutput()
		if err != nil {
			buildErr = &buildError{err: err, output: string(out)}
			return
		}
		builtPath = bin
	})
	if buildErr != nil {
		t.Fatalf("building aiwf: %v", buildErr)
	}
	return builtPath
}

type buildError struct {
	err    error
	output string
}

func (e *buildError) Error() string { return e.err.Error() + "\n" + e.output }

// runBin runs the built binary with args in workdir, prepending
// extraPath onto PATH. Returns combined output and exit error.
func runBin(t *testing.T, workdir, extraPath string, env []string, args ...string) (string, error) {
	t.Helper()
	bin := aiwfBinary(t)
	cmd := exec.Command(bin, args...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=aiwf-test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=aiwf-test",
		"GIT_COMMITTER_EMAIL=test@example.com",
		"PATH="+extraPath+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	cmd.Env = append(cmd.Env, env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// TestIntegration_FreshRepoLifecycle is the end-to-end smoke test:
// build the binary, init a fresh consumer repo, add an entity, run
// the installed pre-push hook directly to confirm it actually fires
// `aiwf check` and reports cleanly. Then break the tree and confirm
// the same hook now exits non-zero.
//
// This is the test that says "yes, the framework works in a real
// consumer repo, not just inside Go test fixtures."
func TestIntegration_FreshRepoLifecycle(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	// `git init` the consumer.
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	// Local git identity; the binary derives actor from this.
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	// 1. aiwf init.
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}

	// aiwf.yaml exists. Two legacy fields must be absent on a fresh
	// init: `actor:` (I2.5 — identity is runtime-derived) and
	// `aiwf_version:` (G47 — the running binary self-reports).
	cfg, err := os.ReadFile(filepath.Join(root, "aiwf.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cfg), "actor:") {
		t.Errorf("aiwf.yaml contains actor: (post-I2.5 init must omit it): %s", cfg)
	}
	if strings.Contains(string(cfg), "aiwf_version:") {
		t.Errorf("aiwf.yaml contains aiwf_version: (post-G47 init must omit it): %s", cfg)
	}

	// pre-push hook exists, is executable, and bakes in the absolute
	// path of the binary (so it doesn't depend on $PATH at push time).
	hookPath := filepath.Join(root, ".git", "hooks", "pre-push")
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("pre-push hook is not executable: %v", info.Mode())
	}
	hookContent, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(hookContent), bin) {
		t.Errorf("hook should contain absolute binary path %q; got:\n%s", bin, hookContent)
	}

	// 2. add an epic — should succeed and produce one commit.
	if addOut, addErr := runBin(t, root, binDir, nil, "add", "epic", "--title", "Foundations"); addErr != nil {
		t.Fatalf("aiwf add: %v\n%s", addErr, addOut)
	}

	// 3. Run the installed hook directly *without* the binary dir on
	// PATH. The hook bakes in the absolute path, so this should still
	// work — that's the entire point of fix-#1.
	if hookOut, hookErr := runHook(t, root, ""); hookErr != nil {
		t.Fatalf("hook on clean tree should pass; got %v\n%s", hookErr, hookOut)
	}

	// 4. Break the tree by introducing a milestone with an unresolved
	// parent reference. The hook should fail, again with no PATH help.
	bad := []byte("---\nid: M-001\ntitle: Broken\nstatus: draft\nparent: E-99\n---\n")
	if wErr := os.WriteFile(filepath.Join(root, "work", "epics", "E-01-foundations", "M-001-bad.md"), bad, 0o644); wErr != nil {
		t.Fatal(wErr)
	}
	out, hookErr := runHook(t, root, "")
	if hookErr == nil {
		t.Fatalf("hook should have failed on broken tree; output:\n%s", out)
	}
	if !strings.Contains(out, "refs-resolve") {
		t.Errorf("hook output should mention the failing check; got:\n%s", out)
	}
	// Silence linter on now-unused binDir variable.
	_ = binDir
}

// runGit invokes git in workdir and returns combined output.
func runGit(workdir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=aiwf-test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=aiwf-test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// TestIntegration_TrunkExplicitMissingIsHardError confirms the
// strict policy: when allocate.trunk is set explicitly in aiwf.yaml
// but the named ref doesn't resolve, the verb fails loudly. (The
// unit test in package trunk pins the package-level error message;
// this test pins that the cmd surfaces it through the binary.)
func TestIntegration_TrunkExplicitMissingIsHardError(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q", "-b", "main"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	// Configure a trunk ref that doesn't exist, plus a remote so the
	// no-remote skip doesn't kick in.
	yamlPath := filepath.Join(root, "aiwf.yaml")
	existing, readErr := os.ReadFile(yamlPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	suffix := []byte("\nallocate:\n  trunk: refs/remotes/origin/typo\n")
	updated := make([]byte, 0, len(existing)+len(suffix))
	updated = append(updated, existing...)
	updated = append(updated, suffix...)
	if writeErr := os.WriteFile(yamlPath, updated, 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}
	if out, gitErr := runGit(root, "remote", "add", "origin", "https://example.invalid/x.git"); gitErr != nil {
		t.Fatalf("git remote add: %v\n%s", gitErr, out)
	}

	out, err := runBin(t, root, binDir, nil, "add", "gap", "--title", "Should fail")
	if err == nil {
		t.Fatalf("expected aiwf add to fail when trunk ref is missing; output:\n%s", out)
	}
	if !strings.Contains(out, "refs/remotes/origin/typo") {
		t.Errorf("error output should name the missing trunk ref; got:\n%s", out)
	}
}

// runHook executes the installed pre-push hook script directly. We
// invoke via /bin/sh to honor the shebang on platforms where the
// file mode might not survive (it does on macOS/Linux test runners,
// but defensive is fine). When extraPath is empty, the host's
// existing PATH is used unchanged — the hook should not depend on
// it because `aiwf init` bakes the absolute binary path into the
// hook script.
func runHook(t *testing.T, root, extraPath string) (string, error) {
	t.Helper()
	cmd := exec.Command("/bin/sh", filepath.Join(root, ".git", "hooks", "pre-push"))
	cmd.Dir = root
	envPath := os.Getenv("PATH")
	if extraPath != "" {
		envPath = extraPath + string(os.PathListSeparator) + envPath
	}
	cmd.Env = append(os.Environ(), "PATH="+envPath)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
