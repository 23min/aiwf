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

// CodeRepoLockBusy and CodeRepoLockAcquireFailed are the error-envelope
// codes AcquireRepoLock reports its two refusal paths under, so a
// --format=json consumer tells them apart by identity rather than by
// matching the message text: repo-lock-busy means another aiwf process
// holds the lock and the caller should back off and retry, while
// repo-lock-acquire-failed means the lock could not be taken at all —
// the lockfile could not be located, opened or locked — which no amount
// of retrying fixes.
//
// They are plain string constants rather than codes.Code descriptors
// because neither is a legality refusal named by a workflow-spec cell —
// D-0011 scopes the descriptor form to that set and leaves every other
// code a bare string until a consumer needs its class.
//
// The envelope's code and the process exit code are independent axes:
// the code says which refusal this is, the exit code says which bucket
// it falls in. These two keep their established exit codes (busy →
// [ExitUsage], acquire-failure → [ExitInternal]); what bucket an
// *uncoded* error belongs in is a separate question, open in G-0483.
const (
	CodeRepoLockBusy          = "repo-lock-busy"
	CodeRepoLockAcquireFailed = "repo-lock-acquire-failed"
)

// AcquireRepoLock takes the per-repo mutation lock and returns a
// release function plus a zero exit code on success. On failure it
// reports the refusal via out — the conventional text stderr line, or
// a JSON error envelope carrying the refusal's own code when out
// requests --format=json — and returns a non-zero exit code the caller
// must propagate (release will be nil). This is the shared chokepoint
// every mutating verb calls before doing any work, so honoring the
// requested format and code here is what makes lock contention
// scriptable at all (G-0391, G-0467).
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
// zero value OutputFormat{}, which renders as plain text.
//
// Read-only verbs must NOT call this — they can run concurrently with
// mutations, seeing at worst a snapshot from before the mutation lands.
// R-AUDIT-0133 in docs/design/legal-workflows-audit.md enumerates which
// verbs those are; internal/policies' readOnlyVerbs covers the subset
// with a mechanical chokepoint, and polices direct disk writes rather
// than this call, so the must-not above rests on review.
func AcquireRepoLock(rootDir, verbDisplay string, out OutputFormat) (release func(), rc int) {
	lock, err := repolock.Acquire(rootDir, lockTimeout)
	if err != nil {
		if errors.Is(err, repolock.ErrBusy) {
			out.emitErrorEnvelope(verbDisplay, CodeRepoLockBusy, fmt.Sprintf("%v; retry in a moment", repolock.ErrBusy))
			return nil, ExitUsage
		}
		out.emitErrorEnvelope(verbDisplay, CodeRepoLockAcquireFailed, fmt.Sprintf("acquiring repo lock: %v", err))
		return nil, ExitInternal
	}
	return func() { _ = lock.Release() }, 0
}
