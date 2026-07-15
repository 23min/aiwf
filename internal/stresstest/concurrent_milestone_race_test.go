package stresstest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// concurrent_milestone_race_test.go — real-subprocess coverage for
// ConcurrentMilestoneRaceScenario (M-0258/AC-1). The pure decision
// logic (raceActorArgs, buildRaceOutcome, and the shared
// classifyAgainstBaseline this scenario's own baseline parameterizes)
// is pinned exhaustively in concurrent_milestone_race_classify_test.go
// against fabricated inputs; these tests confirm real, concurrently-
// launched `aiwf promote`/`aiwf cancel` subprocesses racing the same
// milestone+AC actually produce a parseable envelope per actor and a
// check-clean tree — mirroring concurrent_move_test.go's own shape.

// TestConcurrentMilestoneRaceScenario_RealBinary_ErrorsWhenSetupBinaryMissing
// points Setup itself at a nonexistent binary path — Setup's first
// subprocess launch (seeding the epic) fails before any milestone or
// AC is ever seeded.
func TestConcurrentMilestoneRaceScenario_RealBinary_ErrorsWhenSetupBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	dir := newVerbSequenceTestRepo(t)

	s := NewConcurrentMilestoneRaceScenario(filepath.Join(t.TempDir(), "no-such-aiwf-binary"), 8, 1)
	if err := s.Setup(dir); err == nil {
		t.Fatal("expected Setup to error when the aiwf binary path doesn't exist")
	} else if !strings.Contains(err.Error(), "seeding the epic") {
		t.Fatalf("expected the launch failure to name the epic-seeding step, got: %v", err)
	}
}

// TestConcurrentMilestoneRaceScenario_RealBinary_ErrorsWhenBinaryMissing
// runs a real Setup (so the repo has a real milestone+AC and
// gitHeadCommitCount's own "before" call succeeds), then points a
// fresh scenario carrying Setup's own output at a nonexistent binary
// path for Run — every launched actor's subprocess then fails at the
// OS level, surfacing from the per-actor result-processing loop after
// the (successful) concurrent fan-out.
func TestConcurrentMilestoneRaceScenario_RealBinary_ErrorsWhenBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)

	setup := NewConcurrentMilestoneRaceScenario(bin, 8, 1)
	if err := setup.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	broken := &ConcurrentMilestoneRaceScenario{
		aiwfBin:     filepath.Join(t.TempDir(), "no-such-aiwf-binary"),
		n:           setup.n,
		milestoneID: setup.milestoneID,
	}
	if err := broken.Run(dir); err == nil {
		t.Fatal("expected Run to error when the aiwf binary path doesn't exist")
	} else if !strings.Contains(err.Error(), "running aiwf") {
		t.Fatalf("expected the launch failure to name the actor's aiwf invocation, got: %v", err)
	}
}

// TestConcurrentMilestoneRaceScenario_RealBinary_EveryActorRunsAndTreeStaysCheckClean
// is the AC-1 scenario itself, repeated via RunRepeated for
// statistical coverage across real goroutine/subprocess timing
// (mirroring ConcurrentMoveScenario's own repeated real-binary test):
// n real actors race promote/cancel against one shared milestone+AC,
// and the resulting tree must stay check-clean beyond baseline noise
// on every attempt, regardless of which side of the race won.
func TestConcurrentMilestoneRaceScenario_RealBinary_EveryActorRunsAndTreeStaysCheckClean(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	base := t.TempDir()

	const n = 8
	newScenario := func(seed int64) Scenario {
		return NewConcurrentMilestoneRaceScenario(bin, n, seed)
	}

	rw := newReportWriter(&countingWriter{})
	results, err := RunRepeated(newScenario, base, 5, seedSequence(1, 2, 3, 4, 5), rw, "", nil)
	if err != nil {
		t.Fatalf("RunRepeated: %v", err)
	}
	for i, r := range results {
		if !r.Passed {
			t.Fatalf("attempt %d found violations (dir preserved at %s):\n%+v", i, r.Dir, r.Violations)
		}
	}
}

