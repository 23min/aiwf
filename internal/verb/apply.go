package verb

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/pathutil"
)

// Apply executes a verb's Plan against the consumer repo at root: it
// runs every OpMove via a pure filesystem rename, every OpWrite
// atomically to disk via pathutil.AtomicWriteFile (creating parent
// directories as needed), then builds the single commit and reconciles
// exactly the touched paths into the live index via
// gitops.CommitVerbChange — the one exported commit-construction seam
// (M-0186/AC-5).
//
// Moves run before writes so that when a verb (notably reallocate)
// renames a file/dir and also rewrites files inside that dir, the
// writes land at the new locations.
//
// Isolation (M-0186): CommitTree builds the commit from HEAD's tree
// plus the verb's own removes/writes, entirely against a throwaway
// index — it never reads or writes the live index or worktree. Phase
// 1/2 are pure filesystem operations too (os.Rename,
// pathutil.AtomicWriteFile), so nothing is ever staged into the live
// index before a successful commit. This replaces the earlier
// git-stash isolation dance (G-0275/G-0276): there is nothing left to
// stash, because the live index is never touched until the one,
// narrowly-scoped ReconcilePaths call after the commit lands.
//
// Conflict guard: if the user has already staged a path the verb is
// about to write, Apply refuses before any disk mutation. The two
// intents — the user's staged content, the verb's computed content —
// disagree on what that path should hold; letting the verb proceed
// would have ReconcilePaths silently overwrite the user's staged
// version with the verb's once the commit lands.
//
// Atomicity: Apply is all-or-nothing up to the commit. If any step
// before a successful commit fails (write error, commit failure,
// panic), the worktree is restored to its pre-Apply state via a
// deferred rollback — a pure filesystem operation with no git call, so
// it cannot itself be blocked by lock contention or any other git
// failure. Once the commit lands, it is never rolled back (it's git
// history); a subsequent reconciliation failure is reported but does
// not undo the commit — see reconcileFailureError.
//
// sha is non-empty if and only if err is nil: even in the reconcile-
// failure case (the commit itself landed but syncing the live index
// afterward failed), Apply reports "", err rather than surfacing a
// sha alongside a non-nil error — the sha is not lost, it is already
// embedded in that error's own text (reconcileFailureError), so a
// caller gets a simple "sha present means clean success" contract
// instead of having to special-case a partial-success sha.
func Apply(ctx context.Context, root string, p *Plan) (sha string, err error) {
	staged, stagedErr := gitops.StagedPaths(ctx, root)
	if stagedErr != nil {
		return "", fmt.Errorf("checking pre-staged changes: %w", stagedErr)
	}
	if opErr := checkNoGitOperationInProgress(ctx, root); opErr != nil {
		return "", opErr
	}
	if conflictErr := checkStagedConflict(staged, p.Ops); conflictErr != nil {
		return "", conflictErr
	}

	tx := &applyTx{root: root, ctx: ctx}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.rollback()
			panic(r)
		}
		if err != nil {
			if rbErr := tx.rollback(); rbErr != nil { //coverage:ignore defensive: requires both primary error and rollback failure simultaneously
				err = fmt.Errorf("%w (rollback also failed: %w — manual cleanup may be needed)", err, rbErr)
			}
		}
	}()

	// Phase 1: moves. A pure filesystem rename — CommitTree builds the
	// commit from HEAD's tree plus explicit removes/writes below, not
	// from the live index, so there is no reason to stage the rename
	// there at all. os.Rename does not auto-create parent directories
	// for the destination, so we MkdirAll the target's parent first.
	//
	// One rename undoes the move regardless of file vs. directory (see
	// moveUndo, D-0029): it doesn't need to read what's inside a moved
	// directory, so reversal stays correct even with a permission-denied
	// entry nested inside, and — recorded in the same chronological
	// journal as Phase 2's writes, replayed LIFO on rollback — composes
	// correctly with a later OpWrite that rewrites a file inside the
	// moved directory (undo the rewrite before reversing the move).
	for _, op := range p.Ops {
		if op.Type != OpMove {
			continue
		}
		srcFull := filepath.Join(root, op.Path)
		destFull := filepath.Join(root, op.NewPath)
		if mkdirErr := os.MkdirAll(filepath.Dir(destFull), 0o755); mkdirErr != nil {
			return "", fmt.Errorf("creating parent of %s: %w", op.NewPath, mkdirErr)
		}
		if mvErr := os.Rename(srcFull, destFull); mvErr != nil {
			return "", fmt.Errorf("moving %s -> %s: %w", op.Path, op.NewPath, mvErr)
		}
		tx.journal = append(tx.journal, moveUndo{from: op.Path, to: op.NewPath})
	}

	// Phase 2: writes.
	for _, op := range p.Ops {
		if op.Type != OpWrite {
			continue
		}
		// Capture whatever is on disk at op.Path RIGHT BEFORE this write
		// — not once per path, but once per write. A path written twice
		// (or moved into, then rewritten) gets an undo step per write;
		// LIFO replay on rollback naturally lands a repeatedly-written
		// path on its true pre-Apply state, since each step restores
		// what was there immediately before it ran. G-0170.
		undo, capErr := captureWrite(root, op.Path)
		if capErr != nil {
			return "", capErr
		}
		full := filepath.Join(root, op.Path)
		if mkdirErr := os.MkdirAll(filepath.Dir(full), 0o755); mkdirErr != nil {
			return "", fmt.Errorf("creating %s: %w", filepath.Dir(op.Path), mkdirErr)
		}
		if writeErr := pathutil.AtomicWriteFile(full, op.Content, 0o644); writeErr != nil {
			return "", fmt.Errorf("writing %s: %w", op.Path, writeErr)
		}
		tx.journal = append(tx.journal, undo)
	}

	removes, writes, gatherErr := gatherCommitOps(root, p)
	if gatherErr != nil {
		return "", gatherErr
	}

	// git commit-tree (unlike git commit) has no built-in refusal for a
	// same-tree commit — without this guard, a plan that computes zero
	// Ops without setting AllowEmpty (a verb bug) would silently create
	// an empty commit instead of failing loudly.
	if !p.AllowEmpty && len(removes) == 0 && len(writes) == 0 {
		return "", errors.New("nothing to commit: plan has no file operations")
	}

	var commitErr error
	sha, commitErr = gitops.CommitVerbChange(ctx, root, removes, writes, p.Subject, p.Body, p.Trailers)
	if sha != "" {
		tx.committed = true
	}
	if commitErr != nil {
		var reconcileErr *gitops.ReconcileError
		if errors.As(commitErr, &reconcileErr) {
			return "", reconcileFailureError(ctx, root, reconcileErr.SHA, reconcileErr.Err)
		}
		return "", fmt.Errorf("commit-tree: %w", commitErr)
	}
	return sha, nil
}

