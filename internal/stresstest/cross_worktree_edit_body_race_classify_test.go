package stresstest

import (
	"strings"
	"testing"
)

// cross_worktree_edit_body_race_classify_test.go pins
// classifyCrossWorktreeEditBodyRace — the pure decision logic behind
// CrossWorktreeEditBodyRaceScenario (M-0243/AC-2) — against fabricated
// merge outcomes, so every branch is exercised deterministically
// rather than depending on a real merge's exact text.

func TestClassifyCrossWorktreeEditBodyRace(t *testing.T) {
	t.Parallel()
	const draftA = "operator A's independent edit to the shared entity"
	const draftB = "operator B's independent edit to the shared entity"

	tests := []struct {
		name           string
		conflicted     bool
		mergedContent  string
		wantSubstrings []string // nil means no violations expected
	}{
		{
			name:           "conflicted merge preserves both operators' content in the conflict markers",
			conflicted:     true,
			mergedContent:  "<<<<<<< HEAD\n" + draftA + "\n=======\n" + draftB + "\n>>>>>>> actor-b\n",
			wantSubstrings: nil,
		},
		{
			name:           "conflicted merge but operator A's content is missing from the result",
			conflicted:     true,
			mergedContent:  "<<<<<<< HEAD\nsome other text\n=======\n" + draftB + "\n>>>>>>> actor-b\n",
			wantSubstrings: []string{"lost operator A's content"},
		},
		{
			name:           "conflicted merge but operator B's content is missing from the result",
			conflicted:     true,
			mergedContent:  "<<<<<<< HEAD\n" + draftA + "\n=======\nsome other text\n>>>>>>> actor-b\n",
			wantSubstrings: []string{"lost operator B's content"},
		},
		{
			name:           "conflicted merge but neither operator's content survived",
			conflicted:     true,
			mergedContent:  "<<<<<<< HEAD\nsome other text\n=======\nsome other other text\n>>>>>>> actor-b\n",
			wantSubstrings: []string{"lost operator A's content", "lost operator B's content"},
		},
		{
			name:           "clean (non-conflicting) merge landed on operator A's content",
			conflicted:     false,
			mergedContent:  "---\nid: G-0001\n---\n" + draftA + "\n",
			wantSubstrings: nil,
		},
		{
			name:           "clean (non-conflicting) merge landed on operator B's content",
			conflicted:     false,
			mergedContent:  "---\nid: G-0001\n---\n" + draftB + "\n",
			wantSubstrings: nil,
		},
		{
			name:           "clean (non-conflicting) merge landed on neither operator's content — silent data loss",
			conflicted:     false,
			mergedContent:  "neither operator wrote this",
			wantSubstrings: []string{"silent, untraceable data loss"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyCrossWorktreeEditBodyRace(tc.conflicted, tc.mergedContent, draftA, draftB)
			if len(got) != len(tc.wantSubstrings) {
				t.Fatalf("violations = %+v, want %d matching %v", got, len(tc.wantSubstrings), tc.wantSubstrings)
			}
			for i, want := range tc.wantSubstrings {
				if !strings.Contains(got[i].Message, want) {
					t.Errorf("violation[%d] = %q, want it to contain %q", i, got[i].Message, want)
				}
			}
		})
	}
}
