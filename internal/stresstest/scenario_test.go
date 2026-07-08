package stresstest

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// fakeScenario is a minimal, configurable Scenario used only to drive
// RunScenario's cleanup discipline. Real scenario content (tiers 1-5)
// is out of scope for this milestone.
type fakeScenario struct {
	setupErr   error
	runErr     error
	violations []Violation
}

func (f *fakeScenario) Setup(dir string) error        { return f.setupErr }
func (f *fakeScenario) Run(dir string) error          { return f.runErr }
func (f *fakeScenario) Verify(dir string) []Violation { return f.violations }

func TestRunScenario_CleansUpOnPass(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	result, err := RunScenario(&fakeScenario{}, base)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if !result.Passed {
		t.Fatal("expected Passed to be true for a scenario with no error and no violations")
	}
	if result.Dir != "" {
		t.Fatalf("expected Dir to be empty after a passing scenario cleans up, got %q", result.Dir)
	}

	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("read base dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected the scenario's temp dir to be removed from disk, base dir still has: %v", entries)
	}
}

func TestRunScenario_PreservesDirOnVerifyViolations(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	result, err := RunScenario(&fakeScenario{violations: []Violation{{Message: "invariant broken"}}}, base)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if result.Passed {
		t.Fatal("expected Passed to be false when Verify reports a violation")
	}
	if result.Dir == "" {
		t.Fatal("expected Dir to name the preserved scenario directory")
	}
	if len(result.Violations) != 1 || result.Violations[0].Message != "invariant broken" {
		t.Fatalf("unexpected violations: %+v", result.Violations)
	}
	if _, err := os.Stat(result.Dir); err != nil {
		t.Fatalf("expected the scenario dir to survive on disk for RCA, stat failed: %v", err)
	}
}

func TestRunScenario_PreservesDirOnSetupError(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	result, err := RunScenario(&fakeScenario{setupErr: errors.New("setup failed")}, base)
	if err == nil {
		t.Fatal("expected RunScenario to propagate a Setup error")
	}
	if result.Dir == "" {
		t.Fatal("expected Dir to name the preserved scenario directory on a Setup failure")
	}
	if _, statErr := os.Stat(result.Dir); statErr != nil {
		t.Fatalf("expected the scenario dir to survive on disk, stat failed: %v", statErr)
	}
}

func TestRunScenario_PreservesDirOnRunError(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	result, err := RunScenario(&fakeScenario{runErr: errors.New("run failed")}, base)
	if err == nil {
		t.Fatal("expected RunScenario to propagate a Run error")
	}
	if result.Dir == "" {
		t.Fatal("expected Dir to name the preserved scenario directory on a Run failure")
	}
	if _, statErr := os.Stat(result.Dir); statErr != nil {
		t.Fatalf("expected the scenario dir to survive on disk, stat failed: %v", statErr)
	}
}

func TestRunScenario_ErrorsWhenBaseDirMissing(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	if _, err := RunScenario(&fakeScenario{}, missing); err == nil {
		t.Fatal("expected RunScenario to error when baseDir doesn't exist")
	}
}

// blockingCleanupScenario's Setup populates the scenario dir with a
// child entry, then revokes write permission on the dir itself — a
// directory entry can only be unlinked with write permission on its
// parent, so this forces RunScenario's final os.RemoveAll to fail
// deterministically, without relying on running as a privileged user.
type blockingCleanupScenario struct{}

func (blockingCleanupScenario) Setup(dir string) error {
	child := filepath.Join(dir, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(child, "f"), []byte("x"), 0o644); err != nil {
		return err
	}
	return os.Chmod(dir, 0o500)
}

func (blockingCleanupScenario) Run(dir string) error          { return nil }
func (blockingCleanupScenario) Verify(dir string) []Violation { return nil }

func TestRunScenario_ErrorsWhenCleanupFails(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("permission-based RemoveAll-failure simulation is unix-only")
	}
	base := t.TempDir()

	result, err := RunScenario(blockingCleanupScenario{}, base)
	t.Cleanup(func() {
		if result.Dir != "" {
			_ = os.Chmod(result.Dir, 0o755)
		}
	})

	if err == nil {
		t.Fatal("expected RunScenario to surface a RemoveAll failure")
	}
	if !result.Passed {
		t.Fatal("expected Passed to be true — Verify found no violations, only cleanup failed")
	}
	if result.Dir == "" {
		t.Fatal("expected Dir to name the directory whose cleanup failed")
	}
}
