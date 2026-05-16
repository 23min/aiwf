package cliutil

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/23min/aiwf/internal/repolock"
)

// lockTimeout is how long a mutating verb waits for the repo lock
// before returning the busy-finding. Two seconds matches the human
// expectation of "another aiwf invocation is winding down".
const lockTimeout = 2 * time.Second

// AcquireRepoLock takes the per-repo mutation lock and returns a
// release function plus a zero exit code on success. On failure it
// prints an explanation to stderr and returns a non-zero exit code
// the caller must propagate (release will be nil).
//
// Usage in every mutating verb:
//
//	release, rc := cliutil.AcquireRepoLock(rootDir, "aiwf add")
//	if release == nil {
//	    return rc
//	}
//	defer release()
//
// Read-only verbs (check, history, status, render, doctor, whoami)
// must NOT call this — they can run concurrently with mutations.
func AcquireRepoLock(rootDir, verbDisplay string) (release func(), rc int) {
	lock, err := repolock.Acquire(rootDir, lockTimeout)
	if err != nil {
		if errors.Is(err, repolock.ErrBusy) {
			fmt.Fprintf(os.Stderr,
				"%s: another aiwf process is running on this repo; retry in a moment\n",
				verbDisplay)
			return nil, ExitUsage
		}
		fmt.Fprintf(os.Stderr, "%s: acquiring repo lock: %v\n", verbDisplay, err)
		return nil, ExitInternal
	}
	return func() { _ = lock.Release() }, 0
}
