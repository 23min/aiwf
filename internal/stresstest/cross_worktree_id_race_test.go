package stresstest

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// cross_worktree_id_race_test.go — real-subprocess coverage for
// CrossWorktreeIDRaceScenario (M-0241/AC-3). The pure decision logic
// (classifyCrossWorktreeRace / findEntityFile) is pinned exhaustively
// in cross_worktree_id_race_classify_test.go against fabricated
// inputs; this is the actual AC-3 scenario, repeated via
// M-0240's RunRepeated so at least one attempt hits the real race
// window and exercises the detect-and-resolve path end to end.

// TestCrossWorktreeIDRaceScenario_RealBinary_SequentialActorsDoNotCollide
// runs the two actors SEQUENTIALLY (actor A commits before actor B
// even starts) rather than racing them — real concurrent racing in
// this environment collides reliably enough that the "no collision"
// branch is otherwise never exercised in a repeated run. Sequential
// execution is a real, reachable, non-racing outcome for the same
// two-sibling-worktree setup: actor B's id allocator sees actor A's
// already-committed local ref and picks the next free id, per this
// repo's own cross-branch allocation scan.
func TestCrossWorktreeIDRaceScenario_RealBinary_SequentialActorsDoNotCollide(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := t.TempDir()

	s := NewCrossWorktreeIDRaceScenario(bin, entity.KindGap, 1)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	wtA := filepath.Join(dir, "wt-a")
	wtB := filepath.Join(dir, "wt-b")

	a := launchAddIn(bin, wtA, entity.KindGap, actorATitle)
	envA, err := parseVerbEnvelope([]string{"add", "gap"}, a.out)
	if err != nil {
		t.Fatalf("parse actor A: %v", err)
	}
	b := launchAddIn(bin, wtB, entity.KindGap, actorBTitle)
	envB, err := parseVerbEnvelope([]string{"add", "gap"}, b.out)
	if err != nil {
		t.Fatalf("parse actor B: %v", err)
	}
	if envA.Metadata.EntityID == envB.Metadata.EntityID {
		t.Fatalf("expected sequential actors to avoid colliding, both got %s", envA.Metadata.EntityID)
	}

	if err := s.reconcile(wtA, envA, envB); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if s.Collided() {
		t.Fatal("expected Collided() to be false after a non-colliding sequential run")
	}
	if len(s.Verify(dir)) != 0 {
		t.Fatalf("expected zero violations for a non-colliding attempt, got: %+v", s.Verify(dir))
	}
}

// TestCrossWorktreeIDRaceScenario_ReconcileErrorsWhenAnActorDidNotSucceed
// drives reconcile directly with a fabricated non-"ok" envelope,
// pinning the defensive guard against ever attempting to merge or
// classify a race whose add itself failed.
func TestCrossWorktreeIDRaceScenario_ReconcileErrorsWhenAnActorDidNotSucceed(t *testing.T) {
	t.Parallel()
	s := NewCrossWorktreeIDRaceScenario("unused", entity.KindGap, 1)
	err := s.reconcile(t.TempDir(), verbEnvelope{Status: "ok"}, verbEnvelope{Status: "error"})
	if err == nil {
		t.Fatal("expected reconcile to error when an actor's add did not report ok")
	}
}

func TestCrossWorktreeIDRaceScenario_RealBinary_ErrorsWhenBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	base := t.TempDir()

	s := NewCrossWorktreeIDRaceScenario(filepath.Join(t.TempDir(), "no-such-aiwf-binary"), entity.KindGap, 1)
	if _, err := RunScenario(s, base); err == nil {
		t.Fatal("expected RunScenario to propagate the launch-failure error")
	} else if !strings.Contains(err.Error(), "launching aiwf add across sibling worktrees") {
		t.Fatalf("expected the launch failure to name the cross-worktree add step, got: %v", err)
	}
}

// TestCrossWorktreeIDRaceScenario_RealBinaryRepeatedHitsARealCollision
// is the AC-3 scenario itself, repeated so at least one attempt
// actually races two sibling worktrees into a real duplicate id —
// avoiding the vacuous "the race window was never hit, so trivially
// no violations" pass.
func TestCrossWorktreeIDRaceScenario_RealBinaryRepeatedHitsARealCollision(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	base := t.TempDir()

	var scenarios []*CrossWorktreeIDRaceScenario
	newScenario := func(seed int64) Scenario {
		s := NewCrossWorktreeIDRaceScenario(bin, entity.KindGap, seed)
		scenarios = append(scenarios, s)
		return s
	}

	rw := newReportWriter(&countingWriter{})
	results, err := RunRepeated(newScenario, base, 5, seedSequence(1, 2, 3, 4, 5), rw)
	if err != nil {
		t.Fatalf("RunRepeated: %v", err)
	}
	for i, r := range results {
		if !r.Passed {
			t.Fatalf("attempt %d found violations (dir preserved at %s):\n%+v", i, r.Dir, r.Violations)
		}
	}

	anyCollided := false
	for _, s := range scenarios {
		if s.Collided() {
			anyCollided = true
			break
		}
	}
	if !anyCollided {
		t.Fatal("expected at least one of 5 repeated attempts to hit a real cross-worktree id collision — the detect-and-resolve path was never exercised")
	}
}
