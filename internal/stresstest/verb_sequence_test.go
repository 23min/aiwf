package stresstest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// verb_sequence_test.go — real-subprocess coverage for
// VerbSequenceScenario (M-0241/AC-1). The pure decision logic
// (classifyVerbSequenceStep / classifyCheckFindings) is pinned
// exhaustively in verb_sequence_classify_test.go against fabricated
// envelopes; these tests confirm the real binary's wiring actually
// produces the three JSON shapes that logic classifies: an
// FSM-legal success, an FSM-illegal refusal, and a legal-but-refused-
// by-an-orthogonal-business-rule case (gap's addressed-resolver
// gate) — plus the full end-to-end scenario across every kind.

// runGitOrFatal runs git with args in dir, fatal'ing the test on
// failure. Test-local helper, not exported — this package's
// production code (verb_sequence.go) runs the same git subcommands
// via exec.Command directly rather than sharing a helper with tests.
func runGitOrFatal(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func newVerbSequenceTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGitOrFatal(t, dir, "init", "-q")
	runGitOrFatal(t, dir, "config", "user.email", "stresstest@example.com")
	runGitOrFatal(t, dir, "config", "user.name", "stresstest")
	return dir
}

func TestVerbSequenceScenario_RealBinary_LegalTransitionSucceedsWithOneCommit(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)
	s := &VerbSequenceScenario{aiwfBin: bin}

	addEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "adr", "--title", "t", "--body", "b")
	if err != nil {
		t.Fatalf("add adr: %v", err)
	}
	if addEnv.Status != "ok" {
		t.Fatalf("add adr refused: %+v", addEnv.Error)
	}
	id := addEnv.Metadata.EntityID

	before, err := gitHeadCommitCount(dir)
	if err != nil {
		t.Fatalf("commit count before: %v", err)
	}
	env, err := runAiwfJSON(s.aiwfBin, dir, "promote", id, "accepted")
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	after, err := gitHeadCommitCount(dir)
	if err != nil {
		t.Fatalf("commit count after: %v", err)
	}

	next, violations := classifyVerbSequenceStep(entity.KindADR, "proposed", "accepted", before, after, env)
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %+v", violations)
	}
	if next != "accepted" {
		t.Fatalf("next = %q, want %q", next, "accepted")
	}
	if after != before+1 {
		t.Fatalf("commit count %d -> %d, want exactly +1", before, after)
	}
}

func TestVerbSequenceScenario_RealBinary_IllegalTransitionRefusedAsFSMIllegal(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)
	s := &VerbSequenceScenario{aiwfBin: bin}

	addEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "adr", "--title", "t", "--body", "b")
	if err != nil {
		t.Fatalf("add adr: %v", err)
	}
	id := addEnv.Metadata.EntityID

	before, err := gitHeadCommitCount(dir)
	if err != nil {
		t.Fatalf("commit count before: %v", err)
	}
	// proposed -> superseded is not a legal ADR transition (only
	// accepted/rejected are reachable from proposed).
	env, err := runAiwfJSON(s.aiwfBin, dir, "promote", id, "superseded")
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	after, err := gitHeadCommitCount(dir)
	if err != nil {
		t.Fatalf("commit count after: %v", err)
	}

	if env.Status != "error" || env.Error == nil || env.Error.Code != entity.CodeFSMTransitionIllegal.ID {
		t.Fatalf("expected an fsm-transition-illegal refusal, got status=%s error=%+v", env.Status, env.Error)
	}
	next, violations := classifyVerbSequenceStep(entity.KindADR, "proposed", "superseded", before, after, env)
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %+v", violations)
	}
	if next != "proposed" {
		t.Fatalf("next = %q, want unchanged %q", next, "proposed")
	}
	if after != before {
		t.Fatalf("refused transition landed a commit: %d -> %d", before, after)
	}
}

