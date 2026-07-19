package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/doctor"
)

// doctorReport runs doctor.DoctorReport and collapses its []Problem
// return to the count of error-severity problems — the exit-relevant
// count these tests assert against (warnings never gate doctor's exit,
// so an advisory-only report still counts as zero problems here).
func doctorReport(root string, opts doctor.DoctorOptions) (lines []string, errs int) {
	var problems []doctor.Problem
	lines, problems = doctor.DoctorReport(root, opts)
	for i := range problems {
		if problems[i].Severity == doctor.SeverityError {
			errs++
		}
	}
	return lines, errs
}

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
	// Initialize git repo so the verb can commit. --initial-branch=main
	// matches aiwf's own unconfigured trunk-name default
	// (cliutil.ConfiguredTrunkBranchShortName) — plain `git init` on
	// some git versions/configs defaults to "master" instead, which
	// would spuriously trip the G-0269 activating-promote branch guard
	// for every test that promotes an epic to active.
	if err := osExec(t, root, "git", "init", "-q", "--initial-branch=main"); err != nil {
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

// acBodyFixturePath writes a scratch AC-body file into root (confined
// to the test's own tempdir-backed repo) and returns its path, for
// `aiwf add ac --body-file`. M-0268/AC-2: draft -> in_progress now
// refuses an AC with an empty body, so fixtures that later promote a
// milestone to in_progress need their seeded ACs to carry real prose,
// not just a title.
func acBodyFixturePath(t *testing.T, root string) string {
	t.Helper()
	path := filepath.Join(root, ".test-ac-body.md")
	if err := os.WriteFile(path, []byte("Fixture prose for test setup; not the subject under test.\n"), 0o644); err != nil {
		t.Fatalf("writing AC body fixture: %v", err)
	}
	return path
}
