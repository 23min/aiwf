package stresstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// concurrent_writer_at_scale_test.go — real-subprocess coverage for
// ConcurrentWriterAtScaleScenario (M-0244/AC-1). The pure decision
// logic (classifyConcurrentWriterAtScale) is pinned exhaustively in
// concurrent_writer_at_scale_classify_test.go against fabricated
// data; this is the actual scenario, driving n real, concurrently
// launched `aiwf cancel` subprocesses that all append their
// diagnostic log line to one shared file.

func TestConcurrentWriterAtScaleScenario_RealBinary_ErrorsWhenBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	dir := t.TempDir()

	s := NewConcurrentWriterAtScaleScenario(filepath.Join(t.TempDir(), "no-such-aiwf-binary"), 3, 1)
	if err := s.Setup(dir); err == nil {
		t.Fatal("expected Setup to error when the aiwf binary path doesn't exist")
	} else if !strings.Contains(err.Error(), "seeding gap") {
		t.Fatalf("expected the failure to name the seeding step, got: %v", err)
	}
}

// TestConcurrentWriterAtScaleScenario_RealBinary_RunErrorsWhenActorLaunchFails
// seeds real gaps with a working binary, then swaps in a broken binary
// path before Run — pinning Run's own per-actor launch-failure branch
// directly, since Run (unlike Setup) never even attempts a launch on a
// bad binary without gapIDs already populated.
func TestConcurrentWriterAtScaleScenario_RealBinary_RunErrorsWhenActorLaunchFails(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := t.TempDir()

	s := NewConcurrentWriterAtScaleScenario(bin, 2, 1)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	s.aiwfBin = filepath.Join(t.TempDir(), "no-such-aiwf-binary")

	if err := s.Run(dir); err == nil {
		t.Fatal("expected Run to error when the aiwf binary path doesn't exist")
	} else if !strings.Contains(err.Error(), "actor") {
		t.Fatalf("expected the failure to name the actor, got: %v", err)
	}
}

// TestConcurrentWriterAtScaleScenario_RealBinary_NConcurrentWritersNeverTearOrInterleave
// is the AC-1 scenario itself: n real `aiwf cancel` subprocesses,
// launched close together via goroutines racing real OS process
// scheduling, all appending to one shared diagnostic log file.
func TestConcurrentWriterAtScaleScenario_RealBinary_NConcurrentWritersNeverTearOrInterleave(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	base := t.TempDir()

	const n = 12
	s := NewConcurrentWriterAtScaleScenario(bin, n, 1)
	result, err := RunScenario(s, base)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if !result.Passed {
		t.Fatalf("concurrent-writer-at-scale scenario found violations (dir preserved at %s):\n%+v", result.Dir, result.Violations)
	}
}

// TestConcurrentWriterAtScaleScenario_RealBinary_LogFileHasExactlyNLines
// re-drives the scenario directly (bypassing RunScenario's cleanup) so
// the shared log file can be inspected afterward: confirms every line
// is valid JSON and there are exactly n of them, the concrete claim
// AC-1's own acceptance text makes ("every resulting line parses
// cleanly... none is interleaved or truncated").
func TestConcurrentWriterAtScaleScenario_RealBinary_LogFileHasExactlyNLines(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := t.TempDir()

	const n = 10
	s := NewConcurrentWriterAtScaleScenario(bin, n, 1)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if err := s.Run(dir); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if violations := s.Verify(dir); len(violations) != 0 {
		t.Fatalf("unexpected violations: %+v", violations)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "diag.log"))
	if err != nil {
		t.Fatalf("reading shared diagnostic log: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != n {
		t.Fatalf("got %d log lines, want %d", len(lines), n)
	}
}