func TestVerbSequenceScenario_RealBinary_LegalTransitionRefusedByOrthogonalBusinessRule(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)
	s := &VerbSequenceScenario{aiwfBin: bin}

	addEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "gap", "--title", "t", "--body", "b")
	if err != nil {
		t.Fatalf("add gap: %v", err)
	}
	id := addEnv.Metadata.EntityID

	before, err := gitHeadCommitCount(dir)
	if err != nil {
		t.Fatalf("commit count before: %v", err)
	}
	// open -> addressed IS FSM-legal for gap, but the verb layer
	// additionally requires --by/--by-commit (the
	// gap-addressed-has-resolver rule) — a real, orthogonal
	// business-rule refusal distinct from FSM illegality.
	env, err := runAiwfJSON(s.aiwfBin, dir, "promote", id, "addressed")
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	after, err := gitHeadCommitCount(dir)
	if err != nil {
		t.Fatalf("commit count after: %v", err)
	}

	if env.Status != "error" || env.Error == nil || env.Error.Code == entity.CodeFSMTransitionIllegal.ID {
		t.Fatalf("expected a non-FSM refusal (business rule), got status=%s error=%+v", env.Status, env.Error)
	}
	next, violations := classifyVerbSequenceStep(entity.KindGap, "open", "addressed", before, after, env)
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %+v", violations)
	}
	if next != "open" {
		t.Fatalf("next = %q, want unchanged %q", next, "open")
	}
	if after != before {
		t.Fatalf("refused transition landed a commit: %d -> %d", before, after)
	}
}

// TestVerbSequenceScenario_RealBinary_RunSurfacesAScratchEpicCreationRefusal
// pre-seeds a colliding E-0001 entity file (an id collision the
// `ids-unique` rule refuses at error severity) so Run's very first
// `aiwf add epic` call — seeding the move-target scratch epic
// (M-0250/AC-2), the first entity Run ever creates — reports
// something other than "ok", pinning that Run wraps and surfaces that
// specific refusal rather than pressing on with an empty entity id.
func TestVerbSequenceScenario_RealBinary_RunSurfacesAScratchEpicCreationRefusal(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)

	collidingDir := filepath.Join(dir, "work", "epics", "E-0001-stress-epic")
	if err := os.MkdirAll(collidingDir, 0o755); err != nil {
		t.Fatalf("mkdir colliding epic dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(collidingDir, "epic.md"), []byte("not valid frontmatter\n"), 0o644); err != nil {
		t.Fatalf("write colliding epic file: %v", err)
	}

	s := NewVerbSequenceScenario(bin, 1, 6)
	if err := s.Run(dir); err == nil {
		t.Fatal("expected Run to surface the id-collision refusal on the scratch epic's own `aiwf add epic` call")
	} else if !strings.Contains(err.Error(), "seeding the move-target scratch epic") || !strings.Contains(err.Error(), "did not report ok") {
		t.Fatalf("expected the refusal to name the scratch epic and report a non-ok status, got: %v", err)
	}
}

// TestVerbSequenceScenario_RealBinary_RunSurfacesAnAllKindsLoopCreationRefusal
// pre-seeds a colliding E-0002 entity file — the id Run's special-cased
// epic-creation call allocates, one past the scratch epic's E-0001 —
// so createWalkerEntity's generic "did not report ok" fallback
// (distinct from the scratch epic's own refusal path above) is
// exercised directly.
func TestVerbSequenceScenario_RealBinary_RunSurfacesAnAllKindsLoopCreationRefusal(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)

	collidingDir := filepath.Join(dir, "work", "epics", "E-0002-stress-epic")
	if err := os.MkdirAll(collidingDir, 0o755); err != nil {
		t.Fatalf("mkdir colliding epic dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(collidingDir, "epic.md"), []byte("not valid frontmatter\n"), 0o644); err != nil {
		t.Fatalf("write colliding epic file: %v", err)
	}

	s := NewVerbSequenceScenario(bin, 1, 6)
	if err := s.Run(dir); err == nil {
		t.Fatal("expected Run to surface the id-collision refusal on the special-cased epic-creation `aiwf add` call")
	} else if !strings.Contains(err.Error(), "creating a epic entity") || !strings.Contains(err.Error(), "did not report ok") {
		t.Fatalf("expected the refusal to name the epic kind and report a non-ok status, got: %v", err)
	}
}

