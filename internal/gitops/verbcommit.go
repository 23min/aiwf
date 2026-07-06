package gitops

import (
	"context"
	"fmt"
)

// ReconcileError reports that CommitVerbChange's commit landed but the
// post-commit reconciliation into the live index failed. SHA names the
// commit that landed — git history already exists — so a caller must
// treat this differently from an outright commit failure (nothing to
// roll back; the fix is re-running the reconciliation, not retrying the
// commit).
type ReconcileError struct {
	SHA string
	Err error
}

func (e *ReconcileError) Error() string {
	return fmt.Sprintf("commit %s landed but reconciling the live index failed: %v", e.SHA, e.Err)
}

func (e *ReconcileError) Unwrap() error { return e.Err }

// CommitVerbChange runs the full verb-commit sequence in one call:
// CommitTree builds the commit against a throwaway index, the
// post-commit hook fires (best-effort, matching git's own tolerance for
// that hook), then ReconcilePaths syncs the written paths into the live
// index. This is the one exported entry point for commit-construction
// (M-0186/AC-5) — a future verb-commit consumer reuses this sequence
// rather than re-deriving the ordering of its three underlying steps.
//
// A failure building the commit itself returns before anything lands
// (sha is empty, err is the plain CommitTree error). A failure during
// reconciliation returns as *ReconcileError: the commit already exists
// in git history, so the caller must not treat it as "nothing
// happened."
func CommitVerbChange(ctx context.Context, workdir string, removes []string, writes []PathWrite, subject, body string, trailers []Trailer) (sha string, err error) {
	sha, err = CommitTree(ctx, workdir, removes, writes, subject, body, trailers)
	if err != nil {
		return "", err
	}

	// CommitTree is plumbing (commit-tree + update-ref) — it fires no
	// git hooks at all, unlike the `git commit` porcelain it replaces.
	// Firing post-commit explicitly restores parity for consumers that
	// rely on it (e.g. STATUS.md regeneration, G-0112). Best-effort,
	// matching git's own tolerance for this hook: its exit status is
	// informational only.
	_ = RunPostCommitHook(ctx, workdir)

	if reconcileErr := ReconcilePaths(ctx, workdir, removes, writes); reconcileErr != nil {
		return sha, &ReconcileError{SHA: sha, Err: reconcileErr}
	}
	return sha, nil
}
