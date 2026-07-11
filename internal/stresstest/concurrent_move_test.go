package stresstest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// concurrent_move_test.go — real-subprocess coverage for
// ConcurrentMoveScenario (M-0250/AC-4). The pure decision logic
// (classifyConcurrentMove) is pinned exhaustively in
// concurrent_move_classify_test.go against fabricated outcomes; these
// tests confirm real, concurrently-launched `aiwf move` subprocesses
// racing repolock actually all land under the target epic with
// exactly one commit each, repeated via M-0240's RunRepeated for
// statistical coverage — mirroring
// concurrent_id_allocation_test.go's own shape.

// TestConcurrentMoveScenario_RealBinary_ErrorsWhenBinaryMissing runs a
// real Setup (so the repo has real commits and gitHeadCommitCount's
// own "before" call succeeds), then points a fresh scenario carrying
// Setup's own output at a nonexistent binary path for Run — every
// launched actor's subprocess then fails at the OS level, surfacing
// from the per-actor result-processing loop after the (successful)
// concurrent fan-out.
func TestConcurrentMoveScenario_RealBinary_ErrorsWhenBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)

	setup := NewConcurrentMoveScenario(bin, 3, 1)
	if err := setup.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	broken := &ConcurrentMoveScenario{
		aiwfBin:      filepath.Join(t.TempDir(), "no-such-aiwf-binary"),
		n:            setup.n,
		milestoneIDs: setup.milestoneIDs,
		targetEpic:   setup.targetEpic,
	}
	if err := broken.Run(dir); err == nil {
		t.Fatal("expected Run to error when the aiwf binary path doesn't exist")
	} else if !strings.Contains(err.Error(), "running aiwf move") {
		t.Fatalf("expected the launch failure to name the move call, got: %v", err)
	}
}

// TestConcurrentMoveScenario_RealBinary_ErrorsWhenSetupBinaryMissing
// points Setup itself at a nonexistent binary path, distinct from
// Run's own binary-missing test above — Setup's first subprocess
// launch (seeding the source epic) fails before Run is ever called.
func TestConcurrentMoveScenario_RealBinary_ErrorsWhenSetupBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	dir := newVerbSequenceTestRepo(t)

	s := NewConcurrentMoveScenario(filepath.Join(t.TempDir(), "no-such-aiwf-binary"), 3, 1)
	if err := s.Setup(dir); err == nil {
		t.Fatal("expected Setup to error when the aiwf binary path doesn't exist")
	} else if !strings.Contains(err.Error(), "seeding the source epic") {
		t.Fatalf("expected the launch failure to name the source-epic seeding step, got: %v", err)
	}
}

// TestConcurrentMoveScenario_RealBinary_NConcurrentActorsAllLandUnderTheTargetEpic
// is the AC-4 scenario itself: n real `aiwf move` subprocesses,
// launched close together via goroutines racing real OS process
// scheduling (no artificial delay), each relocating a distinct
// milestone from one shared source epic to one shared target epic.
func TestConcurrentMoveScenario_RealBinary_NConcurrentActorsAllLandUnderTheTargetEpic(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	base := t.TempDir()

	const n = 8
	newScenario := func(seed int64) Scenario {
		return NewConcurrentMoveScenario(bin, n, seed)
	}

	rw := newReportWriter(&countingWriter{})
	results, err := RunRepeated(newScenario, base, 3, seedSequence(1, 2, 3), rw, "", nil)
	if err != nil {
		t.Fatalf("RunRepeated: %v", err)
	}
	for i, r := range results {
		if !r.Passed {
			t.Fatalf("attempt %d found violations (dir preserved at %s):\n%+v", i, r.Dir, r.Violations)
		}
	}
}

// TestConcurrentMoveScenario_RealBinary_DetectsAGenuineDivergence
// points Run at a stand-in "aiwf" that falsely reports every `move`
// as ok without actually moving anything (delegating every other
// subcommand, including `show`, to the real binary) — a genuine,
// real-subprocess-observable divergence: each milestone's real parent
// stays the source epic while the fake move claims success. Closes
// the same class of vacuity gap
// TestCheckListInvariant_RealBinary_DetectsAGenuineDivergence closes
// for AC-3: the "all succeed" test alone can't tell a correctly-wired
// Run from one that silently drops classifyConcurrentMove's result,
// since a healthy repo produces zero violations either way.
func TestConcurrentMoveScenario_RealBinary_DetectsAGenuineDivergence(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	realBin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)

	setup := NewConcurrentMoveScenario(realBin, 3, 1)
	if err := setup.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	fakeBin := writeFakeAiwfMove(t, realBin)
	broken := &ConcurrentMoveScenario{
		aiwfBin:      fakeBin,
		n:            setup.n,
		milestoneIDs: setup.milestoneIDs,
		targetEpic:   setup.targetEpic,
	}
	if err := broken.Run(dir); err != nil {
		t.Fatalf("Run: %v", err)
	}
	violations := broken.Verify(dir)
	if len(violations) == 0 {
		t.Fatal("expected violations from a move that falsely reports ok without moving anything, got none")
	}
}

// writeFakeAiwfMove writes an executable shell script standing in for
// `aiwf`: it falsely reports "move" as ok without executing it, and
// delegates every other subcommand to realBin — so a `show` call
// still reads the real, unchanged on-disk state.
func writeFakeAiwfMove(t *testing.T, realBin string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "aiwf")
	script := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "move" ]; then
  echo '{"status":"ok","findings":[],"result":{},"metadata":{}}'
  exit 0
fi
exec %q "$@"
`, realBin)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil { //nolint:gosec // deliberately executable; a test-local stand-in binary, not attacker-controlled input
		t.Fatalf("writing fake aiwf binary: %v", err)
	}
	return path
}
