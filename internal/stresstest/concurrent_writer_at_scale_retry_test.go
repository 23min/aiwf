//go:build stress

package stresstest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// concurrent_writer_at_scale_retry_test.go — G-0424. Pins the lock-busy
// retry logic ConcurrentWriterAtScaleScenario uses so a concurrent
// `aiwf cancel` that loses the repo-lock race (repolock.ErrBusy →
// ExitUsage) is retried to completion instead of aborting the whole run.
// This file carries the real-binary seam, which holds the repo lock from
// an independent process and so owns its own runner; the pure decision
// helpers it exercises are unit-tested hermetically (and untagged) in
// concurrent_writer_at_scale_retry_hermetic_test.go.

// TestConcurrentWriterAtScaleScenario_RealBinary_RunRetriesPastLockBusy is
// the real-binary seam for G-0424: an independently-held repo lock forces
// the actor's first `aiwf cancel` attempt to lose the race (exit 2 /
// repolock.ErrBusy); the scenario must retry past it, complete the cancel,
// and account for every attempt's diagnostic line (busy retries included)
// so both the classifier and the exact-line-count invariant stay clean.
func TestConcurrentWriterAtScaleScenario_RealBinary_RunRetriesPastLockBusy(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	holderBin := sharedLockHolderBinary(t)
	dir := t.TempDir()

	s := NewConcurrentWriterAtScaleScenario(bin, 1, 1)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	logPath := filepath.Join(dir, "diag.log")

	// Hold the repo lock from an independent, killable process so the
	// actor's first cancel attempt polls for the full lock timeout and then
	// exits busy. Stdin is wired and never written to (mirroring
	// LockKillScenario) so the holder blocks instead of reading EOF and
	// exiting immediately.
	holder := exec.Command(holderBin, dir) //nolint:gosec // holderBin is a path this package's own BuildLockHolder just produced, not attacker-controlled input
	stdout, err := holder.StdoutPipe()
	if err != nil {
		t.Fatalf("wiring holder stdout: %v", err)
	}
	stdinW, err := holder.StdinPipe()
	if err != nil {
		t.Fatalf("wiring holder stdin: %v", err)
	}
	defer func() { _ = stdinW.Close() }()
	if startErr := holder.Start(); startErr != nil {
		t.Fatalf("starting holder: %v", startErr)
	}
	if readyErr := waitForReady(stdout); readyErr != nil {
		_ = holder.Process.Kill()
		_ = holder.Wait()
		t.Fatalf("holder never reported ACQUIRED: %v", readyErr)
	}

	// Release the lock only once the actor's first attempt has actually gone
	// busy — detected by its verb.failed line landing in the shared log —
	// rather than after a fixed sleep a slow subprocess start could race
	// (the very load condition this fix targets). Guaranteeing a real busy
	// loss before release makes >=1 retry deterministic under any load.
	go func() {
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			raw, _ := os.ReadFile(logPath)
			if strings.Contains(string(raw), `"msg":"verb.failed"`) {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		_ = holder.Process.Kill()
		_ = holder.Wait()
	}()

	if runErr := s.Run(dir); runErr != nil {
		t.Fatalf("Run errored instead of retrying past the busy lock: %v", runErr)
	}
	if v := s.Verify(dir); len(v) != 0 {
		t.Fatalf("unexpected violations after retrying past lock-busy: %+v", v)
	}
	if len(s.wantRunIDs) < 2 {
		t.Fatalf("expected >=2 diagnostic invocations (>=1 busy retry + the success), got %d", len(s.wantRunIDs))
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading shared diagnostic log: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != len(s.wantRunIDs) {
		t.Fatalf("log has %d lines, want %d (one per real invocation, busy retries included)", len(lines), len(s.wantRunIDs))
	}
}
