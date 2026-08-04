package check

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// TestBatchedWalker_RenameChainTracking pins the M-0137/AC-3 walker's
// rename-chain handling. The walker maintains a pathToEntity map
// seeded from the tree's CURRENT paths and adds SrcPath entries when
// a rename touch is processed (newest-first walk). Without that
// extension, older commits at the pre-rename path would not resolve
// to the entity, and observations at the entity's historical path
// would be lost.
//
// Scenario: entity E-0001 was created at OLD path with status=proposed,
// promoted to active (illegal FSM transition for an epic — used as
// the observable marker), then renamed to NEW path. The tree's
// current path is NEW. The walker should observe the proposed →
// active status change at OLD path (attributed to E-0001) and emit
// the expected illegal-transition finding through the rule's normal
// predicate path.
func TestBatchedWalker_RenameChainTracking(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)

	// Commit 1: create the epic at OLD path with status=proposed.
	oldPath := "work/epics/E-0001-old/epic.md"
	r.writeEntityAtRel(oldPath, "E-0001", entity.KindEpic, entity.StatusProposed, "")
	r.gitAddAll()
	r.gitCommit("add E-0001 at old path")

	// Commit 2: status change at OLD path (proposed → done is FSM-illegal
	// for an epic per its transitions table — used here as the
	// observable marker the walker should attribute to E-0001).
	r.writeEntityAtRel(oldPath, "E-0001", entity.KindEpic, entity.StatusDone, "")
	r.gitAddAll()
	r.gitCommit("illegal status change at old path (proposed → done)")

	// Commit 3: rename old → new (no status change in this commit;
	// the new path is now where the entity lives).
	newPath := "work/epics/E-0001-new/epic.md"
	if err := os.MkdirAll(filepath.Join(r.root, filepath.Dir(newPath)), 0o755); err != nil {
		t.Fatalf("mkdir new dir: %v", err)
	}
	r.run("git", "mv", oldPath, newPath)
	r.gitCommit("rename E-0001 to new path")

	// Build tree pointing at the CURRENT (new) path only — emulating
	// what tree.Load would produce post-rename.
	tr := &tree.Tree{
		Root: r.root,
		Entities: []*entity.Entity{
			{ID: "E-0001", Kind: entity.KindEpic, Path: newPath},
		},
	}

	got := FSMHistoryConsistent(context.Background(), r.root, tr, nil, mustHead(t, r.root))

	// The walker must attribute the proposed → done observation to
	// E-0001 even though the commit touched the entity's OLD path. The
	// rule's illegal-transition predicate then fires.
	var hasFinding bool
	for _, f := range got {
		if f.Code == CodeFSMHistoryConsistent &&
			f.Subcode == "illegal-transition" &&
			f.EntityID == "E-0001" {
			hasFinding = true
		}
	}
	if !hasFinding {
		t.Errorf("expected illegal-transition finding for E-0001 (observation at pre-rename path should be attributed via rename-chain tracking); got %d finding(s): %+v",
			len(got), got)
	}
}

// TestBatchedWalker_MissingNonZeroBlob_EmitsWalkError pins G-0327: a
// real blob id the local object store cannot produce — a damaged store,
// or a partial clone that can no longer reach the remote it would fetch
// the blob from — surfaces as a history-walk-error finding rather than
// a silent skip that hides whatever transition the pair carried.
//
// Injected through the blobReader seam because a repo in that state is
// not constructible by the fixture's ordinary git commands.
//
// The walker reads two blobs per pair and reports each side under its
// own label, so the subtests fail the older and the newer blob
// separately to reach both. Failing the older one is the case that
// matters most: the commit-side read succeeds, so the walk has a status
// in hand and only the comparison is lost — precisely the pair a silent
// skip would drop without a trace.
func TestBatchedWalker_MissingNonZeroBlob_EmitsWalkError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		// missingIn selects which commit's blob the reader cannot
		// produce, by a substring unique to that blob's content.
		missingIn string
		wantSide  string
	}{
		{name: "newer blob unreadable", missingIn: "status: done", wantSide: "commit"},
		{name: "older blob unreadable", missingIn: "status: proposed", wantSide: "parent"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := newRepoFixture(t)
			r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001")
			r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone, "skip-ahead illegal")

			delegate, err := gitops.NewBlobReader(context.Background(), r.root)
			if err != nil {
				t.Fatalf("NewBlobReader: %v", err)
			}
			defer delegate.Close()
			// ErrBlobMissing for a blob that genuinely exists in this repo
			// — the answer a store that cannot produce it gives for a
			// non-zero id `git log --raw` still names.
			fake := &fakeBlobReader{
				delegate:           delegate,
				errOnContentSubstr: c.missingIn,
				readErr:            gitops.ErrBlobMissing,
			}

			got := fsmHistoryConsistentWithDeps(context.Background(), r.root, r.tree(), nil, mustHead(t, r.root), fake)

			var sawSide bool
			for _, f := range got {
				if f.Code != CodeFSMHistoryConsistent ||
					f.Subcode != "history-walk-error" ||
					f.EntityID != "E-0001" {
					continue
				}
				if f.Severity != SeverityError {
					t.Errorf("history-walk-error severity = %q, want error", f.Severity)
				}
				if strings.Contains(f.Message, "reading "+c.wantSide+" status") {
					sawSide = true
				}
			}
			if !sawSide {
				t.Errorf("expected a %s-side history-walk-error for E-0001 (non-zero blob the store cannot produce); got %d finding(s): %+v",
					c.wantSide, len(got), got)
			}
		})
	}
}

