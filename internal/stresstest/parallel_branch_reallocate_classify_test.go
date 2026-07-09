package stresstest

import (
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
		wantViolations    int
	}{
		{
			name:              "clean resolution: check surfaces it, reallocate ok, post-check clear, push succeeds",
			checkFindings:     idsUnique,
			reallocateStatus:  "ok",
			postCheckFindings: otherFinding,
			pushedClean:       true,
			wantViolations:    0,
		},
		{
			name:              "aiwf check never surfaced the collision",
			checkFindings:     otherFinding,
			reallocateStatus:  "ok",
			postCheckFindings: otherFinding,
			pushedClean:       true,
			wantViolations:    1,
		},
		{
			name:              "reallocate did not report ok",
			checkFindings:     idsUnique,
			reallocateStatus:  "error",
			postCheckFindings: otherFinding,
			pushedClean:       true,
			wantViolations:    1,
		},
		{
			name:              "ids-unique finding still present after reallocate",
			checkFindings:     idsUnique,
			reallocateStatus:  "ok",
			postCheckFindings: idsUnique,
			pushedClean:       true,
			wantViolations:    1,
		},
		{
			name:              "final push after reallocate did not succeed cleanly",
			checkFindings:     idsUnique,
			reallocateStatus:  "ok",
			postCheckFindings: otherFinding,
			pushedClean:       false,
			wantViolations:    1,
		},
		{
			name:              "every check fails at once",
			checkFindings:     otherFinding,
			reallocateStatus:  "error",
			postCheckFindings: idsUnique,
			pushedClean:       false,
			wantViolations:    4,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyParallelBranchReallocate(tc.checkFindings, tc.reallocateStatus, tc.postCheckFindings, tc.pushedClean)
			if len(got) != tc.wantViolations {
				t.Errorf("violations = %d (%+v), want %d", len(got), got, tc.wantViolations)
			}
		})
	}
}