// gatherCommitOps determines the full removes/writes sets CommitTree
// needs, reading back the worktree's current state after both phases
// have fully run — rather than trusting op.Content — so a plan that
// both moves and rewrites the same destination (reallocate, move)
// lands the FINAL bytes regardless of Ops order.
//
// An OpMove's destination may be a single file OR a directory (an
// epic/contract dir move, potentially containing a nested milestone):
// os.Rename moves a directory atomically without altering its internal
// relative structure, so a directory destination is walked recursively,
// producing one old-path/new-path pair per file inside by substituting
// the op's Path/NewPath prefixes. An OpWrite contributes its own path
// directly (including one that rewrites a file inside a just-moved
// directory). Paths are deduped by final path so a move-then-rewrite
// pair produces exactly one write.
func gatherCommitOps(root string, p *Plan) (removes []string, writes []gitops.PathWrite, err error) {
	seen := make(map[string]bool, len(p.Ops))
	addFile := func(oldPath, newPath string) error {
		if seen[newPath] {
			return nil
		}
		seen[newPath] = true
		content, readErr := os.ReadFile(filepath.Join(root, newPath))
		if readErr != nil {
			return fmt.Errorf("reading %s for commit: %w", newPath, readErr)
		}
		if oldPath != "" {
			removes = append(removes, oldPath)
		}
		writes = append(writes, gitops.PathWrite{Path: newPath, Content: content})
		return nil
	}

	for _, op := range p.Ops {
		if op.Type != OpMove {
			continue
		}
		destFull := filepath.Join(root, op.NewPath)
		info, statErr := os.Lstat(destFull)
		if statErr != nil {
			return nil, nil, fmt.Errorf("stat %s for commit: %w", op.NewPath, statErr)
		}
		if !info.IsDir() {
			if addErr := addFile(op.Path, op.NewPath); addErr != nil {
				return nil, nil, addErr
			}
			continue
		}
		walkErr := filepath.WalkDir(destFull, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			rel, relErr := filepath.Rel(destFull, path)
			if relErr != nil { //coverage:ignore WalkDir always yields paths rooted at destFull; Rel can only fail for a path outside destFull's tree
				return relErr
			}
			rel = filepath.ToSlash(rel)
			return addFile(op.Path+"/"+rel, op.NewPath+"/"+rel)
		})
		if walkErr != nil {
			return nil, nil, fmt.Errorf("walking %s for commit: %w", op.NewPath, walkErr)
		}
	}

	for _, op := range p.Ops {
		if op.Type != OpWrite {
			continue
		}
		if addErr := addFile("", op.Path); addErr != nil {
			return nil, nil, addErr
		}
	}

	return removes, writes, nil
}