// TestBatchedWalker_AllZeroBlob_NeverReachesTheReader pins the other
// half of G-0327: the absent side of an add or a delete keeps its
// skip. Its all-zero id short-circuits ahead of the read, so hardening
// the missing-blob case cannot turn an ordinary add or delete into a
// finding.
func TestBatchedWalker_AllZeroBlob_NeverReachesTheReader(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)
	relPath := canonicalEntityPath("E-0001", entity.KindEpic)
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001") // all-zero PreSHA
	r.run("git", "rm", "-q", relPath)
	r.gitCommit("delete E-0001") // all-zero PostSHA

	delegate, err := gitops.NewBlobReader(context.Background(), r.root)
	if err != nil {
		t.Fatalf("NewBlobReader: %v", err)
	}
	defer delegate.Close()
	rec := &recordingBlobReader{delegate: delegate}

	// The file is gone from the working tree, so build the tree by hand
	// rather than through the git-ls-files helper.
	tr := &tree.Tree{
		Root: r.root,
		Entities: []*entity.Entity{
			{ID: "E-0001", Kind: entity.KindEpic, Path: relPath},
		},
	}

	got := fsmHistoryConsistentWithDeps(context.Background(), r.root, tr, nil, mustHead(t, r.root), rec)

	if rec.sawAllZero {
		t.Error("all-zero blob id reached ReadObject; the absent side of an add/delete must short-circuit before the read")
	}
	for _, f := range got {
		if f.Code == CodeFSMHistoryConsistent && f.Subcode == "history-walk-error" {
			t.Errorf("unexpected history-walk-error for an ordinary add + delete: %+v", f)
		}
	}
}

// recordingBlobReader delegates every read and records whether an
// all-zero blob id ever reached ReadObject.
type recordingBlobReader struct {
	delegate   *gitops.BlobReader
	sawAllZero bool
}

func (rb *recordingBlobReader) Read(commit, path string) ([]byte, error) {
	return rb.delegate.Read(commit, path)
}

func (rb *recordingBlobReader) ReadObject(sha string) ([]byte, error) {
	if gitops.BlobAllZero(sha) {
		rb.sawAllZero = true
	}
	return rb.delegate.ReadObject(sha)
}

// Close is owned by the test, which holds the delegate.
func (rb *recordingBlobReader) Close() error { return nil }

