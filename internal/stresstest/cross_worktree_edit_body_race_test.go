package stresstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cross_worktree_edit_body_race_test.go — real-subprocess coverage for
// CrossWorktreeEditBodyRaceScenario (M-0243/AC-2). The pure decision
// logic (classifyCrossWorktreeEditBodyRace) is pinned exhaustively in
// cross_worktree_edit_body_race_classify_test.go against fabricated
// merge outcomes; this is the actual scenario, driving two real
// `aiwf edit-body` subprocesses through a real cross-worktree merge.

func TestCrossWorktreeEditBodyRaceScenario_RealBinary_ConfirmsObservableOutcome(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	base := t.TempDir()

	s := NewCrossWorktreeEditBodyRaceScenario(bin)
	result, err := RunScenario(s, base)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if !result.Passed {
		t.Fatalf("cross-worktree-edit-body-race scenario found violations (dir preserved at %s):\n%+v", result.Dir, result.Violations)
	}
}

func TestCrossWorktreeEditBodyRaceScenario_RealBinary_ErrorsWhenBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	base := t.TempDir()

	s := NewCrossWorktreeEditBodyRaceScenario(filepath.Join(t.TempDir(), "no-such-aiwf-binary"))
	if _, err := RunScenario(s, base); err == nil {
		t.Fatal("expected RunScenario to propagate the launch-failure error")
	} else if !strings.Contains(err.Error(), "seeding the shared entity") {
		t.Fatalf("expected the failure to name the seeding step, got: %v", err)
	}
}

// TestCrossWorktreeEditBodyRaceScenario_RealBinary_SetupSurfacesASeedingRefusal
// pre-seeds a colliding G-0001 entity file in the fresh repo before
// Setup's own `aiwf add` call, mirroring M-0241/AC-5's same pre-seed
// technique, pinning that Setup wraps and surfaces the refusal.
func TestCrossWorktreeEditBodyRaceScenario_RealBinary_SetupSurfacesASeedingRefusal(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := t.TempDir()

	gapsDir := filepath.Join(dir, "main", "work", "gaps")
	if err := os.MkdirAll(gapsDir, 0o755); err != nil {
		t.Fatalf("mkdir colliding gap dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gapsDir, "G-0001-collision.md"), []byte("not valid frontmatter\n"), 0o644); err != nil {
		t.Fatalf("write colliding gap file: %v", err)
	}

	s := NewCrossWorktreeEditBodyRaceScenario(bin)
	if err := s.Setup(dir); err == nil {
		t.Fatal("expected Setup to surface the seeding refusal")
	} else if !strings.Contains(err.Error(), "did not report ok") {
		t.Fatalf("expected the refusal to name the seeding step, got: %v", err)
	}
}

// TestCrossWorktreeEditBodyRaceScenario_RealBinary_RunErrorsWhenOperatorEditNotOK
// removes operator B's worktree copy of the shared entity file after a
// successful Setup, so operator B's edit-body call refuses (nothing to
// edit), pinning that Run surfaces a non-"ok" edit-body status.
func TestCrossWorktreeEditBodyRaceScenario_RealBinary_RunErrorsWhenOperatorEditNotOK(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := t.TempDir()

	s := NewCrossWorktreeEditBodyRaceScenario(bin)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	entityPath := filepath.Join(dir, "wt-b", "work", "gaps", editBodyRaceEntityID+"-race.md")
	if err := os.Remove(entityPath); err != nil {
		t.Fatalf("removing operator B's entity file: %v", err)
	}

	if err := s.Run(dir); err == nil {
		t.Fatal("expected Run to surface operator B's edit-body refusal")
	} else if !strings.Contains(err.Error(), "did not report ok") {
		t.Fatalf("expected the refusal to name the edit-body step, got: %v", err)
	}
}
