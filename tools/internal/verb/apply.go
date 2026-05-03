package verb

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
			return classifyGitError(ctx, root, fmt.Sprintf("git mv %s -> %s", op.Path, op.NewPath), mvErr)
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
		if addErr := gitops.Add(ctx, root, writtenPaths...); addErr != nil {
			return classifyGitError(ctx, root, "git add", addErr)
		}
	}

	commit := gitops.Commit
	if p.AllowEmpty {
		commit = gitops.CommitAllowEmpty
	}
	if commitErr := commit(ctx, root, p.Subject, p.Body, p.Trailers); commitErr != nil {
		return classifyGitError(ctx, root, "git commit", commitErr)
	}
	tx.committed = true
	return nil
}

// classifyGitError inspects a git CLI failure (mv, add, or commit)
// and, when the underlying cause is `.git/index.lock` contention
// from another process (a file watcher, an editor's git extension,
// or a stale lock from a prior crash), wraps the error with
// diagnostic detail and a hint pointing at the G24 audit-only
// recovery path. The classification fires on every git step Apply
// runs, since any of them can hit the lock first.
//
// Lock-holder lookup is best-effort: if `lsof` is missing, exits
// non-zero, or returns no lines, the function falls back to the
// bare error. The kernel never blocks the user on diagnostic
// gathering, and never silently retries — silent retries hide real
// environmental problems and can race against the holder.
//
// Reference: docs/pocv3/plans/provenance-model-plan.md §"Step 5c"
// and docs/pocv3/gaps.md G24.
func classifyGitError(ctx context.Context, root, step string, gitErr error) error {
	if !isIndexLockError(gitErr.Error()) {
		return fmt.Errorf("%s: %w", step, gitErr)
	}
	hint := lockContentionHint(ctx, root)
	if hint == "" {
		return fmt.Errorf(
			"%s failed due to .git/index.lock contention\n"+
				"  another process holds the index lock; wait for it to finish, kill the holder,\n"+
				"  or — if you completed the work manually — re-run with `--audit-only --reason \"...\"`\n"+
				"  underlying error: %w",
			step, gitErr,
		)
	}
	return fmt.Errorf(
		"%s failed due to .git/index.lock contention\n"+
			"  %s\n"+
			"  wait for the holder to finish, kill it, or — if you completed the work manually —\n"+
			"  re-run with `--audit-only --reason \"...\"`\n"+
			"  underlying error: %w",
		step, hint, gitErr,
	)
}

// isIndexLockError reports whether the error string from a failed
// `git commit` indicates `.git/index.lock` contention. Git's exact
// wording varies across versions; we match on the load-bearing
// substrings without anchoring on a full message template.
//
// Path separator: git on every platform (including Windows) emits
// forward-slash paths in its diagnostic messages — that's part of
// git's porcelain stability promise. We still accept backslash
// defensively so a future deviation doesn't silently mis-route
// the lock-contention path back to the generic-error branch.
func isIndexLockError(msg string) bool {
	if strings.Contains(msg, ".git/index.lock") ||
		strings.Contains(msg, `.git\index.lock`) ||
		strings.Contains(msg, "index.lock") {
		return true
	}
	// Older git renders "Unable to create '<path>': File exists."
	if strings.Contains(msg, "Unable to create") && strings.Contains(msg, "lock") {
		return true
	}
	return false
}

// lockContentionHint returns a one-line diagnostic naming the
// process holding `.git/index.lock`, when discoverable. Returns the
// empty string when `lsof` is missing or yields no parseable output —
// the caller falls back to a bare hint in that case.
//
// Resolves the actual git-dir via `git rev-parse --absolute-git-dir`
// so worktrees and submodules point at the right lock file (their
// `.git` is a regular file, not a directory).
func lockContentionHint(ctx context.Context, root string) string {
	gitDir, err := gitops.GitDir(ctx, root)
	if err != nil {
		gitDir = filepath.Join(root, ".git")
	}
	lockPath := filepath.Join(gitDir, "index.lock")
	if _, statErr := os.Stat(lockPath); statErr != nil {
		// The lock cleared between commit failure and our diagnostic
		// — race, but harmless; nothing to report.
		return ""
	}
	if _, lookErr := exec.LookPath("lsof"); lookErr != nil {
		return ""
	}
	cmd := exec.CommandContext(ctx, "lsof", lockPath)
	out, runErr := cmd.Output()
	if runErr != nil {
		return ""
	}
	pid, name := parseLsof(string(out))
	if pid == "" {
		return ""
	}
	if name == "" {
		return fmt.Sprintf("lock holder: PID %s", pid)
	}
	return fmt.Sprintf("lock holder: PID %s (%s)", pid, name)
}

// parseLsof extracts the PID and process name from `lsof <path>`
// output. Format (per lsof(8)):
//
//	COMMAND   PID  USER ...
//	git      4811 peter ...
//
// Returns ("", "") when output has fewer than two lines or the
// second line lacks a PID-shaped column.
func parseLsof(out string) (pid, name string) {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		return "", ""
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 2 {
		return "", ""
	}
	return fields[1], fields[0]
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
