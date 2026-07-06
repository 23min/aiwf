package gitops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// seedRepo inits a git repo at t.TempDir() with one commit on the
// default branch, returning the root. Shared setup for the
// WorktreeAdd family of tests, which all need a base commit to
// branch from.
func seedRepo(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "seed.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(ctx, root, "seed.md"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := Commit(ctx, root, "seed commit", "", nil); err != nil {
		t.Fatalf("commit: %v", err)
	}
	return root
}

func TestBranchExists(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t)

	if exists, err := BranchExists(ctx, root, "main"); err != nil || !exists {
		t.Errorf("BranchExists(main) = %v, %v; want true, nil", exists, err)
	}
	if exists, err := BranchExists(ctx, root, "does-not-exist"); err != nil || exists {
		t.Errorf("BranchExists(does-not-exist) = %v, %v; want false, nil", exists, err)
	}
}

func TestWorktreeAddNewBranch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t)
	wtPath := filepath.Join(t.TempDir(), "wt")

	if err := WorktreeAddNewBranch(ctx, root, wtPath, "feature/x", "main"); err != nil {
		t.Fatalf("WorktreeAddNewBranch: %v", err)
	}
	if _, err := os.Stat(filepath.Join(wtPath, "seed.md")); err != nil {
		t.Errorf("seed.md missing in new worktree: %v", err)
	}
	if exists, err := BranchExists(ctx, root, "feature/x"); err != nil || !exists {
		t.Errorf("branch feature/x should exist after WorktreeAddNewBranch; exists=%v err=%v", exists, err)
	}
}

func TestWorktreeAddNewBranch_DefaultsToHEAD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t)
	wtPath := filepath.Join(t.TempDir(), "wt")

	// Omitting base defers to git's own default (HEAD) rather than
	// aiwf inventing a different fallback.
	if err := WorktreeAddNewBranch(ctx, root, wtPath, "feature/y", ""); err != nil {
		t.Fatalf("WorktreeAddNewBranch with empty base: %v", err)
	}
	if _, err := os.Stat(filepath.Join(wtPath, "seed.md")); err != nil {
		t.Errorf("seed.md missing in new worktree: %v", err)
	}
}

func TestWorktreeAdd_ExistingBranch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t)

	if err := run(ctx, root, "branch", "existing-branch"); err != nil {
		t.Fatalf("git branch: %v", err)
	}
	wtPath := filepath.Join(t.TempDir(), "wt")
	if err := WorktreeAdd(ctx, root, wtPath, "existing-branch"); err != nil {
		t.Fatalf("WorktreeAdd: %v", err)
	}
	if _, err := os.Stat(filepath.Join(wtPath, "seed.md")); err != nil {
		t.Errorf("seed.md missing in new worktree: %v", err)
	}
}

func TestWorktreeAdd_SurfacesGitFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t)

	// main is already checked out in root itself; checking it out
	// again into a second worktree is a real git refusal the wrapper
	// must surface, not swallow.
	wtPath := filepath.Join(t.TempDir(), "wt")
	err := WorktreeAdd(ctx, root, wtPath, "main")
	if err == nil {
		t.Fatal("WorktreeAdd should fail when branch is already checked out elsewhere")
	}
	if !strings.Contains(err.Error(), "already") {
		t.Errorf("error should surface git's own explanation; got: %v", err)
	}
}

// TestParseWorktreeList covers every documented shape of
// `git worktree list --porcelain` output: a single main checkout, a
// main + linked worktrees, a detached-HEAD worktree, a bare repo
// entry (skipped), and the trailing-newline-vs-not edge case.
func TestParseWorktreeList(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want []Worktree
	}{
		{
			name: "main checkout only",
			in: `worktree /repo
HEAD abc123
branch refs/heads/main
`,
			want: []Worktree{
				{Path: "/repo", Branch: "main", HeadSHA: "abc123"},
			},
		},
		{
			name: "main + two linked worktrees",
			in: `worktree /repo
HEAD abc123
branch refs/heads/main

worktree /repo-feature
HEAD def456
branch refs/heads/feature/x

worktree /repo-patch
HEAD 789012
branch refs/heads/patch/g-0122-worktree-view
`,
			want: []Worktree{
				{Path: "/repo", Branch: "main", HeadSHA: "abc123"},
				{Path: "/repo-feature", Branch: "feature/x", HeadSHA: "def456"},
				{Path: "/repo-patch", Branch: "patch/g-0122-worktree-view", HeadSHA: "789012"},
			},
		},
		{
			name: "detached HEAD worktree (no branch line)",
			in: `worktree /repo
HEAD abc123
branch refs/heads/main

worktree /repo-detached
HEAD def456
detached
`,
			want: []Worktree{
				{Path: "/repo", Branch: "main", HeadSHA: "abc123"},
				{Path: "/repo-detached", Branch: "", HeadSHA: "def456"},
			},
		},
		{
			name: "bare repo entry skipped",
			in: `worktree /bare
bare

worktree /repo
HEAD abc123
branch refs/heads/main
`,
			want: []Worktree{
				{Path: "/repo", Branch: "main", HeadSHA: "abc123"},
			},
		},
		{
			name: "no trailing blank line",
			in: `worktree /repo
HEAD abc123
branch refs/heads/main`,
			want: []Worktree{
				{Path: "/repo", Branch: "main", HeadSHA: "abc123"},
			},
		},
		{
			name: "empty input",
			in:   ``,
			want: nil,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseWorktreeList(tc.in)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("parseWorktreeList mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
