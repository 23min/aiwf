//go:build stress

package stresstest

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// mid_write_kill_test.go — real-subprocess coverage for
// MidWriteKillScenario (M-0242/AC-2). The pure decision logic
// (classifyMidWriteKillOutcome) is pinned exhaustively in
// mid_write_kill_classify_test.go against fabricated byte slices;
// this is the actual AC-2 scenario, driving a real, killable aiwf
// subprocess and a real filesystem race window.

func TestMidWriteKillScenario_RealBinary_ConfirmsNoHalfWrittenFile(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	base := t.TempDir()

	s := NewMidWriteKillScenario(bin)
	result, err := RunScenario(s, base)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if !result.Passed {
		t.Fatalf("mid-write-kill scenario found violations (dir preserved at %s):\n%+v", result.Dir, result.Violations)
	}
}

func TestMidWriteKillScenario_RealBinary_ErrorsWhenBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	base := t.TempDir()

	s := NewMidWriteKillScenario(filepath.Join(t.TempDir(), "no-such-aiwf-binary"))
	if _, err := RunScenario(s, base); err == nil {
		t.Fatal("expected RunScenario to propagate the launch-failure error")
	} else if !strings.Contains(err.Error(), "seeding") {
		t.Fatalf("expected the launch failure to name the seeding step, got: %v", err)
	}
}

// TestMidWriteKillScenario_RealBinary_RunErrorsWhenTargetGapFileMissing
// deletes the target repo's seeded gap file after a successful Setup,
// pinning Run's own initial readGapFile call (reading the pre-write
// bytes) rather than readGapFile's already-unit-tested internals.
func TestMidWriteKillScenario_RealBinary_RunErrorsWhenTargetGapFileMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := t.TempDir()

	s := NewMidWriteKillScenario(bin)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "target", "work", "gaps", "G-0001-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("expected exactly one seeded target gap file, got %v (err=%v)", matches, err)
	}
	if err := os.Remove(matches[0]); err != nil {
		t.Fatalf("removing seeded target gap file: %v", err)
	}

	if err := s.Run(dir); err == nil {
		t.Fatal("expected Run to error when the target's seeded gap file is missing")
	} else if !strings.Contains(err.Error(), "reading target's pre-write bytes") {
		t.Fatalf("expected the error to name the pre-write read step, got: %v", err)
	}
}

