//go:build !windows

// Package repolock serializes mutating aiwf invocations on the same
// repository. The lock is held for the duration of a single verb
// (read tree → validate → write → commit) so concurrent allocators
// can't pick the same id.
//
// Implementation: a POSIX advisory file lock (flock(2)) on
// <root>/.git/aiwf.lock. The lockfile is created on first Acquire
// and never removed — the lock lives on the open file descriptor,
// not on the file's existence. Crashed processes release the lock
// via the kernel's fd cleanup, so stale lockfiles never block a
// future invocation.
//
// Read-only verbs (check, history, status, render, doctor) do not
// acquire the lock — they can safely run concurrently with each
// other and with mutations (the worst they see is a snapshot
// from before the mutation lands).
//
// Windows: a separate stub in repolock_windows.go satisfies the
// type contract so the rest of the binary cross-compiles, but
// every Acquire returns an error. The cmd/aiwf entry point's
// assertSupportedOS check refuses to run on Windows up front, so
// users see one clear message instead of a deep stack failure.
package repolock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// ErrBusy is returned by Acquire when another process holds the
// lock and the timeout elapsed without it being released.
var ErrBusy = errors.New("another aiwf process is running on this repo")

// Lock is a held repolock. Callers must Release it (typically via
// defer) before the process exits, although the kernel will release
// it automatically on exit if Release is missed.
type Lock struct {
	f    *os.File
	path string
}

// Acquire takes an exclusive lock on root's aiwf lockfile. If
// another process holds the lock, Acquire polls until the lock is
// released or timeout elapses. A zero timeout returns ErrBusy
// immediately if the lock is held; otherwise the lock is taken and
// returned.
//
// The lockfile lives at <root>/.git/aiwf.lock when .git/ exists
// (the default for any aiwf-managed repo); otherwise at
// <root>/.aiwf.lock as a fallback for tests and for tooling
// integration before `git init`.
func Acquire(root string, timeout time.Duration) (*Lock, error) {
	path, err := lockfilePath(root)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	deadline := time.Now().Add(timeout)
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) //nolint:gosec // f.Fd is a small fd; the int conversion is safe for any value Go's runtime returns
		if err == nil {
			return &Lock{f: f, path: path}, nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) { //coverage:ignore defensive: any non-EWOULDBLOCK flock error indicates a corrupted fd
			_ = f.Close()
			return nil, fmt.Errorf("locking %s: %w", path, err)
		}
		if time.Now().After(deadline) || timeout == 0 {
			_ = f.Close()
			return nil, ErrBusy
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Release frees the lock. Idempotent: a second Release is a no-op.
func (l *Lock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	err := syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN) //nolint:gosec // see Acquire: fd values fit in int by construction
	if cerr := l.f.Close(); cerr != nil && err == nil {  //coverage:ignore defensive: file.Close on a regular file rarely fails
		err = cerr
	}
	l.f = nil
	return err
}

// lockfilePath returns the path of the aiwf lockfile for root.
// Prefers <root>/.git/aiwf.lock so the lockfile is naturally
// outside the project tree (and unaffected by gitignore rules).
// Falls back to <root>/.aiwf.lock when .git/ is absent.
func lockfilePath(root string) (string, error) {
	gitDir := filepath.Join(root, ".git")
	info, err := os.Stat(gitDir)
	if err == nil && info.IsDir() {
		return filepath.Join(gitDir, "aiwf.lock"), nil
	}
	// Fall back to a top-level lockfile when there's no .git/.
	if _, err := os.Stat(root); err != nil {
		return "", fmt.Errorf("locating lockfile: %w", err)
	}
	return filepath.Join(root, ".aiwf.lock"), nil
}
