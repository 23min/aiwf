package stresstest

import (
	"errors"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/repolock"
)

// lock_kill_classify_test.go pins classifyLockKillOutcome — the pure
// decision logic behind LockKillScenario (M-0242/AC-1) — against
// fabricated outcomes, so every branch is exercised deterministically
// rather than depending on a real kill's exact timing.

func TestClassifyLockKillOutcome(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		outcome        lockKillOutcome
		wantViolations int
	}{
		{
			name: "clean run: held while running, killed by signal, immediately re-acquirable",
			outcome: lockKillOutcome{
				probeBeforeKillErr: repolock.ErrBusy,
				holderSignaled:     true,
				probeAfterKillErr:  nil,
			},
			wantViolations: 0,
		},
		{
			name: "probe before kill did not see the lock held — a violation",
			outcome: lockKillOutcome{
				probeBeforeKillErr: nil,
				holderSignaled:     true,
				probeAfterKillErr:  nil,
			},
			wantViolations: 1,
		},
		{
			name: "probe before kill saw a different error, not ErrBusy — a violation",
			outcome: lockKillOutcome{
				probeBeforeKillErr: errors.New("some other error"), //enums:ignore deliberately fabricated non-repolock error for the test
				holderSignaled:     true,
				probeAfterKillErr:  nil,
			},
			wantViolations: 1,
		},
		{
			name: "holder did not terminate by signal — a violation (the kill may not have landed)",
			outcome: lockKillOutcome{
				probeBeforeKillErr: repolock.ErrBusy,
				holderSignaled:     false,
				probeAfterKillErr:  nil,
			},
			wantViolations: 1,
		},
		{
			name: "probe after kill still sees the lock held — a violation (kernel didn't release the fd)",
			outcome: lockKillOutcome{
				probeBeforeKillErr: repolock.ErrBusy,
				holderSignaled:     true,
				probeAfterKillErr:  repolock.ErrBusy,
			},
			wantViolations: 1,
		},
		{
			name: "probe after kill sees a different error — still a violation",
			outcome: lockKillOutcome{
				probeBeforeKillErr: repolock.ErrBusy,
				holderSignaled:     true,
				probeAfterKillErr:  errors.New("some other error"), //enums:ignore deliberately fabricated non-repolock error for the test
			},
			wantViolations: 1,
		},
		{
			name: "every check fails at once — three violations, not short-circuited to one",
			outcome: lockKillOutcome{
				probeBeforeKillErr: nil,
				holderSignaled:     false,
				probeAfterKillErr:  repolock.ErrBusy,
			},
			wantViolations: 3,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyLockKillOutcome(tc.outcome)
			if len(got) != tc.wantViolations {
				t.Errorf("violations = %d (%+v), want %d", len(got), got, tc.wantViolations)
			}
		})
	}
}

// errReader is an io.Reader that always fails with a fixed error —
// used to drive bufio.Scanner into its Err() branch, which a real
// subprocess's stdout pipe has no practical way to trigger on demand.
type errReader struct{ err error }

func (r errReader) Read(_ []byte) (int, error) { return 0, r.err }

func TestWaitForReady(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		r       io.Reader
		wantErr string
	}{
		{
			name: "first line is ACQUIRED",
			r:    strings.NewReader("ACQUIRED\n"),
		},
		{
			name:    "first line is something else",
			r:       strings.NewReader("not-acquired\nACQUIRED\n"),
			wantErr: `"not-acquired"`,
		},
		{
			name:    "reader errors before any line completes",
			r:       errReader{err: errors.New("boom")}, //enums:ignore deliberately fabricated non-repolock error for the test
			wantErr: "boom",
		},
		{
			name:    "reader closes with no output at all",
			r:       strings.NewReader(""),
			wantErr: "closed its stdout",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := waitForReady(tc.r)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("waitForReady: %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("waitForReady = %v, want an error containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestProcessWasSignaled(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("signal-based process termination is unix-only")
	}

	t.Run("nil error", func(t *testing.T) {
		t.Parallel()
		if processWasSignaled(nil) {
			t.Fatal("expected false for a nil wait error")
		}
	})

	t.Run("non-ExitError", func(t *testing.T) {
		t.Parallel()
		if processWasSignaled(errors.New("not an exec.ExitError")) { //enums:ignore deliberately fabricated non-repolock error for the test
			t.Fatal("expected false for an error that isn't *exec.ExitError")
		}
	})

	t.Run("clean nonzero exit is not signaled", func(t *testing.T) {
		t.Parallel()
		err := exec.Command("sh", "-c", "exit 3").Run()
		if !processWasSignaledMatchesExpectation(t, err, false) {
			return
		}
	})

	t.Run("SIGKILL is signaled", func(t *testing.T) {
		t.Parallel()
		cmd := exec.Command("sleep", "5")
		if err := cmd.Start(); err != nil {
			t.Fatalf("starting sleep: %v", err)
		}
		if err := cmd.Process.Kill(); err != nil {
			t.Fatalf("killing sleep: %v", err)
		}
		err := cmd.Wait()
		if !processWasSignaledMatchesExpectation(t, err, true) {
			return
		}
	})
}

// processWasSignaledMatchesExpectation asserts waitErr is a real
// *exec.ExitError (sanity-checking the test fixture itself) and that
// processWasSignaled(waitErr) matches want.
func processWasSignaledMatchesExpectation(t *testing.T, waitErr error, want bool) bool {
	t.Helper()
	var exitErr *exec.ExitError
	if !errors.As(waitErr, &exitErr) {
		t.Fatalf("test fixture did not produce an *exec.ExitError, got: %v (%T)", waitErr, waitErr)
		return false
	}
	if got := processWasSignaled(waitErr); got != want {
		t.Errorf("processWasSignaled(%v) = %v, want %v", waitErr, got, want)
	}
	return true
}