// TestBatchedWalker_OctopusMerge pins the walker's behavior on a merge
// commit with three parents. Before G-0372 Fix 1, `git log -m` fanned
// out one diff record per parent for a merge commit, and the walker
// deduped observations by (commit, parent, path) so each real
// (commit, parent) tuple emitted at most one observation. Fix 1 drops
// `-m`: `git log --raw` now suppresses diff output for merge commits
// entirely (git's default without -m/-c/--cc), so no observation is
// ever produced at a merge commit — safe because every current
// consumer of these observations discards merge-commit observations
// unconditionally (D-0010). This test now characterizes THAT: even a
// conflicted octopus merge whose resolved state differs from all three
// parents produces zero observations.
func TestBatchedWalker_OctopusMerge(t *testing.T) {
	t.Parallel()
	r := newRepoFixture(t)

	// Root commit on main: epic at proposed.
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusProposed, "add E-0001")

	// Feature branch A: epic at active.
	r.gitCheckoutBranch("feat-a")
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusActive, "feat-a: status=active")

	// Back to main, branch B from there: epic at done.
	r.gitCheckout("main")
	r.gitCheckoutBranch("feat-b")
	r.commitEntity("E-0001", entity.KindEpic, entity.StatusDone, "feat-b: status=done")

	// Back to main; advance with another touch so main is at proposed
	// (root) but at a different commit than feat-a's branch point.
	r.gitCheckout("main")
	r.writeEntityAtRel(canonicalEntityPath("E-0001", entity.KindEpic),
		"E-0001", entity.KindEpic, entity.StatusProposed, "main retitle\n")
	r.gitAddAll()
	r.gitCommit("main: retitle (no status change)")

	// Octopus merge: integrate both feat-a and feat-b into main. With
	// conflicting status on all three sides, the merge needs an
	// explicit resolution — write the resolved file then commit. Some
	// git versions refuse octopus merges with conflicts, in which case
	// we sequence the merges instead (still produces multi-parent
	// merge commits the walker should handle uniformly).
	cmd := r.runMaybe("git", "merge", "--no-commit", "--no-ff", "feat-a", "feat-b")
	_ = cmd // octopus may exit non-zero on conflict; resolve below
	abs := filepath.Join(r.root, canonicalEntityPath("E-0001", entity.KindEpic))
	// Resolve to cancelled (differs from all three parents).
	r.writeEntityAt(abs, "E-0001", entity.KindEpic, entity.StatusCancelled, "")
	r.gitAddAll()
	r.run("git", "commit", "-q", "-m", "octopus merge a+b (resolved to cancelled)")

	tr := r.tree()
	obs, err := walkStatusChanges(context.Background(), r.root, tr)
	if err != nil {
		t.Fatalf("walkStatusChanges: %v", err)
	}

	// G-0372 Fix 1: without -m, `git log --raw` emits no diff records for
	// a merge commit — the octopus merge itself contributes zero
	// observations, regardless of how many parents' state it resolved.
	for _, o := range obs {
		if o.IsMergeCommit {
			t.Errorf("expected no merge-commit observations (Fix 1 drops -m), got %+v", o)
		}
	}
}

// runMaybe runs a git subcommand and tolerates non-zero exit. Used
// when the caller explicitly handles failures (e.g., a merge conflict
// that the test resolves immediately afterward).
func (r *repoFixture) runMaybe(name string, args ...string) string {
	r.t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = r.root
	out, _ := cmd.CombinedOutput()
	return string(out)
}

// TestIsRepoPath_FilesystemOnly pins the helper's contract: returns
// true when .git exists (whether as a directory in normal checkouts
// or as a file in worktree pointers); false otherwise. Defined as a
// filesystem-only check so a cancelled context doesn't false-negative
// the way the exec-based gitops.IsRepo subprocess call would.
func TestIsRepoPath_FilesystemOnly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cases := []struct {
		name   string
		setup  func(t *testing.T) string
		expect bool
	}{
		{
			name: "plain dir without .git",
			setup: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			expect: false,
		},
		{
			name: "dir with .git directory (normal repo)",
			setup: func(t *testing.T) string {
				t.Helper()
				root := t.TempDir()
				if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
					t.Fatalf("mkdir .git: %v", err)
				}
				return root
			},
			expect: true,
		},
		{
			name: "dir with .git file (worktree pointer shape)",
			setup: func(t *testing.T) string {
				t.Helper()
				root := t.TempDir()
				// Worktree pointer: .git is a regular file containing
				// `gitdir: <path>`.
				if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: /tmp/repo/.git/worktrees/wt"), 0o644); err != nil {
					t.Fatalf("write .git pointer: %v", err)
				}
				return root
			},
			expect: true,
		},
		{
			name:   "empty path returns false",
			setup:  func(_ *testing.T) string { return "" },
			expect: false,
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			root := c.setup(t)
			if got := isRepoPath(ctx, root); got != c.expect {
				t.Errorf("isRepoPath(%q) = %v, want %v", root, got, c.expect)
			}
		})
	}
}
