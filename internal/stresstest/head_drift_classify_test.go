package stresstest

import (
	"strings"
	"testing"
)

// head_drift_classify_test.go pins classifyHeadDrift — the pure
// decision logic behind HeadDriftScenario (M-0243/AC-5) — against
// fabricated outcomes, so every branch is exercised deterministically.
//
// A refused promote (G-0269's branch guard blocked it outright) or a
// commit landing on the preflight-observed branch both report no
// violation — the guard is doing its job. A commit landing on the
// interloper branch instead is a regression, reported as a violation.

func TestClassifyHeadDrift(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                     string
		promStatus               string
		landedOnPreflightBranch  bool
		landedOnInterloperBranch bool
		wantSubstring            string // "" means no violation expected
	}{
		{
			name:                     "regression: the promote committed onto the interloper branch, not the preflight-observed one",
			promStatus:               "ok",
			landedOnPreflightBranch:  false,
			landedOnInterloperBranch: true,
			wantSubstring:            "G-0269",
		},
		{
			name:                     "the promote landed on the preflight-observed branch",
			promStatus:               "ok",
			landedOnPreflightBranch:  true,
			landedOnInterloperBranch: false,
			wantSubstring:            "",
		},
		{
			name:                     "unexpected: the commit landed on neither branch",
			promStatus:               "ok",
			landedOnPreflightBranch:  false,
			landedOnInterloperBranch: false,
			wantSubstring:            "landed on neither",
		},
		{
			name:                     "the promote was refused outright (a guard blocked it) — not the silent-landing failure mode",
			promStatus:               "error",
			landedOnPreflightBranch:  false,
			landedOnInterloperBranch: true,
			wantSubstring:            "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyHeadDrift(tc.promStatus, tc.landedOnPreflightBranch, tc.landedOnInterloperBranch)
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
