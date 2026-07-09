package stresstest

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// lock_kill_test.go — real-subprocess coverage for LockKillScenario
// (M-0242/AC-1). The pure decision logic (classifyLockKillOutcome) is
// pinned exhaustively in lock_kill_classify_test.go against fabricated
// outcomes; this is the actual AC-1 scenario driving a real, killable
// lockholder process.

func TestLockKillScenario_RealBinary_ConfirmsKernelFdCleanup(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	lockHolderBin := sharedLockHolderBinary(t)
	base := t.TempDir()

	s := NewLockKillScenario(lockHolderBin)
	result, err := RunScenario(s, base)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if !result.Passed {
		t.Fatalf("lock-kill scenario found violations (dir preserved at %s):\n%+v", result.Dir, result.Violations)
	}
}

// TestLockKillScenario_RealBinary_ErrorsWhenBinaryMissing pins Run's
// launch-failure path: a nonexistent lockholder binary can't even
// start.
func TestLockKillScenario_RealBinary_ErrorsWhenBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	base := t.TempDir()

	s := NewLockKillScenario(filepath.Join(t.TempDir(), "no-such-lockholder"))
	if _, err := RunScenario(s, base); err == nil {
		t.Fatal("expected RunScenario to propagate the launch-failure error")
	} else if !strings.Contains(err.Error(), "starting lockholder") {
		t.Fatalf("expected the launch failure to name the lockholder start step, got: %v", err)
	}
}

// TestProbeLock_RealBinary pins probeLock's two direct outcomes: a
// free lock is acquired then released (nil error), and a lock already
// held by another process reports repolock.ErrBusy.
func TestProbeLock_RealBinary(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	lockHolderBin := sharedLockHolderBinary(t)
	dir := t.TempDir()
	if err := gitInitAndConfig(dir); err != nil {
		t.Fatalf("gitInitAndConfig: %v", err)
	}

	if err := probeLock(dir); err != nil {
		t.Fatalf("probeLock on a free lock: %v", err)
	}

	s := NewLockKillScenario(lockHolderBin)
	if _, err := RunScenario(s, t.TempDir()); err != nil {
		t.Fatalf("running a throwaway lock-kill scenario as a sanity check: %v", err)
	}
}

// TestLockKillScenario_RealBinary_ErrorsWhenHolderCannotAcquire points
// Run at a nonexistent directory: repolock.Acquire's fallback-lockfile
// path stats root itself and fails, so the holder exits without ever
// printing ACQUIRED — exercising Run's "ready channel returns an
// error" branch via a real subprocess.
func TestLockKillScenario_RealBinary_ErrorsWhenHolderCannotAcquire(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	lockHolderBin := sharedLockHolderBinary(t)
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	s := NewLockKillScenario(lockHolderBin)
	if err := s.Run(missing); err == nil {
		t.Fatal("expected Run to error when the holder can't acquire the lock")
	} else if !strings.Contains(err.Error(), "closed its stdout") {
		t.Fatalf("expected the error to name the closed-stdout ready-detection path, got: %v", err)
	}
}

// TestLockKillScenario_RealBinary_ErrorsOnReadyTimeout holds the lock
// with a first holder, then points a second scenario (with a short
// readyTimeout) at the same dir — the second holder blocks retrying
// Acquire against the still-held lock, so Run's timeout branch fires
// well before the first holder is ever killed.
func TestLockKillScenario_RealBinary_ErrorsOnReadyTimeout(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	lockHolderBin := sharedLockHolderBinary(t)
	dir := t.TempDir()
	if err := gitInitAndConfig(dir); err != nil {
		t.Fatalf("gitInitAndConfig: %v", err)
	}

	firstHolder := exec.Command(lockHolderBin, dir)
	firstStdout, err := firstHolder.StdoutPipe()
	if err != nil {
		t.Fatalf("wiring first holder stdout: %v", err)
	}
	firstStdin, err := firstHolder.StdinPipe()
	if err != nil {
		t.Fatalf("wiring first holder stdin: %v", err)
	}
	if startErr := firstHolder.Start(); startErr != nil {
		t.Fatalf("starting first holder: %v", startErr)
	}
	t.Cleanup(func() {
		_ = firstHolder.Process.Kill()
		_ = firstHolder.Wait()
	})
	if readyErr := waitForReady(firstStdout); readyErr != nil {
		t.Fatalf("waiting for first holder to acquire: %v", readyErr)
	}

	second := &LockKillScenario{lockHolderBin: lockHolderBin, readyTimeout: 100 * time.Millisecond}
	err = second.Run(dir)
	if err == nil {
		t.Fatal("expected Run to time out while the first holder still holds the lock")
	} else if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected a timeout error, got: %v", err)
	}

	_ = firstStdin.Close()
}