// checkStagedConflict refuses Apply when the user has already staged
// content for a path the verb is about to write, rename, or — for a
// directory OpMove — for a path nested inside the moved directory.
// gatherCommitOps walks a moved directory's destination recursively
// and captures whatever is on disk for every nested file, so a staged
// edit nested under op.Path/op.NewPath is part of the verb's real
// write set even though it is not one of the two paths named on the
// op itself; checking prefixes here (rather than walking the
// filesystem before Phase 1 has even run) keeps the guard in sync with
// that write set using only the staged-path strings already in hand.
//
// The two intents (the user's staged content for a path, the verb's
// computed content for that same path) cannot both land in the verb's
// commit, and letting the verb proceed would have the post-commit
// ReconcilePaths step silently overwrite the user's staged version
// with the verb's once the commit lands. The error message names every
// conflicting path and points the user at `git restore --staged` /
// `git stash` so recovery is mechanical.
//
// Pre-staged paths *outside* the verb's path set are simply left
// alone — Apply never touches the live index for any path it did not
// itself write, so they survive the verb's commit untouched.
func checkStagedConflict(staged []string, ops []FileOp) error {
	if len(staged) == 0 || len(ops) == 0 {
		return nil
	}
	var conflicts []string
	for _, s := range staged {
		if stagedPathConflicts(s, ops) {
			conflicts = append(conflicts, s)
		}
	}
	if len(conflicts) == 0 {
		return nil
	}
	return fmt.Errorf(
		"pre-staged changes overlap with this verb's writes: %s\n"+
			"  the verb cannot decide between your staged content and the content it computed\n"+
			"  run `git restore --staged %s` to unstage your changes, or `git stash` to set them aside,\n"+
			"  then re-run the verb — unrelated staged paths survive the verb's commit",
		strings.Join(conflicts, ", "),
		strings.Join(conflicts, " "),
	)
}

// stagedPathConflicts reports whether a staged path overlaps with one
// of the plan's ops: an OpWrite conflicts only on an exact match; an
// OpMove also conflicts on any path nested under its source or
// destination directory, matching the nested writes gatherCommitOps
// discovers by walking a moved directory.
func stagedPathConflicts(staged string, ops []FileOp) bool {
	for _, op := range ops {
		switch op.Type {
		case OpWrite:
			if staged == op.Path {
				return true
			}
		case OpMove:
			if staged == op.Path || staged == op.NewPath ||
				strings.HasPrefix(staged, op.Path+"/") ||
				strings.HasPrefix(staged, op.NewPath+"/") {
				return true
			}
		}
	}
	return false
}

