package gitops

import (
	"context"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestFetchAll_RefreshesRemoteTrackingRefs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// Upstream on main; clone pins its state; upstream then advances
	// out-of-band, so the clone's refs/remotes/origin/main goes stale.
	up := initTestRepo(t)
	commitFile(t, ctx, up, "work/gaps/G-0005-a.md", "a\n")
	clone := cloneRepo(t, up)
	commitFile(t, ctx, up, "work/gaps/G-0009-b.md", "b\n")

	before, err := LsTreePaths(ctx, clone, "refs/remotes/origin/main", "work/")
	if err != nil {
		t.Fatalf("ls-tree before: %v", err)
	}
	if slices.Contains(before, "work/gaps/G-0009-b.md") {
		t.Fatal("precondition: clone already carries G-0009 before fetch")
	}

	if ferr := FetchAll(ctx, clone); ferr != nil {
		t.Fatalf("FetchAll: %v", ferr)
	}

	after, err := LsTreePaths(ctx, clone, "refs/remotes/origin/main", "work/")
	if err != nil {
		t.Fatalf("ls-tree after: %v", err)
	}
	if !slices.Contains(after, "work/gaps/G-0009-b.md") {
		t.Errorf("after FetchAll, refs/remotes/origin/main is missing G-0009: %v", after)
	}
}

func TestFetchAll_NoRemote_NoError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	commitFile(t, ctx, dir, "x.txt", "x\n")
	// `git fetch --all` with no remotes is a clean no-op (exit 0) — there
	// is nothing to fetch, so it must not error.
	if err := FetchAll(ctx, dir); err != nil {
		t.Errorf("FetchAll with no remotes = %v, want nil (no-op)", err)
	}
}

func TestFetchAll_BadRemote_Errors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	commitFile(t, ctx, dir, "x.txt", "x\n")
	// A remote pointing at a nonexistent local path fails fast, offline —
	// the error the caller's best-effort policy degrades on.
	mustRun(t, ctx, dir, "remote", "add", "origin", filepath.Join(t.TempDir(), "nope.git"))
	if err := FetchAll(ctx, dir); err == nil {
		t.Error("FetchAll with an unreachable remote = nil error, want error")
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
