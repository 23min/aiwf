package stresstest

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// mid_write_kill_observe_test.go — the decision half of
// MidWriteKillScenario's observation: when one attempt stops watching,
// and what it concludes. Nothing here starts a process or waits on
// one, so it belongs in the untagged lane that runs on every push,
// unlike the scenario driver in mid_write_kill_test.go.

// timeoutForObserve is a hang guard short enough to fire inside a
// test. The cases that must not reach it end on their own evidence —
// a file already on disk, an exit already delivered — so shortening it
// cannot make them flake.
const timeoutForObserve = 50 * time.Millisecond

// TestTempFilePresent pins the scan's two verdicts: an empty directory
// holds no temp file, and one matching AtomicWriteFile's naming
// convention is recognized.
func TestTempFilePresent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	present, err := tempFilePresent(dir)
	if err != nil {
		t.Fatalf("tempFilePresent on an empty dir: %v", err)
	}
	if present {
		t.Fatal("expected not to find a temp file in an empty dir")
	}

	if writeErr := os.WriteFile(filepath.Join(dir, "entity.md.aiwf-tmp-12345"), []byte("x"), 0o644); writeErr != nil {
		t.Fatalf("seeding a temp file: %v", writeErr)
	}
	present, err = tempFilePresent(dir)
	if err != nil {
		t.Fatalf("tempFilePresent with a temp file present: %v", err)
	}
	if !present {
		t.Fatal("expected to find the seeded temp file")
	}
}

// TestTempFilePresent_ErrorsOnUnreadableDir pins the os.ReadDir error
// branch via a nonexistent directory.
func TestTempFilePresent_ErrorsOnUnreadableDir(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	if _, err := tempFilePresent(missing); err == nil {
		t.Fatal("expected tempFilePresent to error on a nonexistent directory")
	}
}

// TestObserveTempFile pins the three ways one observation ends: the
// temp file appears, the watched process exits first, and neither
// happens before the hang guard. The exit-first case is the one the
// retry rests on — it must report a miss rather than an error, and
// hand back the process's own exit result.
func TestObserveTempFile(t *testing.T) {
	t.Parallel()

	t.Run("temp file appears", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "entity.md.aiwf-tmp-1"), []byte("x"), 0o644); err != nil {
			t.Fatalf("seeding a temp file: %v", err)
		}
		// An open channel stands in for a process still running.
		observed, exitErr, err := observeTempFile(dir, make(chan error), timeoutForObserve)
		if err != nil || exitErr != nil {
			t.Fatalf("observeTempFile = (_, %v, %v), want no errors", exitErr, err)
		}
		if !observed {
			t.Error("expected the seeded temp file to be observed")
		}
	})

	t.Run("process exits first", func(t *testing.T) {
		t.Parallel()
		exited := make(chan error, 1)
		wantExit := errors.New("promote failed")
		exited <- wantExit

		observed, exitErr, err := observeTempFile(t.TempDir(), exited, timeoutForObserve)
		if err != nil {
			t.Fatalf("observeTempFile errored on a clean exit: %v", err)
		}
		if observed {
			t.Error("expected a miss when the process exits before the temp file appears")
		}
		if !errors.Is(exitErr, wantExit) {
			t.Errorf("exitErr = %v, want the process's own %v", exitErr, wantExit)
		}
	})

	t.Run("unreadable directory", func(t *testing.T) {
		t.Parallel()
		missing := filepath.Join(t.TempDir(), "does-not-exist")

		observed, _, err := observeTempFile(missing, make(chan error), timeoutForObserve)
		if err == nil || errors.Is(err, errMidWriteHangGuard) {
			t.Fatalf("err = %v, want the directory read failure rather than the guard", err)
		}
		if observed {
			t.Error("expected no observation when the directory cannot be read")
		}
	})

	t.Run("hang guard fires", func(t *testing.T) {
		t.Parallel()
		observed, _, err := observeTempFile(t.TempDir(), make(chan error), timeoutForObserve)
		if !errors.Is(err, errMidWriteHangGuard) {
			t.Fatalf("err = %v, want errMidWriteHangGuard", err)
		}
		if observed {
			t.Error("expected no observation when the guard fires")
		}
	})
}