// checkNoGitOperationInProgress refuses Apply when a merge,
// cherry-pick, revert, or rebase is already under way in root's repo.
// Apply's commit machinery (gitops.CommitVerbChange) moves HEAD via
// commit-tree + update-ref independently of any pending operation's
// state; running it mid-operation leaves that operation's on-disk
// markers (MERGE_HEAD, etc.) pointing at a HEAD that has since moved,
// corrupting whatever the operator does next to finish it (G-0329).
// Resolved via gitops.GitDir, not root/".git", so a linked worktree
// checks its own per-worktree gitdir — these markers live there, not
// in the shared common dir.
func checkNoGitOperationInProgress(ctx context.Context, root string) error {
	gitDir, err := gitops.GitDir(ctx, root)
	if err != nil {
		return fmt.Errorf("checking for an in-progress git operation: %w", err)
	}
	markers := []struct{ path, label string }{
		{filepath.Join(gitDir, "MERGE_HEAD"), "a merge"},
		{filepath.Join(gitDir, "CHERRY_PICK_HEAD"), "a cherry-pick"},
		{filepath.Join(gitDir, "REVERT_HEAD"), "a revert"},
	}
	for _, m := range markers {
		if _, statErr := os.Stat(m.path); statErr == nil {
			return fmt.Errorf("%s is in progress in this repo; complete or abort it before running this command", m.label)
		}
	}
	for _, dir := range []string{"rebase-merge", "rebase-apply"} {
		if info, statErr := os.Stat(filepath.Join(gitDir, dir)); statErr == nil && info.IsDir() {
			return errors.New("a rebase is in progress in this repo; complete or abort it before running this command")
		}
	}
	return nil
}

// reconcileFailureError composes the error Apply returns when the
// commit landed but ReconcilePaths (syncing the verb's paths into the
// live index) failed. `--audit-only` recovery does not apply here —
// the commit already exists in git history, complete with trailers.
// The fix is re-running `git add` for the affected paths once the
// underlying issue (commonly `.git/index.lock` contention from an
// unrelated process — the one lock ReconcilePaths can still hit, since
// unlike CommitTree it does touch the live index) clears.
func reconcileFailureError(ctx context.Context, root, sha string, reconcileErr error) error {
	var hint string
	if isIndexLockError(reconcileErr.Error()) {
		hint = lockContentionHint(ctx, root)
	}
	if hint == "" {
		return fmt.Errorf(
			"verb commit %s landed but syncing your index failed: %w\n"+
				"  your commit is safe; run `git add` for the affected paths once the issue clears\n"+
				"  (`git status` shows what's affected)",
			sha, reconcileErr,
		)
	}
	return fmt.Errorf(
		"verb commit %s landed but syncing your index failed: %w\n"+
			"  %s\n"+
			"  your commit is safe; run `git add` for the affected paths once the issue clears",
		sha, reconcileErr, hint,
	)
}