// TestVerbSequenceScenario_RealBinary_RunErrorsWhenBinaryMissing
// points the scenario at a nonexistent binary path so the very first
// subprocess launch (inside runAiwfJSON, called from Run's `aiwf
// add epic`) fails at the OS level rather than exiting non-zero —
// pinning that this mechanical failure surfaces as a wrapped Go
// error rather than a misread envelope.
func TestVerbSequenceScenario_RealBinary_RunErrorsWhenBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	dir := newVerbSequenceTestRepo(t)

	s := NewVerbSequenceScenario(filepath.Join(t.TempDir(), "no-such-aiwf-binary"), 1, 6)
	err := s.Run(dir)
	if err == nil {
		t.Fatal("expected Run to error when the aiwf binary path doesn't exist")
	}
	if !strings.Contains(err.Error(), "running aiwf") {
		t.Fatalf("expected the launch failure to surface via runAiwfJSON's wrapping, got: %v", err)
	}
}

// TestVerbSequenceScenario_RealBinary_StepRenameSucceeds pins
// stepRename's real wiring: a fresh rename against a real entity
// reports ok and produces no violation.
func TestVerbSequenceScenario_RealBinary_StepRenameSucceeds(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)
	s := &VerbSequenceScenario{aiwfBin: bin}

	addEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "adr", "--title", "t", "--body", "b")
	if err != nil {
		t.Fatalf("add adr: %v", err)
	}
	id := addEnv.Metadata.EntityID

	violations, err := s.stepRename(dir, id)
	if err != nil {
		t.Fatalf("stepRename: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %+v", violations)
	}
	if s.renameCounter != 1 {
		t.Fatalf("renameCounter = %d, want 1", s.renameCounter)
	}
}

// TestVerbSequenceScenario_RealBinary_StepRetitleSucceeds mirrors
// StepRenameSucceeds for stepRetitle.
func TestVerbSequenceScenario_RealBinary_StepRetitleSucceeds(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)
	s := &VerbSequenceScenario{aiwfBin: bin}

	addEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "adr", "--title", "t", "--body", "b")
	if err != nil {
		t.Fatalf("add adr: %v", err)
	}
	id := addEnv.Metadata.EntityID

	violations, err := s.stepRetitle(dir, id)
	if err != nil {
		t.Fatalf("stepRetitle: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %+v", violations)
	}
	if s.retitleCounter != 1 {
		t.Fatalf("retitleCounter = %d, want 1", s.retitleCounter)
	}
}

// TestVerbSequenceScenario_RealBinary_StepArchiveNoOpWhenNothingTerminal
// pins that archive --apply against a repo with nothing terminal is a
// legitimate ok, not a violation.
func TestVerbSequenceScenario_RealBinary_StepArchiveNoOpWhenNothingTerminal(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)
	s := &VerbSequenceScenario{aiwfBin: bin}

	violations, err := s.stepArchive(dir)
	if err != nil {
		t.Fatalf("stepArchive: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %+v", violations)
	}
}

