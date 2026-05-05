//go:build windows

// Windows stub for repolock. The binary cross-compiles cleanly so
// the cmd/aiwf entry point's assertSupportedOS check can refuse to
// run with one clear message; this file just satisfies the type
// contract and never holds an actual lock.
package repolock

import (
	"errors"
	"time"
)

// ErrBusy mirrors the Unix package's sentinel; on Windows it's
// only returned via the shared error path. ErrUnsupported names
// the platform refusal explicitly.
var (
	ErrBusy        = errors.New("another aiwf process is running on this repo")
	ErrUnsupported = errors.New("repolock: Windows is not supported (POSIX flock not available); aiwf refuses to run on Windows up front")
)

// Lock is the type the rest of the engine expects; on Windows it
// carries no state and Release is a no-op.
type Lock struct{}

// Acquire on Windows always errors. Callers should never reach this
// in practice — assertSupportedOS gates execution before any verb
// touches a Lock — but the function exists so the package compiles.
func Acquire(root string, timeout time.Duration) (*Lock, error) {
	return nil, ErrUnsupported
}

// Release is a no-op on Windows.
func (l *Lock) Release() error { return nil }