// isIndexLockError reports whether the error string from a failed git
// operation indicates `.git/index.lock` contention. Git's exact
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
		// The lock cleared between the failure and our diagnostic —
		// race, but harmless; nothing to report.
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
	if name == "" { //coverage:ignore parseLsof only pairs a non-empty pid with an empty name if fields[0] were empty, which strings.Fields never produces — structurally unreachable given parseLsof's own contract, not a race
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

// undoStep reverses one completed Phase 1/2 mutation. applyTx.journal
// records these in execution order; rollback replays them in reverse
// (LIFO) — see D-0029. LIFO is what makes a directory move composed
// with a later rewrite of a file inside it reversible: the rewrite's
// undo (restore pre-rewrite bytes) runs before the move's undo
// (rename the directory back), so the directory carries the
// correctly-restored file back with it in one rename.
type undoStep interface {
	undo(root string) error
}

// moveUndo reverses a completed OpMove (file or directory alike) via a
// direct rename back. A rename doesn't need to read what's inside a
// directory, so this stays correct even with a permission-denied entry
// nested inside — the property the pre-unification `dirMoves` design
// also relied on (see D-0029).
type moveUndo struct {
	from, to string
}

func (u moveUndo) undo(root string) error {
	toFull := filepath.Join(root, u.to)
	if _, statErr := os.Lstat(toFull); statErr != nil {
		// Already gone (removed by something else before rollback ran,
		// or never really landed) — nothing to reverse.
		return nil
	}
	fromFull := filepath.Join(root, u.from)
	if mvErr := os.Rename(toFull, fromFull); mvErr != nil {
		return fmt.Errorf("reversing move %s -> %s on rollback: %w", u.to, u.from, mvErr)
	}
	return nil
}

// writeUndo reverses a completed OpWrite by restoring the bytes
// captured immediately before that write ran (captureWrite), or
// removing the path if it didn't exist before that write.
type writeUndo struct {
	path    string
	existed bool
	content []byte
}

func (u writeUndo) undo(root string) error {
	full := filepath.Join(root, u.path)
	if !u.existed {
		if rmErr := os.Remove(full); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			return fmt.Errorf("removing %s on rollback: %w", u.path, rmErr)
		}
		return nil
	}
	if mkErr := os.MkdirAll(filepath.Dir(full), 0o755); mkErr != nil { //coverage:ignore requires concurrent FS mutation: the parent was readable moments earlier when this write's capture ran
		return fmt.Errorf("creating parent of %s on rollback: %w", u.path, mkErr)
	}
	if wErr := pathutil.AtomicWriteFile(full, u.content, 0o644); wErr != nil {
		return fmt.Errorf("restoring %s to pre-apply state: %w", u.path, wErr)
	}
	return nil
}

// captureWrite returns the undoStep that reverses a write about to
// happen at rel, snapshotting whatever is currently on disk there (or
// recording its absence). Must be called immediately before the write
// it protects — capturing per-write, not once per path, is what lets
// LIFO replay land a repeatedly-written path on its true pre-Apply
// state. G-0170.
func captureWrite(root, rel string) (writeUndo, error) {
	full := filepath.Join(root, rel)
	data, err := os.ReadFile(full)
	if errors.Is(err, os.ErrNotExist) {
		return writeUndo{path: rel, existed: false}, nil
	}
	if err != nil {
		return writeUndo{}, fmt.Errorf("capturing pre-write state of %s: %w", rel, err)
	}
	return writeUndo{path: rel, existed: true, content: data}, nil
}

// applyTx tracks a partial Apply's completed mutations so the deferred
// rollback can restore the repo to its pre-call state.
//
// journal is the chronological undo log Phase 1/2 append to as each
// mutation succeeds; rollback replays it LIFO. This makes a failed
// commit leave the worktree exactly as the operator left it — including
// uncommitted edits at touched paths, and any directory moves composed
// with rewrites of files inside them — rather than reverting to HEAD
// or mishandling the composition (G-0170, D-0029).
type applyTx struct {
	root      string
	ctx       context.Context
	journal   []undoStep
	committed bool // when true, rollback is a no-op — the mutation succeeded
}

// rollback reverses every recorded mutation in strict LIFO order — the
// most recent action undone first. Pure filesystem: no git call runs
// here, so rollback cannot itself be blocked by lock contention or any
// other git failure. Safe to call multiple times. A no-op once the
// verb's commit has landed (nothing to undo — the mutation succeeded).
func (t *applyTx) rollback() error {
	if t.committed {
		return nil
	}
	var firstErr error
	for i := len(t.journal) - 1; i >= 0; i-- {
		if err := t.journal[i].undo(t.root); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
