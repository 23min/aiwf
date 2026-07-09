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

	addEnv, err := s.runAiwfJSON(dir, "add", "adr", "--title", "t", "--body", "b")
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
	env, err := s.runAiwfJSON(dir, "promote", id, "accepted")
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

	addEnv, err := s.runAiwfJSON(dir, "add", "adr", "--title", "t", "--body", "b")
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
	env, err := s.runAiwfJSON(dir, "promote", id, "superseded")
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

	addEnv, err := s.runAiwfJSON(dir, "add", "gap", "--title", "t", "--body", "b")
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
	env, err := s.runAiwfJSON(dir, "promote", id, "addressed")
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

// TestVerbSequenceScenario_RealBinary_RunSurfacesACreationRefusal
// pre-seeds a colliding E-0001 entity file (an id collision the
// `ids-unique` rule refuses at error severity) so Run's very first
// `aiwf add epic` call reports something other than "ok", pinning
// that Run wraps and surfaces the refusal rather than pressing on
// with an empty entity id.
func TestVerbSequenceScenario_RealBinary_RunSurfacesACreationRefusal(t *testing.T) {
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
		t.Fatal("expected Run to surface the id-collision refusal on the very first `aiwf add epic` call")
	} else if !strings.Contains(err.Error(), "did not report ok") {
		t.Fatalf("expected the refusal to be reported as a non-ok status, got: %v", err)
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
