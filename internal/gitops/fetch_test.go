package gitops

import (
	"context"
	"os/exec"
	"slices"
	"testing"
)

func TestFetchBranch_RefreshesRemoteTrackingRef(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// Upstream repo with an initial commit on main.
	up := initTestRepo(t)
	commitFile(t, ctx, up, "work/gaps/G-0005-a.md", "a\n")
	// Clone it; the clone's refs/remotes/origin/main pins this state.
	clone := cloneRepo(t, up)
	// Advance upstream out-of-band — the clone's tracking ref is now stale.
	commitFile(t, ctx, up, "work/gaps/G-0009-b.md", "b\n")

	before, err := LsTreePaths(ctx, clone, "refs/remotes/origin/main", "work/")
	if err != nil {
		t.Fatalf("ls-tree before: %v", err)
	}
	if slices.Contains(before, "work/gaps/G-0009-b.md") {
		t.Fatal("precondition: clone already carries G-0009 before fetch")
	}

	if ferr := FetchBranch(ctx, clone, "origin", "main"); ferr != nil {
		t.Fatalf("FetchBranch: %v", ferr)
	}

	after, err := LsTreePaths(ctx, clone, "refs/remotes/origin/main", "work/")
	if err != nil {
		t.Fatalf("ls-tree after: %v", err)
	}
	if !slices.Contains(after, "work/gaps/G-0009-b.md") {
		t.Errorf("after FetchBranch, refs/remotes/origin/main is missing G-0009: %v", after)
	}
}

func TestFetchBranch_NoRemote_Errors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	commitFile(t, ctx, dir, "x.txt", "x\n")
	// No 'origin' remote configured → git fetch exits non-zero.
	if err := FetchBranch(ctx, dir, "origin", "main"); err == nil {
		t.Error("FetchBranch with no 'origin' remote = nil error, want error")
	}
}

// cloneRepo clones src into a fresh temp dir and returns the clone path.
func cloneRepo(t *testing.T, src string) string {
	t.Helper()
	dst := t.TempDir()
	cmd := exec.Command("git", "clone", "-q", src, dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone: %v\n%s", err, out)
	}
	return dst
}
