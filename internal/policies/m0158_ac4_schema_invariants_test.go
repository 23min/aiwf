package policies

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec"
)

// TestM0158_AC4_SchemaInvariantsHoldOverBranchCells pins
// M-0158/AC-4: every cell in `branch.Rules()` satisfies the
// per-Rule schema invariants the M-0123 family already enforces
// over layers 1–3:
//
//   - Outcome ≠ OutcomeUnspecified (the zero-value sentinel — if
//     this fires it means a Rule literal forgot to set Outcome).
//   - Illegal ⇒ RejectionLayer ≠ RejectionLayerNone.
//   - VerbTime ⇒ BlockingStrict (verb-time rejections always
//     refuse strictly; warning severity is a check-time concept).
//   - Legal ⇒ ExpectedErrorCode == "" (legal cells don't reject;
//     a populated code on a Legal cell is a paste error).
//   - Sources.Decision (when non-empty) resolves — for layer-4
//     this is most commonly "ADR-0010" or "ADR-0003". The
//     resolver check is deliberately narrow: it asserts the
//     value has the ADR-NNNN or D-NNNN shape, not full
//     filesystem resolution (the M-0123/AC-6 test handles that
//     for layers 1–3; extending it to cover branch is
//     downstream of this milestone).
//
// The invariants are the same predicates the existing
// internal/policies/m0123_ac5_drift_test.go enforces over
// `spec.Rules()`. Re-asserting them here against
// `branch.Rules()` is the M-0158 way of saying "the layer-4
// cells join the policed set." A future refactor could merge
// the two tests by iterating the union; for now keeping them
// separate makes the layer-4 ownership explicit in failures.
func TestM0158_AC4_SchemaInvariantsHoldOverBranchCells(t *testing.T) {
	t.Parallel()

	for _, r := range indexBranchRulesByID(t) {
		if r.Outcome == spec.OutcomeUnspecified {
			t.Errorf("M-0158/AC-4: %q has Outcome=Unspecified (forgot to set Outcome in the Rule literal?)", r.ID)
		}
		if r.Outcome == spec.OutcomeIllegal && r.RejectionLayer == spec.RejectionLayerNone {
			t.Errorf("M-0158/AC-4: %q is Illegal but RejectionLayer=None (illegal cells must name where they're rejected)", r.ID)
		}
		if r.RejectionLayer == spec.RejectionLayerVerbTime && !r.BlockingStrict {
			t.Errorf("M-0158/AC-4: %q has VerbTime rejection but BlockingStrict=false (verb-time rejections always block strictly)", r.ID)
		}
		if r.Outcome == spec.OutcomeLegal && r.ExpectedErrorCode != "" {
			t.Errorf("M-0158/AC-4: %q is Legal but ExpectedErrorCode=%q (Legal cells must leave ExpectedErrorCode empty)", r.ID, r.ExpectedErrorCode)
		}
		if d := r.Sources.Decision; d != "" {
			if !looksLikeDecisionRef(d) {
				t.Errorf("M-0158/AC-4: %q has Sources.Decision=%q which does not match ADR-NNNN or D-NNNN shape", r.ID, d)
			}
		}
	}
}

// looksLikeDecisionRef returns true when the string matches the
// ADR-NNNN or D-NNNN shape — what the M-0123 catalog uses for
// Sources.Decision values.
func looksLikeDecisionRef(s string) bool {
	if strings.HasPrefix(s, "ADR-") {
		return hasDigits(s[len("ADR-"):])
	}
	if strings.HasPrefix(s, "D-") {
		return hasDigits(s[len("D-"):])
	}
	return false
}

func hasDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
