package verb

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
)

// Apply executes a verb's Plan against the consumer repo at root: it
// runs every OpMove via `git mv`, every OpWrite directly to disk
// (creating parent directories as needed), stages the writes with
// `git add`, then creates the single commit with the plan's subject
// and trailers.
//
// Moves run before writes so that when a verb (notably reallocate)
// renames a file/dir and also rewrites files inside that dir, the
// writes land at the new locations.
//
// Atomicity: Apply is all-or-nothing. If any step after the first
// mutation fails (write error, commit failure, panic), the worktree
// and index are restored to their pre-Apply state via a deferred
// rollback. The repo ends up exactly as if Apply had never been
// called. This preserves the framework's "every mutating verb
// produces exactly one git commit" guarantee under partial failure.
func Apply(ctx context.Context, root string, p *Plan) (err error) {
	tx := &applyTx{root: root, ctx: ctx}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.rollback()
			panic(r)
		}
		if err != nil {
			if rbErr := tx.rollback(); rbErr != nil { //coverage:ignore defensive: requires both primary error and rollback failure simultaneously
				err = fmt.Errorf("%w (rollback also failed: %v — manual cleanup may be needed)", err, rbErr)
			}
		}
	}()

	// Phase 1: moves.
	for _, op := range p.Ops {
		if op.Type != OpMove {
			continue
		}
		if mvErr := gitops.Mv(ctx, root, op.Path, op.NewPath); mvErr != nil {
			return fmt.Errorf("git mv %s -> %s: %w", op.Path, op.NewPath, mvErr)
		}
		tx.touchedPaths = append(tx.touchedPaths, op.Path, op.NewPath)
	}

	// Phase 2: writes.
	writtenPaths := []string{}
	for _, op := range p.Ops {
		if op.Type != OpWrite {
			continue
		}
		full := filepath.Join(root, op.Path)
		preexisted := fileExists(full)
		if mkdirErr := os.MkdirAll(filepath.Dir(full), 0o755); mkdirErr != nil {
			return fmt.Errorf("creating %s: %w", filepath.Dir(op.Path), mkdirErr)
		}
		if writeErr := os.WriteFile(full, op.Content, 0o644); writeErr != nil {
			return fmt.Errorf("writing %s: %w", op.Path, writeErr)
		}
		writtenPaths = append(writtenPaths, op.Path)
		tx.touchedPaths = append(tx.touchedPaths, op.Path)
		if !preexisted {
			tx.createdFiles = append(tx.createdFiles, op.Path)
		}
	}

	if len(writtenPaths) > 0 {
		if addErr := gitops.Add(ctx, root, writtenPaths...); addErr != nil { //coverage:ignore defensive against git CLI failure; not reachable from a clean repo + valid op set
			return fmt.Errorf("git add: %w", addErr)
		}
	}

	if commitErr := gitops.Commit(ctx, root, p.Subject, p.Body, p.Trailers); commitErr != nil {
		return fmt.Errorf("git commit: %w", commitErr)
	}
	tx.committed = true
	return nil
}

// applyTx tracks the paths a partial Apply has touched so the
// deferred rollback can restore the repo to its pre-call state.
type applyTx struct {
	root         string
	ctx          context.Context
	touchedPaths []string // every path that may need restoring (sources + dests)
	createdFiles []string // brand-new files that didn't exist at HEAD; remove on rollback
	committed    bool     // when true, rollback is a no-op
}

// rollback restores the worktree and index to HEAD for every touched
// path, then removes any brand-new files. Safe to call multiple
// times. Returns the first non-nil error encountered.
func (t *applyTx) rollback() error {
	if t == nil || t.committed {
		return nil
	}
	if len(t.touchedPaths) == 0 && len(t.createdFiles) == 0 {
		return nil
	}
	dedup := dedupePaths(t.touchedPaths)
	// `git restore --staged --worktree -- <paths>` undoes index +
	// worktree changes for tracked paths. For paths that didn't
	// exist at HEAD (newly created files) git restore yields a
	// "pathspec did not match" warning but still resets staged
	// state for the existing paths. We then explicitly remove the
	// new files from the worktree and from the index.
	var firstErr error
	if rErr := restorePaths(t.ctx, t.root, dedup); rErr != nil {
		firstErr = rErr
	}
	for _, p := range t.createdFiles {
		full := filepath.Join(t.root, p)
		if rmErr := os.Remove(full); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) && firstErr == nil {
			firstErr = fmt.Errorf("removing %s: %w", p, rmErr)
		}
	}
	return firstErr
}

// fileExists reports whether path resolves to a regular file at the
// time of the call. Used to distinguish "writing to an existing
// tracked file" from "creating a brand-new file" so rollback knows
// whether to remove the file or just unstage it.
func fileExists(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// dedupePaths removes duplicates while preserving order.
func dedupePaths(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, p := range in {
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

// restorePaths runs `git restore --staged --worktree -- <paths>` to
// reset the index and worktree to HEAD for every path. Brand-new
// paths produce a pathspec warning that we ignore — they are
// unstaged separately, and the worktree file is removed by the
// caller.
func restorePaths(ctx context.Context, root string, paths []string) error {
	return gitops.Restore(ctx, root, paths...)
}
