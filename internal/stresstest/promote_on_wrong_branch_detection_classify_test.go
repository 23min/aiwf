package stresstest

import (
	"strings"
	"testing"
)

// promote_on_wrong_branch_detection_classify_test.go pins
// classifyPromoteOnWrongBranchDetection — the pure decision logic
// behind PromoteOnWrongBranchDetectionScenario (G-0270) — against
// fabricated `aiwf check` outcomes, so every branch is exercised
// deterministically.

func TestClassifyPromoteOnWrongBranchDetection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		epicID        string
		findings      []verbEnvelopeFinding
		wantSubstring string // "" means no violation expected
	}{
		{
			name:   "detected: a promote-on-wrong-branch finding names this epic",
			epicID: "E-0001",
			findings: []verbEnvelopeFinding{
				{Code: "promote-on-wrong-branch", EntityID: "E-0001"},
			},
			wantSubstring: "",
		},
		{
			name:          "not detected: no findings at all",
			epicID:        "E-0001",
			findings:      nil,
			wantSubstring: "G-0270",
		},
		{
			name:   "not detected: a promote-on-wrong-branch finding exists but names a different entity",
			epicID: "E-0001",
			findings: []verbEnvelopeFinding{
				{Code: "promote-on-wrong-branch", EntityID: "E-0002"},
			},
			wantSubstring: "G-0270",
		},
		{
			name:   "not detected: findings exist but none is promote-on-wrong-branch",
			epicID: "E-0001",
			findings: []verbEnvelopeFinding{
				{Code: "isolation-escape", EntityID: "E-0001"},
			},
			wantSubstring: "G-0270",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyPromoteOnWrongBranchDetection(tc.epicID, verbEnvelope{Findings: tc.findings})
			if tc.wantSubstring == "" {
				if len(got) != 0 {
					t.Fatalf("violations = %+v, want none", got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("violations = %+v, want exactly 1", got)
			}
			if !strings.Contains(got[0].Message, tc.wantSubstring) {
				t.Fatalf("violation message = %q, want it to contain %q", got[0].Message, tc.wantSubstring)
			}
		})
	}
}