// TestConcurrentMilestoneRaceScenario_RealBinary_OutcomeShapeAndCommitAccounting
// drives Setup+Run directly (bypassing RunRepeated/RunScenario) so the
// test can inspect the scenario's own captured fields — the invariants
// the milestone FSM and the open-AC cancel guard together guarantee
// regardless of which actor wins the race: every actor is accounted
// for, exactly one promote actor succeeds (the AC can only transition
// open -> met once), the AC always ends up "met", the milestone lands
// on one of its two legal outcomes, and the commit count landed after
// the race matches exactly the number of actors that reported "ok" —
// the harness sanity signal the milestone spec calls for.
func TestConcurrentMilestoneRaceScenario_RealBinary_OutcomeShapeAndCommitAccounting(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)

	const n = 8
	const attempts = 5
	for attempt := 0; attempt < attempts; attempt++ {
		dir := newVerbSequenceTestRepo(t)
		s := NewConcurrentMilestoneRaceScenario(bin, n, int64(attempt))
		if err := s.Setup(dir); err != nil {
			t.Fatalf("attempt %d: Setup: %v", attempt, err)
		}
		if err := s.Run(dir); err != nil {
			t.Fatalf("attempt %d: Run: %v", attempt, err)
		}
		if violations := s.Verify(dir); len(violations) != 0 {
			t.Fatalf("attempt %d: Verify: %+v", attempt, violations)
		}

		if len(s.outcomes) != n {
			t.Fatalf("attempt %d: len(outcomes) = %d, want %d", attempt, len(s.outcomes), n)
		}
		var promoteCount, cancelCount, okCount, promoteOKCount, cancelOKCount int
		for _, oc := range s.outcomes {
			switch oc.operation {
			case raceOpPromote:
				promoteCount++
				if oc.status == "ok" {
					promoteOKCount++
				}
			case raceOpCancel:
				cancelCount++
				if oc.status == "ok" {
					cancelOKCount++
				}
			default:
				t.Fatalf("attempt %d: unexpected operation %q", attempt, oc.operation)
			}
			if oc.status == "ok" {
				okCount++
			}
		}
		if promoteCount != n/2 || cancelCount != n-n/2 {
			t.Fatalf("attempt %d: promoteCount=%d cancelCount=%d, want %d/%d", attempt, promoteCount, cancelCount, n/2, n-n/2)
		}
		if promoteOKCount != 1 {
			t.Fatalf("attempt %d: promoteOKCount = %d, want exactly 1 (the AC can only transition open -> met once)", attempt, promoteOKCount)
		}
		if cancelOKCount != 0 && cancelOKCount != 1 {
			t.Fatalf("attempt %d: cancelOKCount = %d, want 0 or 1", attempt, cancelOKCount)
		}

		if s.finalACStatus != "met" {
			t.Fatalf("attempt %d: finalACStatus = %q, want %q", attempt, s.finalACStatus, "met")
		}
		if s.finalMilestoneStatus != "draft" && s.finalMilestoneStatus != "cancelled" {
			t.Fatalf("attempt %d: finalMilestoneStatus = %q, want draft or cancelled", attempt, s.finalMilestoneStatus)
		}
		if cancelOKCount == 1 && s.finalMilestoneStatus != "cancelled" {
			t.Fatalf("attempt %d: a cancel actor reported ok but finalMilestoneStatus = %q, want cancelled", attempt, s.finalMilestoneStatus)
		}
		if cancelOKCount == 0 && s.finalMilestoneStatus != "draft" {
			t.Fatalf("attempt %d: no cancel actor reported ok but finalMilestoneStatus = %q, want draft", attempt, s.finalMilestoneStatus)
		}

		if s.after != s.before+okCount {
			t.Fatalf("attempt %d: commit count %d -> %d after %d ok outcomes, want exactly +%d", attempt, s.before, s.after, okCount, okCount)
		}
	}
}

// TestConcurrentMilestoneRaceScenario_RealBinary_DetectsAnUnexpectedCheckFinding
// points Run at a stand-in "aiwf" that falsely reports an unbaselined
// finding from `aiwf check` while delegating every other subcommand
// (including the actors' own promote/cancel and the post-race show
// calls) to the real binary — closing the same class of vacuity gap
// ConcurrentMoveScenario's own DetectsAGenuineDivergence test closes:
// a healthy run alone can't tell a correctly-wired check-clean layer
// (Run threading checkEnv.Findings through classifyAgainstBaseline
// into s.violations, and Verify returning it) from one that silently
// drops it, since a healthy repo produces zero violations either way.
func TestConcurrentMilestoneRaceScenario_RealBinary_DetectsAnUnexpectedCheckFinding(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	realBin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)

	setup := NewConcurrentMilestoneRaceScenario(realBin, 8, 1)
	if err := setup.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	fakeBin := writeFakeAiwfUnexpectedCheckFinding(t, realBin)
	broken := &ConcurrentMilestoneRaceScenario{
		aiwfBin:     fakeBin,
		n:           setup.n,
		milestoneID: setup.milestoneID,
	}
	if err := broken.Run(dir); err != nil {
		t.Fatalf("Run: %v", err)
	}
	violations := broken.Verify(dir)
	if len(violations) == 0 {
		t.Fatal("expected violations from a check finding outside the curated baseline, got none")
	}
}

// writeFakeAiwfUnexpectedCheckFinding writes an executable shell script
// standing in for `aiwf`: it falsely reports an unbaselined
// warning-severity finding for "check", and delegates every other
// subcommand to realBin — so the actors' own promote/cancel calls and
// the post-race show calls still run against the real, unchanged
// binary.
func writeFakeAiwfUnexpectedCheckFinding(t *testing.T, realBin string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "aiwf")
	script := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "check" ]; then
  echo '{"status":"findings","findings":[{"code":"some-unbaselined-code","severity":"warning"}],"result":{},"metadata":{}}'
  exit 1
fi
exec %q "$@"
`, realBin)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil { //nolint:gosec // deliberately executable; a test-local stand-in binary, not attacker-controlled input
		t.Fatalf("writing fake aiwf binary: %v", err)
	}
	return path
}
