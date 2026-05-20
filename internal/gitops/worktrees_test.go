package gitops

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

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
