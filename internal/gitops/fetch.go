package gitops

import "context"

// FetchBranch runs `git fetch <remote> <branch>` in workdir, refreshing
// only that single branch's remote-tracking ref (refs/remotes/<remote>/<branch>
// in a standard clone) — not a full `git fetch --all`. Wraps git
// failures; the caller decides whether a failure is fatal or, as in the
// allocator's opt-in `--fetch` (M-0213), best-effort.
//
// The narrow single-branch fetch is deliberate: the allocator only needs
// the configured trunk ref refreshed, and a `--all` would pull every
// branch and tag the operator didn't ask for.
func FetchBranch(ctx context.Context, workdir, remote, branch string) error {
	return run(ctx, workdir, "fetch", remote, branch)
}
