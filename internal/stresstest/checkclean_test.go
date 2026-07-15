package stresstest

import (
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// checkclean_test.go pins classifyAgainstBaseline (M-0257/AC-2) — the
// shared check-clean-baseline oracle every scenario's own curated map
// parameterizes — against fabricated findings, with a baseline map
// deliberately DIFFERENT from verbSequenceExpectedWarnings so these
// tests prove the helper is genuinely generic, not hardcoded to
// verb-sequence's own baseline.

func TestClassifyAgainstBaseline(t *testing.T) {
	t.Parallel()
	baseline := map[string]bool{
		check.CodeProvenanceUntrailedScopeUndefined: true,
		check.CodeArchiveSweepPending:               true,
	}

	tests := []struct {
		name           string
		findings       []verbEnvelopeFinding
		wantViolations int
	}{
		{
			name:           "no findings",
			findings:       nil,
			wantViolations: 0,
		},
		{
			name: "every finding is a warning in the baseline",
			findings: []verbEnvelopeFinding{
				{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "warning"},
				{Code: check.CodeArchiveSweepPending, Severity: "warning"},
			},
			wantViolations: 0,
		},
		{
			name: "a warning with a code outside the baseline is a violation",
			findings: []verbEnvelopeFinding{
				{Code: "some-unbaselined-code", Severity: "warning"}, //enums:ignore deliberately fabricated non-code for the test, not a real finding
			},
			wantViolations: 1,
		},
		{
			name: "an error-severity finding is always a violation, even a code in the baseline",
			findings: []verbEnvelopeFinding{
				{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "error"},
			},
			wantViolations: 1,
		},
		{
			name: "mixed: one baselined warning, one unbaselined warning, one baselined-code error",
			findings: []verbEnvelopeFinding{
				{Code: check.CodeArchiveSweepPending, Severity: "warning"},
				{Code: "some-unbaselined-code", Severity: "warning"}, //enums:ignore deliberately fabricated non-code for the test, not a real finding
				{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "error"},
			},
			wantViolations: 2,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyAgainstBaseline(tc.findings, baseline)
			if len(got) != tc.wantViolations {
				t.Fatalf("violations = %+v, want %d", got, tc.wantViolations)
			}
		})
	}
}

// TestClassifyAgainstBaseline_EmptyBaselineFlagsEveryFinding pins the
// empty-baseline case ParallelBranchReallocateScenario's own map uses
// (M-0257/AC-1): with no curated entries at all, ANY finding —
// warning or error — is a violation.
func TestClassifyAgainstBaseline_EmptyBaselineFlagsEveryFinding(t *testing.T) {
	t.Parallel()
	got := classifyAgainstBaseline([]verbEnvelopeFinding{
		{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "warning"},
	}, map[string]bool{})
	if len(got) != 1 {
		t.Fatalf("violations = %+v, want 1", got)
	}
}
