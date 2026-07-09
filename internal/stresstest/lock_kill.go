package stresstest

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/23min/aiwf/internal/repolock"
)

// lock_kill.go — M-0242/AC-1: LockKillScenario drives a real,
// independently killable OS process (the lockholder helper binary)
// that acquires internal/repolock's exclusive lock on a disposable
// repo, confirms the lock is externally observable as held via
// repolock's own zero-timeout non-blocking probe mode
// (Acquire(dir, 0), which already exists — no code change to
// internal/repolock, per M-0242/AC-3), SIGKILLs the holder, and
// confirms a subsequent probe re-acquires immediately — the kernel
// fd-cleanup-on-exit behavior repolock's own doc comment claims.

// LockKillScenario implements Scenario.
type LockKillScenario struct {
	lockHolderBin string
	// readyTimeout bounds how long Run waits for the holder to report
	// ACQUIRED. Defaulted by the constructor; tests in this package
	// may set it directly (same-package struct literal) to force the
	// timeout branch deterministically and quickly.
	readyTimeout time.Duration
	violations   []Violation
}

// defaultReadyTimeout is generous enough that a healthy holder always
// reports ACQUIRED well within it on any machine this runs on.
const defaultReadyTimeout = 5 * time.Second

// NewLockKillScenario builds a scenario driving lockHolderBin (the
// built internal/stresstest/lockholder binary) as the lock-holding
// process to kill.
func NewLockKillScenario(lockHolderBin string) *LockKillScenario {
	return &LockKillScenario{lockHolderBin: lockHolderBin, readyTimeout: defaultReadyTimeout}
}

// Setup git-inits dir so the lockfile resolves to the real
// <dir>/.git/aiwf.lock path repolock documents, matching what a real
// aiwf invocation would use.
func (s *LockKillScenario) Setup(dir string) error {
	return gitInitAndConfig(dir)
}

// Run launches the lockholder against dir, waits for its readiness
// signal, probes the lock state before and after killing it, and
// records classifyLockKillOutcome's verdict.
func (s *LockKillScenario) Run(dir string) error {
	holder := exec.Command(s.lockHolderBin, dir) //nolint:gosec // lockHolderBin is a path this package's own BuildLockHolder just produced, not attacker-controlled input
	stdout, err := holder.StdoutPipe()
	if err != nil { //coverage:ignore defensive: StdoutPipe on a freshly constructed, not-yet-started exec.Cmd has no realistic failure mode
		return fmt.Errorf("wiring lockholder stdout: %w", err)
	}
	// Without an explicit Stdin, exec.Cmd reads from /dev/null, which
	// returns EOF immediately — the holder would read that EOF and
	// exit cleanly right after printing "ACQUIRED", never actually
	// blocking long enough to be killed. Wiring a real pipe and never
	// writing to (or closing) it keeps the holder genuinely blocked
	// until this scenario kills it.
	stdinW, err := holder.StdinPipe()
	if err != nil { //coverage:ignore defensive: StdinPipe on a freshly constructed, not-yet-started exec.Cmd has no realistic failure mode
		return fmt.Errorf("wiring lockholder stdin: %w", err)
	}
	defer func() { _ = stdinW.Close() }()
	if startErr := holder.Start(); startErr != nil {
		return fmt.Errorf("starting lockholder: %w", startErr)
	}

	ready := make(chan error, 1)
	go func() { ready <- waitForReady(stdout) }()

	select {
	case err := <-ready:
		if err != nil {
			_ = holder.Process.Kill()
			_ = holder.Wait()
			return err
		}
	case <-time.After(s.readyTimeout):
		_ = holder.Process.Kill()
		_ = holder.Wait()
		return errors.New("timed out waiting for lockholder to report ACQUIRED")
	}

	probeBeforeKillErr := probeLock(dir)

	if killErr := holder.Process.Kill(); killErr != nil { //coverage:ignore defensive: killing the process this scenario itself just started, and which is confirmed alive and blocked in a stdin read, has no realistic failure mode on the unix platforms this package targets
		return fmt.Errorf("killing lockholder: %w", killErr)
	}
	waitErr := holder.Wait()

	outcome := lockKillOutcome{
		probeBeforeKillErr: probeBeforeKillErr,
		holderSignaled:     processWasSignaled(waitErr),
		probeAfterKillErr:  probeLock(dir),
	}
	s.violations = classifyLockKillOutcome(outcome)
	return nil
}

// Verify returns the violations Run collected.
func (s *LockKillScenario) Verify(_ string) []Violation {
	return s.violations
}

// waitForReady blocks until r's first line is exactly "ACQUIRED" (the
// lockholder's readiness signal), or returns an error describing why
// it never confirmed. Split out of Run so the malformed-output and
// scan-error paths are directly unit-testable without a real
// subprocess.
func waitForReady(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	if scanner.Scan() {
		if scanner.Text() == "ACQUIRED" {
			return nil
		}
		return fmt.Errorf("lockholder's first line was %q, want %q", scanner.Text(), "ACQUIRED")
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return fmt.Errorf("reading lockholder's readiness signal: %w", scanErr)
	}
	return errors.New("lockholder closed its stdout before reporting ACQUIRED")
}

// probeLock attempts a non-blocking Acquire against dir — the same
// zero-timeout probe mode internal/repolock already ships (Acquire's
// own doc comment: "A zero timeout returns ErrBusy immediately if the
// lock is held"). A successful acquire is released immediately since
// this call is observational only, never meant to hold the lock
// itself.
func probeLock(dir string) error {
	lock, err := repolock.Acquire(dir, 0)
	if err != nil {
		return err
	}
	return lock.Release()
}

// lockKillOutcome is the raw evidence classifyLockKillOutcome judges.
type lockKillOutcome struct {
	probeBeforeKillErr error
	holderSignaled     bool
	probeAfterKillErr  error
}

// classifyLockKillOutcome judges one lock-kill attempt's outcome.
// Every check runs independently — a broken run can fail more than
// one at once, and each is reported rather than short-circuited.
func classifyLockKillOutcome(o lockKillOutcome) []Violation {
	var violations []Violation
	if !errors.Is(o.probeBeforeKillErr, repolock.ErrBusy) {
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"expected the lock to be held (repolock.ErrBusy) while the holder was running, got: %v", o.probeBeforeKillErr)})
	}
	if !o.holderSignaled {
		violations = append(violations, Violation{Message: "the lock-holder process did not appear to terminate by signal (SIGKILL) — the kill may not have landed"})
	}
	if o.probeAfterKillErr != nil {
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"expected an immediate re-acquire to succeed after killing the lock holder (proving kernel fd cleanup), got: %v", o.probeAfterKillErr)})
	}
	return violations
}