// TestVerbSequenceScenario_RealBinary_StepArchiveSweepsATerminalEntity
// confirms stepArchive actually sweeps a genuinely terminal entity
// (an ADR promoted to rejected, one of ADR's two terminal statuses —
// accepted is NOT terminal, since accepted -> superseded stays legal),
// not just tolerates the no-op case.
func TestVerbSequenceScenario_RealBinary_StepArchiveSweepsATerminalEntity(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)
	s := &VerbSequenceScenario{aiwfBin: bin}

	addEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "adr", "--title", "t", "--body", "b")
	if err != nil {
		t.Fatalf("add adr: %v", err)
	}
	id := addEnv.Metadata.EntityID
	if promEnv, promErr := runAiwfJSON(s.aiwfBin, dir, "promote", id, "rejected"); promErr != nil {
		t.Fatalf("promote: %v", promErr)
	} else if promEnv.Status != "ok" {
		t.Fatalf("promote refused: %+v", promEnv.Error)
	}

	violations, err := s.stepArchive(dir)
	if err != nil {
		t.Fatalf("stepArchive: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %+v", violations)
	}

	showEnv, err := runAiwfJSON(s.aiwfBin, dir, "show", id)
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if !strings.Contains(showEnv.Result.Path, "/archive/") {
		t.Fatalf("show %s path = %q, want it under an archive/ subdir after stepArchive", id, showEnv.Result.Path)
	}
}

// TestVerbSequenceScenario_RealBinary_StepMoveRelocatesAndAlternates
// pins stepMove's real wiring: a move to mv.target() succeeds,
// relocates the milestone, and applyMoved() swaps current/other so a
// second stepMove call moves it back.
func TestVerbSequenceScenario_RealBinary_StepMoveRelocatesAndAlternates(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)
	s := &VerbSequenceScenario{aiwfBin: bin}

	epicAEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "epic", "--title", "epic a", "--body", "b")
	if err != nil {
		t.Fatalf("add epic a: %v", err)
	}
	epicA := epicAEnv.Metadata.EntityID
	epicBEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "epic", "--title", "epic b", "--body", "b")
	if err != nil {
		t.Fatalf("add epic b: %v", err)
	}
	epicB := epicBEnv.Metadata.EntityID
	msEnv, err := runAiwfJSON(s.aiwfBin, dir, "add", "milestone", "--epic", epicA, "--tdd", "none", "--title", "m", "--body", "b")
	if err != nil {
		t.Fatalf("add milestone: %v", err)
	}
	msID := msEnv.Metadata.EntityID

	mv := &moveState{current: epicA, other: epicB}
	violations, err := s.stepMove(dir, msID, mv)
	if err != nil {
		t.Fatalf("stepMove: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %+v", violations)
	}
	if mv.current != epicB || mv.other != epicA {
		t.Fatalf("after first move: current=%q other=%q, want current=%q other=%q", mv.current, mv.other, epicB, epicA)
	}
	showEnv, err := runAiwfJSON(s.aiwfBin, dir, "show", msID)
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if showEnv.Result.Parent != epicB {
		t.Fatalf("milestone parent = %q, want %q", showEnv.Result.Parent, epicB)
	}

	violations, err = s.stepMove(dir, msID, mv)
	if err != nil {
		t.Fatalf("second stepMove: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("unexpected violations on the move back: %+v", violations)
	}
	if mv.current != epicA || mv.other != epicB {
		t.Fatalf("after second move: current=%q other=%q, want current=%q other=%q", mv.current, mv.other, epicA, epicB)
	}
}

