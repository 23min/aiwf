package gitops

import "context"

// FetchAll runs `git fetch --all` in workdir, refreshing every
// remote-tracking ref (refs/remotes/*) across all configured remotes.
// Wraps git failures; the caller decides whether a failure is fatal or,
// as in the allocator's opt-in `--fetch` (M-0214), best-effort.
//
// This is the broadened successor to M-0213's single-branch trunk fetch:
// the allocator's remote-side view now spans every remote-tracking ref
// (trunk.RemoteRefIDs), so `--fetch` refreshes all of them, not just the
// trunk branch. With no remotes configured it is a clean no-op (git
// exits 0) — nothing to fetch.
func FetchAll(ctx context.Context, workdir string) error {
	return run(ctx, workdir, "fetch", "--all")
}
