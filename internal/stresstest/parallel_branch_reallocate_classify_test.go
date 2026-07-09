package stresstest

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// parallel_branch_reallocate_classify_test.go pins
// classifyParallelBranchReallocate — the pure decision logic behind
// ParallelBranchReallocateScenario (M-0243/AC-1) — against fabricated
// envelope data, so every branch is exercised deterministically rather
// than depending on a real merge/push sequence's exact outcome.

func TestClassifyParallelBranchReallocate(t *testing.T) {
	t.Parallel()
	idsUnique := []verbEnvelopeFinding{{Code: check.CodeIDsUnique, Severity: "error"}}
	otherFinding := []verbEnvelopeFinding{{Code: "some-other-code", Severity: "warning"}}

	tests := []struct {
		name              string
		checkFindings     []verbEnvelopeFinding
		reallocateStatus  string
		postCheckFindings []verbEnvelopeFinding
		pushedClean       bool
		wantSubstrings    []string // nil means no violations expected
	}{
		{
			name:              "clean resolution: check surfaces it, reallocate ok, post-check clear, push succeeds",
			checkFindings:     idsUnique,
			reallocateStatus:  "ok",
			postCheckFindings: otherFinding,
			pushedClean:       true,
			wantSubstrings:    nil,
		},
		{
			name:              "aiwf check never surfaced the collision",
			checkFindings:     otherFinding,
			reallocateStatus:  "ok",
			postCheckFindings: otherFinding,
			pushedClean:       true,
			wantSubstrings:    []string{"did not surface it as"},
		},
		{
			name:              "reallocate did not report ok",
			checkFindings:     idsUnique,
			reallocateStatus:  "error",
			postCheckFindings: otherFinding,
			pushedClean:       true,
			wantSubstrings:    []string{"did not cleanly resolve the collision"},
		},
		{
			name:              "ids-unique finding still present after reallocate",
			checkFindings:     idsUnique,
			reallocateStatus:  "ok",
			postCheckFindings: idsUnique,
			pushedClean:       true,
			wantSubstrings:    []string{"finding still present after"},
		},
		{
			name:              "final push after reallocate did not succeed cleanly",
			checkFindings:     idsUnique,
			reallocateStatus:  "ok",
			postCheckFindings: otherFinding,
			pushedClean:       false,
			wantSubstrings:    []string{"did not succeed cleanly"},
		},
		{
			name:              "every check fails at once",
			checkFindings:     otherFinding,
			reallocateStatus:  "error",
			postCheckFindings: idsUnique,
			pushedClean:       false,
			wantSubstrings: []string{
				"did not surface it as",
				"did not cleanly resolve the collision",
				"finding still present after",
				"did not succeed cleanly",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyParallelBranchReallocate(tc.checkFindings, tc.reallocateStatus, tc.postCheckFindings, tc.pushedClean)
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
