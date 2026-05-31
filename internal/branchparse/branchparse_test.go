package branchparse

import "testing"

// TestParseEntityFromBranch covers the ritual-shape branch grammar
// defined by ADR-0010: `epic/E-NNNN-<slug>`, `milestone/M-NNNN-<slug>`,
// `patch/g-NNNN-<slug>` (case-insensitive id segment). Other shapes
// yield "". This is the source of truth M-0102 lifts out of
// internal/cli/status/worktrees.go so M-0103's preflight and the
// existing aiwf status --worktrees correlation share one regex set.
func TestParseEntityFromBranch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, branch, want string
	}{
		{"epic branch with slug", "epic/E-0010-cobra-and-completion", "E-0010"},
		{"epic branch id-only", "epic/E-0010", "E-0010"},
		{"milestone branch with slug", "milestone/M-0007-cache", "M-0007"},
		{"milestone branch id-only", "milestone/M-0007", "M-0007"},
		{"patch branch lowercase id", "patch/g-0099-isolation", "G-0099"},
		{"patch branch uppercase id", "patch/G-0099-isolation", "G-0099"},
		{"narrow-legacy id width preserved on output", "epic/E-01-old", "E-01"},
		{"main branch returns empty", "main", ""},
		{"empty branch returns empty", "", ""},
		{"fix prefix returns empty", "fix/something", ""},
		{"chore prefix returns empty", "chore/something", ""},
		{"patch without id segment returns empty", "patch/some-topic", ""},
		{"epic without id segment returns empty", "epic/no-id-here", ""},
		{"wrong kind id (E- under milestone/) accepted by id-shape", "milestone/E-0010-mismatch", "E-0010"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ParseEntityFromBranch(tc.branch); got != tc.want {
				t.Errorf("ParseEntityFromBranch(%q) = %q, want %q", tc.branch, got, tc.want)
			}
		})
	}
}