// TestVerbSequenceScenario_RealBinary_WalkDispatchesEveryOperation
// drives walk itself (not the individual stepX methods directly) with
// a seed/step count empirically confirmed to draw every one of the
// five weighted operations at least once, pinning that walk's switch
// statement really does dispatch to every case — not just that each
// stepX method works in isolation, and not left to the statistical
// luck of whichever seeds TestVerbSequenceScenario_FullWalkAcrossAllKindsPasses
// happens to use. seed=0/steps=30 was found by exhaustively searching
// seeds 0..199 with the walker's exact weight table.
func TestVerbSequenceScenario_RealBinary_WalkDispatchesEveryOperation(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)

	epicAEnv, err := runAiwfJSON(bin, dir, "add", "epic", "--title", "epic a", "--body", "b")
	if err != nil {
		t.Fatalf("add epic a: %v", err)
	}
	epicA := epicAEnv.Metadata.EntityID
	epicBEnv, err := runAiwfJSON(bin, dir, "add", "epic", "--title", "epic b", "--body", "b")
	if err != nil {
		t.Fatalf("add epic b: %v", err)
	}
	epicB := epicBEnv.Metadata.EntityID
	msEnv, err := runAiwfJSON(bin, dir, "add", "milestone", "--epic", epicA, "--tdd", "none", "--title", "m", "--body", "b")
	if err != nil {
		t.Fatalf("add milestone: %v", err)
	}
	msID := msEnv.Metadata.EntityID
	showEnv, err := runAiwfJSON(bin, dir, "show", msID)
	if err != nil {
		t.Fatalf("show: %v", err)
	}

	s := NewVerbSequenceScenario(bin, 0, 30)
	mv := &moveState{current: epicA, other: epicB}
	if err := s.walk(dir, entity.KindMilestone, msID, showEnv.Result.Status, mv); err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(s.violations) != 0 {
		t.Fatalf("unexpected violations: %+v", s.violations)
	}
	if s.renameCounter == 0 {
		t.Error("renameCounter == 0, want walk to have dispatched to the rename case at least once")
	}
	if s.retitleCounter == 0 {
		t.Error("retitleCounter == 0, want walk to have dispatched to the retitle case at least once")
	}
	if s.archiveCounter <= 0 {
		t.Errorf("archiveCounter = %d, want walk to have dispatched to the archive case at least once", s.archiveCounter)
	}
	if s.moveCounter <= 0 {
		t.Errorf("moveCounter = %d, want walk to have dispatched to the move case at least once", s.moveCounter)
	}
}

// TestVerbSequenceScenario_RealBinary_RunConstructsMoveStateForTheMilestone
// drives Run itself (not walk directly), pinning that Run's own `mv :=
// &moveState{current: epicID, other: altEpicID}` construction (only
// reachable when kind == milestone) is exercised through the real
// end-to-end path, not just via the direct walk() call above. Before
// G-0401's fix this needed a hand-picked seed/steps combination that
// happened to keep the epic non-terminal through its own walk — the
// FSM's proposed/active/done/cancelled shape made even a single
// promote draw a coin flip on landing terminal, and the milestone was
// skipped whenever it did. Now that the milestone is created
// immediately after the epic and before the epic ever takes a walk
// step, the outcome no longer depends on the epic's walk at all — this
// runs at 12 steps, matching cmd/stresstest/registry.go's
// defaultVerbSequenceSteps, to confirm the fix holds at the scenario's
// real registered parameters, not just a contrived minimal case.
func TestVerbSequenceScenario_RealBinary_RunConstructsMoveStateForTheMilestone(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := newVerbSequenceTestRepo(t)

	s := NewVerbSequenceScenario(bin, 0, 12)
	if err := s.Run(dir); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(s.violations) != 0 {
		t.Fatalf("unexpected violations: %+v", s.violations)
	}

	showEnv, err := runAiwfJSON(bin, dir, "show", "M-0001")
	if err != nil {
		t.Fatalf("show M-0001: %v", err)
	}
	if showEnv.Status != "ok" {
		t.Fatalf("M-0001 was not created (show status=%s) — the milestone should now always be created regardless of the epic's own walk", showEnv.Status)
	}
	if s.moveCounter == 0 {
		t.Error("moveCounter == 0, want the milestone's walk to have drawn move at least once at the registered scenario's real step count")
	}
}

func TestVerbSequenceScenario_FullWalkAcrossAllKindsPasses(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)

	for _, seed := range []int64{1, 2, 42} {
		seed := seed
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			result, err := RunScenario(NewVerbSequenceScenario(bin, seed, 6), base)
			if err != nil {
				t.Fatalf("RunScenario: %v", err)
			}
			if !result.Passed {
				t.Fatalf("verb-sequence scenario found violations (dir preserved at %s):\n%+v", result.Dir, result.Violations)
			}
		})
	}
}
