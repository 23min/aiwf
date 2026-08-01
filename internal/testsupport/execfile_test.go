package testsupport

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Counting ETXTBSY under load would be the load-dependent oracle G-0468
// removed from the stress scenarios — it passes on a quiet machine
// whatever the helper does. TestWriteExecutable_TakesForkLock avoids
// that by watching the lock rather than the outcome, so its observation
// is bounded by the lock's state and not by a clock.

// TestWriteExecutable_ProducesAFileThatExecs pins the helper's whole
// contract from the caller's side: the bytes arrive and the result runs
// as a program.
func TestWriteExecutable_ProducesAFileThatExecs(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "stand-in")

	if err := WriteExecutable(path, []byte("#!/bin/sh\necho hello-from-stand-in\n")); err != nil {
		t.Fatalf("WriteExecutable: %v", err)
	}

	out, err := exec.Command(path).Output()
	if err != nil {
		t.Fatalf("running the stand-in: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "hello-from-stand-in" {
		t.Errorf("stand-in printed %q, want %q", got, "hello-from-stand-in")
	}
}

// TestWriteExecutable_SetsTheExecutableBit pins the mode explicitly, so
// a change to executablePerm that still lets /bin/sh scripts run for
// some other reason cannot pass unnoticed.
func TestWriteExecutable_SetsTheExecutableBit(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "stand-in")

	if err := WriteExecutable(path, []byte("#!/bin/sh\nexit 0\n")); err != nil {
		t.Fatalf("WriteExecutable: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != executablePerm {
		t.Errorf("perm = %v, want %v", got, os.FileMode(executablePerm))
	}
}

// TestWriteExecutable_WrapsTheWriteFailureWithThePath pins the error
// arm: a fixture that cannot write its stand-in must say which path it
// failed on, since the caller's own message names only what the stand-in
// was for.
func TestWriteExecutable_WrapsTheWriteFailureWithThePath(t *testing.T) {
	t.Parallel()
	// A path under a file, not a directory: ENOTDIR on every unix.
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("seeding the blocking file: %v", err)
	}
	path := filepath.Join(file, "stand-in")

	err := WriteExecutable(path, []byte("#!/bin/sh\nexit 0\n"))
	if err == nil {
		t.Fatal("WriteExecutable succeeded writing under a regular file, want an error")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not name the path %q", err, path)
	}
}

// TestWriteExecutable_TakesForkLock is the pin that fails if the guard
// is removed: with ForkLock held for writing, a helper that takes it for
// reading cannot complete, so a WriteExecutable that returns has skipped
// the lock.
//
// The assertion is load-invariant in the direction that matters. While
// this test holds the write lock a guarded helper cannot proceed however
// slow the machine is, so a real failure cannot be manufactured by load;
// load can only delay an unguarded write past the window, which costs a
// detection, never a false alarm.
//
// Serial: it blocks every fork in the process while it holds the lock.
// Listed in setup_test.go's skip-list.
func TestWriteExecutable_TakesForkLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stand-in")
	done := make(chan error, 1)

	syscall.ForkLock.Lock()
	go func() { done <- WriteExecutable(path, []byte("#!/bin/sh\nexit 0\n")) }()

	select {
	case err := <-done:
		syscall.ForkLock.Unlock()
		t.Fatalf("WriteExecutable returned (%v) while ForkLock was held for writing; it did not take the read lock", err)
	case <-time.After(250 * time.Millisecond):
		// Still blocked on the read lock, which is the point.
	}

	syscall.ForkLock.Unlock()
	if err := <-done; err != nil {
		t.Fatalf("WriteExecutable after the lock was released: %v", err)
	}
}

// TestWriteExecutable_ReleasesForkLock pins that the helper does not
// leave ForkLock held: a leaked read lock would wedge every subsequent
// fork in the process, turning a fixture bug into a whole-suite hang.
// A second write followed by an exec exercises both, since the exec
// takes the lock for writing.
func TestWriteExecutable_ReleasesForkLock(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, name := range []string{"first", "second"} {
		path := filepath.Join(dir, name)
		if err := WriteExecutable(path, []byte("#!/bin/sh\nexit 0\n")); err != nil {
			t.Fatalf("WriteExecutable(%s): %v", name, err)
		}
		if err := exec.Command(path).Run(); err != nil {
			t.Fatalf("running %s: %v", name, err)
		}
	}
}
