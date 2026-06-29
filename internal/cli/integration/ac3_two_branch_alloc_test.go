package integration

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestAdd_TwoBranchesNoCollision pins M-0212/AC-3: two local branches
// sharing one object store (the sibling-worktree scenario) do not
// collide. Branch A allocates a gap and commits it; an allocation
// driven from branch B — which forked before A's commit and so does
// not carry it in its working tree — observes A's ref via the
// broadened local-refs scan and allocates the NEXT id, not a duplicate.
//
// Driven through the real `aiwf add` dispatcher so the full seam is
// exercised: treeload → trunk.LocalRefIDs → Tree.AllocationIDs →
// entity.AllocateID. Without the M-0212 scan branch B would re-allocate
// A's id (the collision this milestone prevents).
func TestAdd_TwoBranchesNoCollision(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	// aiwf init writes files but does not git-commit; stage a base
	// commit so branchA's add has a parent to fork branchB from.
	mustGit(t, root, "add", "-A")
	mustGit(t, root, "commit", "-q", "-m", "base")

	// Branch A forks from the base commit and allocates the first gap.
	mustGit(t, root, "checkout", "-b", "branchA")
	mustRun(t, "add", "gap", "--title", "Alpha gap", "--actor", "human/test", "--root", root)
	gotA := soleGapID(t, root)

	// Branch B forks from BEFORE A's add commit (branchA~1 = the init
	// commit), so its working tree does not contain A's gap. A naive
	// {working-tree + trunk} allocator would hand back the same id.
	mustGit(t, root, "checkout", "-b", "branchB", "branchA~1")
	mustRun(t, "add", "gap", "--title", "Bravo gap", "--actor", "human/test", "--root", root)
	gotB := soleGapID(t, root)

	if gotB == gotA {
		t.Fatalf("collision: both branches allocated %s; the local-refs scan should have skipped branch A's id", gotA)
	}
	if gotA != "G-0001" || gotB != "G-0002" {
		t.Errorf("ids = (A=%s, B=%s), want (G-0001, G-0002)", gotA, gotB)
	}
}

// mustGit runs a git command in root, failing the test on error.
func mustGit(t *testing.T, root string, args ...string) {
	t.Helper()
	if err := osExec(t, root, "git", args...); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}

// soleGapID globs the single gap file in root's working tree and
// returns its id (e.g. "G-0002"), derived from the filename.
func soleGapID(t *testing.T, root string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, "work", "gaps", "G-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob work/gaps/G-*.md: matches=%v err=%v", matches, err)
	}
	parts := strings.SplitN(filepath.Base(matches[0]), "-", 3)
	if len(parts) < 2 {
		t.Fatalf("unexpected gap filename %q", matches[0])
	}
	return parts[0] + "-" + parts[1]
}
