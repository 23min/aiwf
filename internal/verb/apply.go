package verb

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/23min/aiwf/internal/entity"
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
	// The sovereign-force rules are checked here, not inside each verb,
	// because a verb's trailer set is incomplete when the verb returns:
	// the CLI layer appends aiwf-principal, aiwf-on-behalf-of,
	// aiwf-authorized-by and aiwf-scope-ends to the plan afterwards.
	// Apply is the one seam downstream both of that decoration and of the
	// verbs that assemble a complete set themselves, so it is the only
	// point where the whole set is visible while nothing has been written
	// yet.
	//
	// The subset is deliberate and its reasoning lives on
	// CheckForceTrailerCoherence. In short: a verb whose trailer set is
	// incomplete for a reason unrelated to force is the push's business,
	// not this seam's.
	//
	// It runs first so a refusal costs no filesystem work and leaves HEAD
	// where it was.
	if cohErr := CheckForceTrailerCoherence(p.Trailers); cohErr != nil {
		return "", cohErr
	}
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
	// An unborn HEAD has nothing to diverge from, and a verb's own
	// commit is routinely a repo's first, so the guard is skipped rather
	// than erroring on `git diff HEAD`.
	hasHEAD, headErr := gitops.HasHEAD(ctx, root)
	if headErr != nil { //coverage:ignore defensive: HasHEAD errors only when the directory is no git repo, which StagedPaths above already refused
		return "", fmt.Errorf("checking for uncommitted changes: %w", headErr)
	}
	// The carried set is enumerated whether or not HEAD resolves, because
	// the two checks below need different halves of it. Only the record's
	// side needs HEAD; what is a symbolic link on disk is answerable
	// without one.
	carried, carriedErr := planCarriedPaths(ctx, root, p.Ops, hasHEAD)
	if carriedErr != nil {
		return "", fmt.Errorf("checking for uncommitted changes: %w", carriedErr)
	}
	// Unconditional: a link the commit path would dereference corrupts the
	// record just as thoroughly in a repo whose first commit this is.
	if linkErr := checkCarriedSymlinks(root, carried, p.Ops); linkErr != nil {
		return "", linkErr
	}
	if hasHEAD {
		diverged, divErr := gitops.DivergentPaths(ctx, root, carried)
		if divErr != nil {
			return "", fmt.Errorf("checking for uncommitted changes: %w", divErr)
		}
		if conflictErr := checkUncommittedConflict(ctx, root, diverged, p.Ops); conflictErr != nil {
			return "", conflictErr
		}
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
		if _, covered := planOpForPath(s, ops); covered {
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

// planCarriedPaths returns every repo-relative path this plan's commit
// would carry, gathered from the two sides the guard compares.
//
// A move's carried set is wider than the paths the verb named: os.Rename
// takes a directory's whole contents, and gatherCommitOps rebuilds the
// commit by walking what it finds at the destination. The disk under a
// move's source is therefore one side, and it reaches paths no git query
// reports — an ignored file, one carrying `assume-unchanged`, one a
// sparse checkout omits.
//
// HEAD's tree under that same prefix is the other side, and it is not
// redundant. A path the record carries and the working tree lacks is
// never re-written at the destination and never removed from the source,
// so the commit strands it at the old location while its siblings move —
// a split directory `aiwf check` reports no error on.
//
// An OpWrite contributes its own destination. Paths are deduped and
// sorted, so a refusal names them the same way twice.
func planCarriedPaths(ctx context.Context, root string, ops []FileOp, consultHEAD bool) ([]string, error) {
	seen := make(map[string]bool, len(ops))
	var out []string
	add := func(p string) {
		if seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}

	for _, op := range ops {
		switch op.Type {
		case OpWrite:
			add(op.Path)
		case OpMove:
			// Both ends: the source is what the move carries, and the
			// destination is what it would land on — an untracked file
			// already sitting there is content the commit would replace
			// without anyone naming it.
			for _, end := range []string{op.Path, op.NewPath} {
				if err := addCarriedUnder(ctx, root, end, add, consultHEAD); err != nil {
					return nil, err
				}
			}
		default:
			// A new op type contributes no paths here, so the guard would
			// not see whatever it carries. Refusing is the conservative
			// reading: an op this function cannot enumerate is one whose
			// commit contents it cannot vouch for.
			//coverage:ignore unreachable: OpType is a closed set of two, both handled above; a third is a source change that lands here first
			return nil, fmt.Errorf("cannot determine what op type %d carries", op.Type)
		}
	}
	sort.Strings(out)
	return out, nil
}

// addCarriedUnder reports every path a move of src would carry, from
// both sides: the working tree beneath it, and the record beneath it.
// A directory contributes its contents; anything else is the single path
// it is — including a source missing from disk, which is named here
// rather than left to surface as an os.Rename failure in Phase 1.
//
// Shared by the commit-side guard and the archive sweep's per-candidate
// decline, so the two cannot drift on what a move is considered to
// carry.
func addCarriedUnder(ctx context.Context, root, src string, add func(string), consultHEAD bool) error {
	info, statErr := os.Lstat(filepath.Join(root, src))
	if statErr == nil && info.IsDir() {
		if walkErr := addFilesUnder(root, src, add); walkErr != nil {
			return fmt.Errorf("walking %s to determine what the commit carries: %w", src, walkErr)
		}
	} else {
		add(src)
	}
	if !consultHEAD {
		// An unborn HEAD records nothing, so the working tree is the whole
		// carried set.
		return nil
	}
	headPaths, lsErr := gitops.LsTreePaths(ctx, root, "HEAD", src+"/")
	if lsErr != nil { //coverage:ignore defensive: HEAD resolves — both callers consult HasHEAD first, and the same repo has already answered a git query
		return fmt.Errorf("listing %s at HEAD: %w", src, lsErr)
	}
	for _, p := range headPaths {
		add(p)
	}
	return nil
}

// addFilesUnder reports every file beneath prefix on disk, as a
// repo-relative slash path, mirroring the walk gatherCommitOps performs
// at the destination once the move has happened.
func addFilesUnder(root, prefix string, add func(string)) error {
	base := filepath.Join(root, prefix)
	return filepath.WalkDir(base, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil { //coverage:ignore defensive: the root was just stat'd as a directory, and WalkDir surfaces per-entry errors only for unreadable subdirectories
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(base, path)
		if relErr != nil { //coverage:ignore WalkDir always yields paths rooted at base; Rel can only fail for a path outside base's tree
			return relErr
		}
		add(prefix + "/" + filepath.ToSlash(rel))
		return nil
	})
}

// planOpForPath returns the op whose write set covers path, and whether
// any does. An OpWrite covers only its exact Path; an OpMove also covers
// anything nested under its source or destination directory, matching
// the nested writes gatherCommitOps discovers by walking a moved
// directory. Prefix-matching the op's path strings is equivalent to that
// walk and needs no filesystem access, so the answer is available before
// Phase 1 has moved anything.
//
// Both working-tree guards resolve a path through this one rule, so the
// staged and uncommitted checks cannot drift apart on which paths a plan
// is considered to touch.
func planOpForPath(path string, ops []FileOp) (FileOp, bool) {
	for _, op := range ops {
		switch op.Type {
		case OpWrite:
			if path == op.Path {
				return op, true
			}
		case OpMove:
			if path == op.Path || path == op.NewPath ||
				strings.HasPrefix(path, op.Path+"/") ||
				strings.HasPrefix(path, op.NewPath+"/") {
				return op, true
			}
		}
	}
	return FileOp{}, false
}

// CarriedSymlinkError reports that a verb was refused because a path it
// would carry is a symbolic link, which the commit path cannot record as
// one. Callers map it to a usage-level exit: the operator can resolve it.
type CarriedSymlinkError struct {
	// Carried names links the plan would sweep up under a move. The
	// commit stores a copy of each link's target at that path.
	Carried []string
	// Named names links a verb writes to directly. The verb's own content
	// lands there, so nothing unowned is recorded — but the link is still
	// replaced by a regular file.
	Named []string
}

// Paths returns every blocking link, carried first.
func (e *CarriedSymlinkError) Paths() []string {
	return append(append([]string{}, e.Carried...), e.Named...)
}

func (e *CarriedSymlinkError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "this verb would replace a symbolic link it cannot record as one: %s\n",
		strings.Join(e.Paths(), ", "))
	b.WriteString("  the commit stores every path it touches as a regular file, so the link\n")
	b.WriteString("  itself is not preserved\n")
	if len(e.Carried) > 0 {
		fmt.Fprintf(&b, "  swept up under a move, so the commit would record a copy of whatever the\n"+
			"  link points at — content no verb computed, under this verb's own trailer, and\n"+
			"  for a link pointing outside the repo, content from outside it: %s\n"+
			"  move it out of the way, or replace it with a real file\n",
			strings.Join(e.Carried, " "))
	}
	if len(e.Named) > 0 {
		fmt.Fprintf(&b, "  written by this verb, so its own content would land there — but at a path\n"+
			"  the operator set up as a link, silently converted: %s\n"+
			"  replace the link with a real file if that is what it should be\n",
			strings.Join(e.Named, " "))
	}
	b.WriteString("  then re-run the verb")
	return b.String()
}

// checkCarriedSymlinks refuses when any path the plan would carry is a
// symbolic link.
//
// The refusal is unconditional rather than keyed on divergence, because
// divergence is the wrong question here. A link whose target string still
// equals the record is unchanged by every measure git offers — and the
// commit path would still dereference it (gatherCommitOps reads content
// with os.ReadFile) and store the result at mode 100644 (CommitTree's
// cacheInfo), replacing the link with a copy of its target and leaving
// the working tree reporting a type change nothing can clear.
//
// Recording links faithfully is the fix this defers to; until then a
// refusal is the honest answer, since the alternative silently rewrites
// the record.
func checkCarriedSymlinks(root string, carried []string, ops []FileOp) error {
	err := &CarriedSymlinkError{}
	for _, p := range carried {
		info, statErr := os.Lstat(filepath.Join(root, filepath.FromSlash(p)))
		if statErr != nil {
			// Absent or uninspectable paths are the divergence
			// comparison's to report, with remedies of their own.
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		if op, ok := planOpForPath(p, ops); ok && op.Type == OpWrite && p == op.Path {
			err.Named = append(err.Named, p)
			continue
		}
		err.Carried = append(err.Carried, p)
	}
	if len(err.Paths()) == 0 {
		return nil
	}
	sort.Strings(err.Carried)
	sort.Strings(err.Named)
	return err
}

// UncommittedConflictError reports that a verb was refused because a
// path it was about to commit carries changes the verb did not compute.
// Callers map it to a usage-level exit: the operator can resolve it, and
// nothing in aiwf's own machinery is broken.
//
// The three path roles are kept apart because they have different
// remedies, and offering the wrong one is worse than offering none:
// `git restore` errors on a path git has never recorded, discards work
// irrecoverably on one it has, and is the whole fix for one missing from
// the working tree.
type UncommittedConflictError struct {
	// Tracked names blocking paths that have a committed version whose
	// bytes the working copy no longer matches.
	Tracked []string
	// Untracked names blocking paths HEAD has no version of, including
	// ignored files a move would carry.
	Untracked []string
	// Missing names blocking paths HEAD records and the working tree
	// lacks. A move would strand each at its old location rather than
	// carrying it, since the commit's writes come from what is on disk.
	Missing []string
}

// Paths returns every blocking path: tracked, then untracked, then
// missing.
func (e *UncommittedConflictError) Paths() []string {
	out := append([]string{}, e.Tracked...)
	out = append(out, e.Untracked...)
	return append(out, e.Missing...)
}

func (e *UncommittedConflictError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "uncommitted changes overlap with this verb's writes: %s\n",
		strings.Join(e.Paths(), ", "))
	b.WriteString("  the verb commits whatever is on disk at the paths it touches, so it would\n")
	b.WriteString("  record your changes as its own work, under its own trailer\n")
	if len(e.Tracked) > 0 {
		fmt.Fprintf(&b, "  commit a body edit on its own with `aiwf edit-body <id>`, or set it aside with\n"+
			"  `git stash -u` (`git restore %s` discards it outright)\n",
			strings.Join(e.Tracked, " "))
	}
	if len(e.Untracked) > 0 {
		// `-u` covers untracked paths but not ignored ones, and this
		// bucket holds both: the comparison finds a path regardless of
		// `.gitignore`, so the remedy has to reach as far as the
		// detection does. `-a` covers each, which is why it is named
		// rather than left to whichever case the operator happens to hit.
		fmt.Fprintf(&b, "  untracked here, so there is nothing to restore: commit it, move it out of the\n"+
			"  way, or set it aside with `git stash -a` (`-u` alone skips a path `.gitignore`\n"+
			"  matches, and this refusal reaches those too) — %s\n",
			strings.Join(e.Untracked, " "))
	}
	if len(e.Missing) > 0 {
		// `git restore` alone refuses a path carrying skip-worktree —
		// which is how a sparse checkout omits one, and so is the likeliest
		// way to arrive here. Clearing the bit first is a no-op on a path
		// that does not carry it, so one form serves both.
		fmt.Fprintf(&b, "  recorded but absent from your working tree, so a move would strand it where it is\n"+
			"  instead of carrying it: bring it back with\n"+
			"  `git update-index --no-skip-worktree -- %s && git restore %s`\n"+
			"  (a sparse checkout or `skip-worktree` hides a path from every other check)\n",
			strings.Join(e.Missing, " "), strings.Join(e.Missing, " "))
	}
	b.WriteString("  then re-run the verb; unrelated uncommitted paths survive the verb's commit untouched")
	return b.String()
}

// checkUncommittedConflict refuses Apply when a path the plan would
// commit holds content the verb did not compute — an unblessed body
// edit, a hand-edited field, an untracked file that happens to sit
// inside a directory being moved. Without it a verb commits those bytes
// verbatim under its own trailer, so `aiwf history` attributes a change
// to an act that did not make it (ADR-0038).
//
// The check runs before Phase 1, which is the only point where the
// operator's working copy is still readable: by the time commit
// construction reads the tree back, the verb's own moves and writes have
// replaced it.
//
// Two path roles are treated differently, and both readings are
// load-bearing:
//
//   - A path the plan names as an OpWrite destination and git has never
//     tracked is left alone. There is no committed version for the write
//     to contradict, so it creates the record rather than laundering one.
//     `aiwf.yaml` in a freshly-initialised repo is the case that matters:
//     `aiwf init` leaves it uncommitted by design, and the verbs that
//     rewrite it would otherwise be unreachable until it was committed.
//   - A path that is merely nested under a move is refused whether it
//     has a committed version or not, because no verb named it and its
//     bytes are carried into the commit sight-unseen.
//
// The divergence set is computed by comparing HEAD's blobs against disk
// for the paths the plan would carry (planCarriedPaths, then
// gitops.DivergentPaths), not by intersecting those paths with git's
// report of what the operator changed. The two answer different
// questions, and only the first is about what the commit records: an
// ignored file, one carrying `assume-unchanged` or `skip-worktree`, and
// one a sparse checkout omits are all paths git declines to report while
// the commit carries them regardless (G-0492, G-0487).
//
// A write may declare it adopts the working copy (AdoptsWorkingCopy).
// The claim is verified rather than trusted (adoptionPreservesFrontmatter):
// the working copy's own frontmatter must still match HEAD's, so the
// exemption carries a changed body — which is the point — and can never
// carry a field the operator edited by hand; and the write's own content
// must carry nothing beyond a legitimate re-serialization of that working
// copy, so the exemption cannot be claimed for content the plan computed
// on its own.
func checkUncommittedConflict(ctx context.Context, root string, diverged []gitops.Divergence, ops []FileOp) error {
	if len(ops) == 0 || len(diverged) == 0 {
		return nil
	}
	conflict := &UncommittedConflictError{}
	for _, d := range diverged {
		// Every path here came from ops, so the lookup resolves; what it
		// is consulted for is which op covers the path, and whether the
		// path is that op's own named destination.
		op, _ := planOpForPath(d.Path, ops)
		named := op.Type == OpWrite && d.Path == op.Path
		if named && d.Kind == gitops.DivergenceAbsentFromHEAD {
			continue
		}
		if named && d.Kind == gitops.DivergenceModified && op.AdoptsWorkingCopy {
			adopted, err := adoptionPreservesFrontmatter(ctx, root, d.Path, op.Content)
			if err != nil { //coverage:ignore defensive: propagates only the IO/parse faults adoptionPreservesFrontmatter and reconstructedFrontmatterMatches annotate as unreachable
				return err
			}
			if adopted {
				continue
			}
		}
		switch d.Kind {
		case gitops.DivergenceModified:
			conflict.Tracked = append(conflict.Tracked, d.Path)
		case gitops.DivergenceAbsentFromHEAD:
			conflict.Untracked = append(conflict.Untracked, d.Path)
		case gitops.DivergenceAbsentFromDisk:
			conflict.Missing = append(conflict.Missing, d.Path)
		default: //coverage:ignore unreachable: DivergenceKind is a closed set of three, each handled above; a fourth is a source change that would land here first
			// A fourth way for a working copy to disagree with the record
			// would need a remedy of its own. Refusing without one at
			// least keeps the commit from carrying it unexamined.
			conflict.Tracked = append(conflict.Tracked, d.Path)
		}
	}
	if len(conflict.Paths()) == 0 {
		return nil
	}
	return conflict
}

// adoptionPreservesFrontmatter reports whether an adopting write at path
// may proceed. Two conditions both have to hold, and each closes a
// different gap:
//
//   - The working copy's frontmatter still equals HEAD's, so the body is
//     the only thing the operator changed. This is what makes
//     `aiwf edit-body` workable without opening a hole: that verb exists
//     to commit a divergent body, so refusing on divergence would block
//     the one route out of every other refusal this guard raises.
//   - The write's own content carries nothing beyond the working copy
//     itself. `edit-body` has two ways of building an adopting write, and
//     both must be recognized: bless mode commits the working copy's
//     bytes verbatim, so content matches the working copy's frontmatter
//     exactly; explicit mode re-serializes it through the loaded entity
//     model, which normalizes fields the loader always normalizes (a
//     milestone's stray `area`, e.g.) — so content may instead match a
//     reconstruction of the working copy through that same normalization
//     (reconstructedFrontmatterMatches). Either is a legitimate way to
//     reflect the working copy; a write matching neither is content the
//     plan computed on its own, which the first condition alone cannot
//     see, since it never looks at the write's own bytes at all.
//
// Comparisons are field-based rather than byte-based throughout, because
// a verb legitimately re-canonicalizes frontmatter it did not change: a
// committed non-canonical field order would otherwise read as divergence.
// What the exemption guarantees is therefore specific: no field the
// operator hand-edited rides in, and no field the write's content adds
// beyond what the working copy — read one of the two ways a verb
// legitimately reads it — already declares. It is not a claim that the
// verb changed no field, which is the verb's own body-only contract to
// keep.
//
// A path with no committed version is not adopted. It cannot be reached
// in that state anyway — an untracked named write returns earlier — and
// treating a missing HEAD version as a match would grant the exemption
// on the strength of a comparison that never happened.
func adoptionPreservesFrontmatter(ctx context.Context, root, path string, content []byte) (bool, error) {
	headBytes, err := gitops.ReadFromHEAD(ctx, root, filepath.ToSlash(path))
	if err != nil { //coverage:ignore defensive: ReadFromHEAD maps a missing path to (nil, nil); a non-nil error needs git absent or a broken workdir
		return false, fmt.Errorf("reading HEAD version of %s: %w", path, err)
	}
	if headBytes == nil { //coverage:ignore unreachable: a path with no HEAD version is untracked, and an untracked named write returns before this call
		return false, nil
	}
	diskBytes, err := os.ReadFile(filepath.Join(root, path))
	if err != nil { //coverage:ignore defensive: only edit-body sets the adoption flag, and it resolved this entity from this path through tree.Load moments earlier; a dirty report alone would not imply the file exists, since git reports a deletion as dirty too
		return false, fmt.Errorf("reading working copy of %s: %w", path, err)
	}
	if !entity.SameFrontmatterFields(headBytes, diskBytes) {
		return false, nil
	}
	if entity.SameFrontmatterFields(diskBytes, content) {
		return true, nil
	}
	return reconstructedFrontmatterMatches(path, diskBytes, content)
}

// reconstructedFrontmatterMatches reports whether content's frontmatter
// equals a fresh parse-normalize-reserialize pass over diskBytes — the
// same pipeline tree.Load and every serializing verb run to build the
// entity a write's content is derived from. A write reflecting nothing
// but that pipeline's own output compares equal; a write carrying a
// field the pipeline would not have produced from diskBytes — fabricated,
// or copied from somewhere other than this path's own working copy —
// does not.
//
// Called only once adoptionPreservesFrontmatter's direct comparison
// against diskBytes has already failed, so diskBytes is trusted to parse:
// it already round-tripped through entity.SameFrontmatterFields's own
// YAML decode twice over.
func reconstructedFrontmatterMatches(path string, diskBytes, content []byte) (bool, error) {
	// A path the loader would not recognize as any entity kind has no
	// tree-load pipeline to reconstruct through — refused rather than
	// waved through for want of a comparison.
	kind, ok := entity.PathKind(path)
	if !ok {
		return false, nil
	}
	// diskBytes already parsed as a YAML mapping for the frontmatter-
	// fields comparison in adoptionPreservesFrontmatter, which tolerates
	// any key; the typed decode here does not. A field neither Entity
	// nor any verb ever declared makes no legitimate re-serialization
	// producible, so refuse rather than compare against nothing.
	parsed, err := entity.Parse(path, diskBytes)
	if err != nil {
		return false, nil
	}
	parsed.Kind = kind
	entity.NormalizeForKind(parsed, kind)
	reconstructed, err := entity.Serialize(parsed, nil)
	if err != nil { //coverage:ignore defensive: marshaling a freshly-parsed Entity back to YAML fails only for a shape Serialize itself cannot represent, which the successful entity.Parse above already ruled out
		return false, fmt.Errorf("reconstructing %s for comparison: %w", path, err)
	}
	return entity.SameFrontmatterFields(reconstructed, content), nil
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
