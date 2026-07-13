package doctor_test

import (
	"fmt"
	"testing"

	climod "github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/doctor"
)

// M-0255/AC-1 backfill: doctor.Run(selfCheck=true) drives runSelfCheck,
// which carries the largest concentration of flagged sites in this
// milestone's scope. All five tests below are serial (no t.Parallel):
// each either mutates the package-level doctor.Dispatcher var directly,
// or relies on runSelfCheck's own internal os.Setenv/os.Unsetenv of
// HOME/PATH/GOPROXY around its execution — both are process-global
// state that would race against a sibling test running in parallel.
//
// Every test that touches doctor.Dispatcher saves and restores it via
// defer, regardless of whether it expects a prior value, so no test's
// outcome depends on file or test execution order within the binary.
//
// The two os.MkdirTemp-second-call guards (fakeHome, and the
// .gitconfig write into it) and setLocalGitIdentity's error guard are
// `//coverage:ignore`d in selfcheck.go itself: each is unreachable
// independent of an earlier guard in the same function that would fire
// first. The three step-loop setup/verify/verifyOutput FAIL branches
// are also `//coverage:ignore`d there — reaching them needs corrupting
// on-disk state a real *earlier* step in the same 29-step sequence
// produced, not a direct trigger.

// TestRun_SelfCheckDispatcherUnset covers runSelfCheck's own top guard:
// a nil Dispatcher (the wiring bug the comment at dispatcher.go
// describes) refuses immediately with ExitInternal.
func TestRun_SelfCheckDispatcherUnset(t *testing.T) {
	orig := doctor.Dispatcher
	defer func() { doctor.Dispatcher = orig }()
	doctor.Dispatcher = nil

	rc := doctor.Run("", true, false)
	if rc != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", rc)
	}
}

// TestRun_SelfCheckTempDirCreationFailure covers the first
// os.MkdirTemp guard: a TMPDIR pointing at a non-existent directory
// makes os.MkdirTemp("", ...) fail deterministically.
func TestRun_SelfCheckTempDirCreationFailure(t *testing.T) {
	orig := doctor.Dispatcher
	defer func() { doctor.Dispatcher = orig }()
	doctor.Dispatcher = func([]string) int { return cliutil.ExitOK }

	t.Setenv("TMPDIR", "/nonexistent-aiwf-self-check-tmpdir")

	rc := doctor.Run("", true, false)
	if rc != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", rc)
	}
}

// TestRun_SelfCheckGitInitFailure covers the gitops.Init guard: a PATH
// with no `git` executable on it makes the underlying `git init -q -b
// main` subprocess fail to even start.
func TestRun_SelfCheckGitInitFailure(t *testing.T) {
	orig := doctor.Dispatcher
	defer func() { doctor.Dispatcher = orig }()
	doctor.Dispatcher = func([]string) int { return cliutil.ExitOK }

	t.Setenv("PATH", t.TempDir())

	rc := doctor.Run("", true, false)
	if rc != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", rc)
	}
}

// TestRun_SelfCheckStepRCMismatch covers the step-loop's rc-mismatch
// FAIL branch, including its nested `captured != ""` arm (the fake
// dispatcher's second call writes to stdout before returning a bad
// rc, so runCaptured's captured buffer is non-empty), and via the
// first step succeeding first, the loop's per-step "ok" print. Runs
// against the real environment (real git, real tempdirs) so
// gitops.Init/setLocalGitIdentity succeed for real and execution
// actually reaches the step loop.
func TestRun_SelfCheckStepRCMismatch(t *testing.T) {
	orig := doctor.Dispatcher
	defer func() { doctor.Dispatcher = orig }()
	calls := 0
	doctor.Dispatcher = func([]string) int {
		calls++
		if calls == 1 {
			return cliutil.ExitOK
		}
		fmt.Println("synthetic verb output for the failing step")
		return cliutil.ExitUsage
	}

	rc := doctor.Run("", true, false)
	if rc != cliutil.ExitFindings {
		t.Errorf("rc = %d, want ExitFindings", rc)
	}
	if calls < 2 {
		t.Errorf("calls = %d, want at least 2 (first succeeds, second mismatches)", calls)
	}
}

// TestRun_SelfCheckEndToEnd drives the real, full self-check sequence
// — the same one `make selfcheck` / CI runs as a subprocess — entirely
// in-process. Importing internal/cli wires doctor.Dispatcher to the
// real Execute via that package's own init(); this test additionally
// sets it explicitly so its outcome never depends on import-init
// ordering relative to the other tests in this binary, which override
// doctor.Dispatcher themselves.
func TestRun_SelfCheckEndToEnd(t *testing.T) {
	orig := doctor.Dispatcher
	defer func() { doctor.Dispatcher = orig }()
	doctor.Dispatcher = climod.Execute

	rc := doctor.Run("", true, false)
	if rc != cliutil.ExitOK {
		t.Errorf("rc = %d, want ExitOK (self-check should pass end to end)", rc)
	}
}
