package stresstest

import (
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

// TestWaitForTempFile_RealBinary pins waitForTempFile's two direct
// outcomes: it finds a temp file that already exists, and it times
// out when none ever appears.
func TestWaitForTempFile_RealBinary(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	found, err := waitForTempFile(dir, timeoutForTest)
	if err != nil {
		t.Fatalf("waitForTempFile on an empty dir: %v", err)
	}
	if found {
		t.Fatal("expected not to find a temp file in an empty dir")
	}

	if writeErr := os.WriteFile(filepath.Join(dir, "entity.md.aiwf-tmp-12345"), []byte("x"), 0o644); writeErr != nil {
		t.Fatalf("seeding a temp file: %v", writeErr)
	}
	found, err = waitForTempFile(dir, timeoutForTest)
	if err != nil {
		t.Fatalf("waitForTempFile with a temp file present: %v", err)
	}
	if !found {
		t.Fatal("expected to find the seeded temp file")
	}
}

// TestWaitForTempFile_ErrorsOnUnreadableDir pins the os.ReadDir error
// branch via a nonexistent directory.
func TestWaitForTempFile_ErrorsOnUnreadableDir(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	if _, err := waitForTempFile(missing, timeoutForTest); err == nil {
		t.Fatal("expected waitForTempFile to error on a nonexistent directory")
	}
}

// TestReadGapFile_ErrorsWhenNoneOrMultipleMatch pins readGapFile's
// count-mismatch branch (zero matches; more than one match).
func TestReadGapFile_ErrorsWhenNoneOrMultipleMatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		files []string
	}{
		{name: "zero matches", files: nil},
		{name: "two matches", files: []string{"G-0001-a.md", "G-0001-b.md"}},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			gapsDir := filepath.Join(root, "work", "gaps")
			if err := os.MkdirAll(gapsDir, 0o755); err != nil {
				t.Fatalf("mkdir gapsDir: %v", err)
			}
			for _, f := range tc.files {
				if err := os.WriteFile(filepath.Join(gapsDir, f), []byte("x"), 0o644); err != nil {
					t.Fatalf("seeding %s: %v", f, err)
				}
			}
			if _, err := readGapFile(root, "G-0001"); err == nil {
				t.Fatalf("expected readGapFile to error for %s", tc.name)
			}
		})
	}
}

const timeoutForTest = 50 * time.Millisecond

// TestMidWriteKillScenario_RealBinary_RunSurfacesAControlPromoteLaunchRefusal
// holds the control repo's repolock via the AC-1 lockholder helper
// before calling Run: `aiwf promote`'s lock-busy refusal
// (internal/cli/cliutil.AcquireRepoLock) prints a plain-text message
// and exits without ever emitting a --format=json envelope (G-0391) —
// so runAiwfJSON's own JSON-parse step fails, pinning that Run wraps
// and surfaces that failure via the "running the control promote"
// step, not a parsed non-ok envelope.
func TestMidWriteKillScenario_RealBinary_RunSurfacesAControlPromoteLaunchRefusal(t *testing.T) {
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
	} else if !strings.Contains(err.Error(), "running the control promote") {
		t.Fatalf("expected the failure to name the control-promote step, got: %v", err)
	}

	_ = stdinW.Close()
}

// TestMidWriteKillScenario_RealBinary_RunSurfacesAControlPromoteFSMRefusal
// pre-advances the control repo's gap to "wontfix" (terminal, no
// outgoing transitions) before calling Run, so Run's own internal
// "promote to wontfix" attempt is refused as FSM-illegal — a refusal
// that (unlike the lock-busy path above) DOES emit a valid
// --format=json envelope with status "error", pinning Run's
// promoteEnv.Status != "ok" branch specifically.
func TestMidWriteKillScenario_RealBinary_RunSurfacesAControlPromoteFSMRefusal(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := t.TempDir()

	s := NewMidWriteKillScenario(bin)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	preEnv, err := runAiwfJSON(bin, filepath.Join(dir, "control"), "promote", "G-0001", "wontfix")
	if err != nil {
		t.Fatalf("pre-advancing control to wontfix: %v", err)
	}
	if preEnv.Status != "ok" {
		t.Fatalf("pre-advancing control to wontfix did not report ok: %+v", preEnv)
	}

	if err := s.Run(dir); err == nil {
		t.Fatal("expected Run to surface the control-promote FSM refusal")
	} else if !strings.Contains(err.Error(), "control promote did not report ok") {
		t.Fatalf("expected the refusal to name the control-promote step, got: %v", err)
	}
}

// TestMidWriteKillScenario_RealBinary_ErrorsOnReadyTimeout forces
// Run's own "never observed the temp file" branch with a near-zero
// readyTimeout — the target promote's write can't possibly land
// within it, so Run kills the (probably not-yet-writing) subprocess
// and reports the timeout, rather than hanging or silently passing.
func TestMidWriteKillScenario_RealBinary_ErrorsOnReadyTimeout(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	dir := t.TempDir()

	s := NewMidWriteKillScenario(bin)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	s.readyTimeout = time.Nanosecond

	if err := s.Run(dir); err == nil {
		t.Fatal("expected Run to time out waiting to observe the sibling temp file")
	} else if !strings.Contains(err.Error(), "timed out waiting") {
		t.Fatalf("expected a timeout error, got: %v", err)
	}
}