// TestMidWriteKillScenario_RealBinary_SetupSurfacesASeedingRefusal
// pre-seeds a colliding G-0001 entity file in the control repo (an id
// collision the ids-unique rule refuses at error severity, mirroring
// M-0241/AC-5's same pre-seed technique) so Setup's `add gap` call in
// that repo reports something other than "ok", pinning that Setup
// wraps and surfaces the refusal.
func TestMidWriteKillScenario_RealBinary_SetupSurfacesASeedingRefusal(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := t.TempDir()

	gapsDir := filepath.Join(dir, "control", "work", "gaps")
	if err := os.MkdirAll(gapsDir, 0o755); err != nil {
		t.Fatalf("mkdir colliding gap dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gapsDir, "G-0001-collision.md"), []byte("not valid frontmatter\n"), 0o644); err != nil {
		t.Fatalf("write colliding gap file: %v", err)
	}

	s := NewMidWriteKillScenario(bin)
	if err := s.Setup(dir); err == nil {
		t.Fatal("expected Setup to surface the id-collision refusal from the `aiwf add` call in the control repo")
	} else if !strings.Contains(err.Error(), "did not report ok") {
		t.Fatalf("expected the refusal to be reported as a non-ok status, got: %v", err)
	}
}

// TestMidWriteKillScenario_RealBinary_RunSurfacesAControlPromoteLockBusyRefusal
// holds the control repo's repolock via the AC-1 lockholder helper
// before calling Run: `aiwf promote`'s lock-busy refusal
// (internal/cli/cliutil.AcquireRepoLock) emits a valid --format=json
// envelope with status "error" (G-0391's fix), so runAiwfJSON parses
// it cleanly and Run's own promoteEnv.Status != "ok" branch reports
// it — the same classification path as an FSM refusal (see the
// FSMRefusal test below), just reached via lock contention instead.
func TestMidWriteKillScenario_RealBinary_RunSurfacesAControlPromoteLockBusyRefusal(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	lockHolderBin := sharedLockHolderBinary(t)
	dir := t.TempDir()

	s := NewMidWriteKillScenario(bin)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	holder := exec.Command(lockHolderBin, filepath.Join(dir, "control"))
	stdout, err := holder.StdoutPipe()
	if err != nil {
		t.Fatalf("wiring holder stdout: %v", err)
	}
	stdinW, err := holder.StdinPipe()
	if err != nil {
		t.Fatalf("wiring holder stdin: %v", err)
	}
	if startErr := holder.Start(); startErr != nil {
		t.Fatalf("starting holder: %v", startErr)
	}
	t.Cleanup(func() {
		_ = holder.Process.Kill()
		_ = holder.Wait()
	})
	if readyErr := waitForReady(stdout); readyErr != nil {
		t.Fatalf("waiting for holder to acquire: %v", readyErr)
	}

	err = s.Run(dir)
	if err == nil {
		t.Fatal("expected Run to surface the control-promote refusal while the lock is held")
	} else if !strings.Contains(err.Error(), "control promote did not report ok") {
		t.Fatalf("expected the refusal to name the control-promote step, got: %v", err)
	}

	_ = stdinW.Close()
}

// TestMidWriteKillScenario_RealBinary_RunSurfacesAControlPromoteFSMRefusal
// pre-advances the control repo's gap to "addressed" (a terminal state
// distinct from Run's "wontfix" target) before calling Run, so Run's own
// internal "promote to wontfix" attempt is a non-self FSM-illegal
// transition (addressed -> wontfix) and is refused — a refusal that
// (unlike the lock-busy path above) DOES emit a valid --format=json
// envelope with status "error", pinning Run's promoteEnv.Status != "ok"
// branch specifically. A same-status target (wontfix -> wontfix) is now a
// NoOp, not a refusal (ADR-0036), so the pre-advance must land on a
// *different* terminal to keep the transition genuinely illegal.
func TestMidWriteKillScenario_RealBinary_RunSurfacesAControlPromoteFSMRefusal(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := t.TempDir()

	s := NewMidWriteKillScenario(bin)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	// --force reaches the terminal `addressed` without a resolver; the only
	// requirement here is that the control gap ends in a terminal state
	// other than wontfix, so Run's promote-to-wontfix is refused.
	preEnv, err := runAiwfJSON(bin, filepath.Join(dir, "control"), "promote", "G-0001", "addressed",
		"--force", "--reason", "seed a terminal state so the control promote to wontfix is a non-self FSM refusal")
	if err != nil {
		t.Fatalf("pre-advancing control to addressed: %v", err)
	}
	if preEnv.Status != "ok" {
		t.Fatalf("pre-advancing control to addressed did not report ok: %+v", preEnv)
	}

	if err := s.Run(dir); err == nil {
		t.Fatal("expected Run to surface the control-promote FSM refusal")
	} else if !strings.Contains(err.Error(), "control promote did not report ok") {
		t.Fatalf("expected the refusal to name the control-promote step, got: %v", err)
	}
}

// TestMidWriteKillScenario_RealBinary_ErrorsWhenTheHangGuardFires
// forces Run's guard branch with a near-zero hangGuard: the target
// promote can neither write nor exit within it, so Run kills the
// subprocess and reports the wedge rather than hanging or silently
// passing.
func TestMidWriteKillScenario_RealBinary_ErrorsWhenTheHangGuardFires(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := t.TempDir()

	s := NewMidWriteKillScenario(bin)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	s.hangGuard = time.Nanosecond

	if err := s.Run(dir); !errors.Is(err, errMidWriteHangGuard) {
		t.Fatalf("expected errMidWriteHangGuard, got: %v", err)
	}
}
