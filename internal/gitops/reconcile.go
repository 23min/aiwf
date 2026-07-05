package gitops

import (
	"context"
	"fmt"
)

// ReconcilePaths stages exactly the given writes' content into the live
// index, so `git status` reports them clean against the commit CommitTree
// just built — without staging, unstaging, or otherwise touching any
// other path. Pairs with CommitTree: CommitTree builds a commit against a
// throwaway index and never reads or writes the live index or worktree;
// ReconcilePaths is the deliberately narrow follow-up step that syncs
// only the paths the commit actually wrote, leaving whatever else the
// caller has staged or modified untouched.
//
// removes mirrors CommitTree's own removes: a rename's vacated old path
// must be evicted from the live index too, or `git status` would keep
// reporting it as present. A remove for a path the live index doesn't
// have is a no-op, not an error.
//
// Each write's content is hashed into the object database (an identical
// hash-object call already ran inside CommitTree for the same content —
// git is content-addressed, so this is a cheap no-op repeat, not a
// duplicate write) and staged via `update-index --add --cacheinfo`
// against the real index, one path at a time — a failure partway through
// leaves every already-processed path reconciled rather than aborting
// the whole batch.
func ReconcilePaths(ctx context.Context, workdir string, removes []string, writes []PathWrite) error {
	for _, path := range removes {
		err := run(ctx, workdir, "update-index", "--force-remove", path)
		if err != nil {
			return fmt.Errorf("removing %s: %w", path, err)
		}
	}
	for _, w := range writes {
		blobSHA, err := hashObject(ctx, workdir, w.Content)
		if err != nil {
			return fmt.Errorf("hashing blob for %s: %w", w.Path, err)
		}
		cacheInfo := fmt.Sprintf("100644,%s,%s", blobSHA, w.Path)
		err = run(ctx, workdir, "update-index", "--add", "--cacheinfo", cacheInfo)
		if err != nil {
			return fmt.Errorf("update-index %s: %w", w.Path, err)
		}
	}
	return nil
}
