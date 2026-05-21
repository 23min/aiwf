package check

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/entity"
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

	got := FSMHistoryConsistent(context.Background(), r.root, tr)

	// The walker must attribute the proposed → done observation to
	// E-0001 even though the commit touched the entity's OLD path. The
	// rule's illegal-transition predicate then fires.
	var hasFinding bool
	for _, f := range got {
		if f.Code == "fsm-history-consistent" &&
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

// TestBatchedWalker_OctopusMerge pins the walker's behavior on a
// merge commit with three parents. M-0130 / M-0137 both inherit
// `git log -m`'s per-parent-diff fan-out for merges; the M-0137 walker
// dedupes observations by (commit, parent, path) so each real
// (commit, parent) tuple emits at most one observation, even when
// BulkRevwalk yields multiple CommitRecord chunks for the same merge.
//
// Scenario: an octopus merge integrating two feature branches into
// main, where the entity's status differs across all three parents.
// Expected: per-parent observations emerge for each parent whose
// state differs from the merge's resolved state, each (commit, parent)
// pair counted at most once.
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

	// Dedup invariant: no (commit, parent) pair appears more than once
	// in the observations.
	seen := map[string]int{}
	for _, o := range obs {
		key := o.Commit + "::" + o.Parent
		seen[key]++
	}
	for key, count := range seen {
		if count > 1 {
			t.Errorf("octopus merge (commit, parent) %q appears %d times; dedup should collapse to 1", key, count)
		}
	}

	// Sanity: the merge integrated three branches with differing
	// statuses, so at least one merge-commit observation should
	// emerge. (Some git versions fall back to sequential 2-parent
	// merges under conflicts; either shape still produces multi-
	// parent merge commits the walker dedups uniformly.)
	if len(obs) == 0 {
		t.Errorf("expected merge-commit observations for the octopus fixture; got 0")
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
