package cliutil

import (
	"errors"
	"fmt"
	"time"

	"github.com/23min/aiwf/internal/repolock"
)

// lockTimeout is how long a mutating verb waits for the repo lock
// before returning the busy-finding. Two seconds matches the human
// expectation of "another aiwf invocation is winding down".
const lockTimeout = 2 * time.Second

// AcquireRepoLock takes the per-repo mutation lock and returns a
// release function plus a zero exit code on success. On failure it
// reports the refusal via out — the conventional text stderr line, or
// a JSON error envelope when out requests --format=json (G-0391: this
// is the shared chokepoint every mutating verb calls before doing any
// work, so a caller scripting against --format=json got a plain-text
// stderr line and empty stdout on lock contention, regardless of the
// format it asked for) — and returns a non-zero exit code the caller
// must propagate (release will be nil).
//
// Usage in every mutating verb:
//
//	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf add", out)
//	if release == nil {
//	    return rc
//	}
//	defer release()
//
// A verb with no --format flag of its own (init, update) passes the
// zero value OutputFormat{}, which behaves exactly as plain text
// always did.
//
// Read-only verbs (check, history, status, render, doctor, whoami)
// must NOT call this — they can run concurrently with mutations.
func AcquireRepoLock(rootDir, verbDisplay string, out OutputFormat) (release func(), rc int) {
	lock, err := repolock.Acquire(rootDir, lockTimeout)
	if err != nil {
		if errors.Is(err, repolock.ErrBusy) {
			out.emitErrorEnvelope(verbDisplay, "", "another aiwf process is running on this repo; retry in a moment")
			return nil, ExitUsage
		}
		out.emitErrorEnvelope(verbDisplay, "", fmt.Sprintf("acquiring repo lock: %v", err))
		return nil, ExitInternal
	}
	return func() { _ = lock.Release() }, 0
}
