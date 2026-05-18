package integration

import (
	"os"
	"os/exec"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

// setupCLITestRepo gives the test process a git identity and an
// initialized repo; returns the repo root.
//
// Hook discipline: every test calling `aiwf init` via this in-process
// dispatcher must pass `--skip-hook` unless it specifically wants to
// verify hook installation. The hook bakes in `os.Executable()`,
// which under `go test` resolves to the test binary — letting git
// then exec the test binary as a hook can hang or behave
// unpredictably. Tests that need consumer-parity hook firing should
// use the runBin-style subprocess pattern (see cmd/aiwf's binary
// integration tests) where a real aiwf binary is built and driven
// as a child process.
func setupCLITestRepo(t *testing.T) string {
	t.Helper()
	// GIT identity is seeded once by TestMain (setup_test.go) via
	// os.Setenv — t.Setenv would panic under t.Parallel.
	root := t.TempDir()
	if got := cli.Execute([]string{"check", "--root=" + root}); got != cliutil.ExitOK {
		t.Fatalf("baseline check on tmpdir = %d", got)
	}
	// Initialize git repo so the verb can commit.
	if err := osExec(t, root, "git", "init", "-q"); err != nil {
		t.Fatalf("git init: %v", err)
	}
	return root
}

// osExec runs a command in workdir. Returns the error if any. Output
// is logged via t.Logf on failure so a flaky git call leaves
// breadcrumbs.
func osExec(t *testing.T, workdir, name string, args ...string) error {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("%s output: %s", name, out)
	}
	return err
}

// mustRun invokes cli.Execute with args; non-zero exit fails the
// test with the args and rc. Used by tests that exercise a verb
// chain end-to-end and don't care to assert the exit codes
// individually.
func mustRun(t *testing.T, args ...string) {
	t.Helper()
	if rc := cli.Execute(args); rc != cliutil.ExitOK {
		t.Fatalf("aiwf %v: rc=%d", args, rc)
	}
}
