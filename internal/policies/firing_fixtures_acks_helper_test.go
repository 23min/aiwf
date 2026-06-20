package policies

import (
	"path/filepath"
	"testing"
)

// TestFiringFixtures_AcksHelperLift is the G-0262 burn-down positive
// control for acks-helper-lift (M-0166/AC-3), the heaviest policy: it
// traces ackedSHAs identifier provenance across internal/check/ and
// internal/cli/check/ and has 11 dark Violation construction lines across
// structural, call-cardinality, and provenance classes. Each row drives a
// distinct class; coverage confirms every dark line is lit. (A single
// fixture may light several lines — extra violations are harmless; the
// rows are organized by the class each one targets.)
func TestFiringFixtures_AcksHelperLift(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		files map[string]string
	}{
		// Empty tree: acks.go missing, CLI gather dir missing, zero
		// WalkAcknowledgedSHAs call sites.
		{name: "empty", files: map[string]string{}},

		// acks.go present without the WalkAcknowledgedSHAs decl;
		// fsm_history_consistent.go still declares the lifted walker;
		// a rule-internal recompute call.
		{
			name: "no-decl+leftover-walker+internal-recompute",
			files: map[string]string{
				"internal/check/acks.go":                   "package check\n",
				"internal/check/fsm_history_consistent.go": "package check\n\nfunc walkAcknowledgedSHAs() map[string]bool { return nil }\n",
				"internal/check/recompute.go":              "package check\n\nfunc r() { _ = WalkAcknowledgedSHAs() }\n",
				// CLI gather dir present so the policy does not early-return
				// (the recompute scan is gated behind hasCliCheck).
				"internal/cli/check/gather.go": "package check\n",
			},
		},

		// acks.go has the decl; CLI gather calls it more than once.
		{
			name: "multiple-call-sites",
			files: map[string]string{
				"internal/check/acks.go":       "package check\n\nfunc WalkAcknowledgedSHAs() map[string]bool { return nil }\n",
				"internal/cli/check/gather.go": "package check\n\nfunc g() {\n\t_ = check.WalkAcknowledgedSHAs()\n\t_ = check.WalkAcknowledgedSHAs()\n}\n",
			},
		},

		// Consumer called with an `ackedSHAs` arg that has no provenance
		// in the enclosing function.
		{
			name: "consumer-no-provenance",
			files: map[string]string{
				"internal/cli/check/gather.go": "package check\n\nfunc g() { _ = check.FSMHistoryConsistent(ackedSHAs) }\n",
			},
		},

		// Consumer called without an `ackedSHAs` arg at all.
		{
			name: "consumer-no-arg",
			files: map[string]string{
				"internal/cli/check/gather.go": "package check\n\nfunc g() { _ = check.FSMHistoryConsistent(somethingElse) }\n",
			},
		},

		// Consumer FuncDecl has the ackedSHAs parameter but the body
		// never reads it.
		{
			name: "consumer-body-ignores-param",
			files: map[string]string{
				"internal/check/consumer.go":   "package check\n\nfunc FSMHistoryConsistent(ackedSHAs map[string]bool) []int { return nil }\n",
				"internal/cli/check/gather.go": "package check\n",
			},
		},

		// A forwarder calls a leaf predicate with a non-ackedSHAs arg at
		// the tracked position.
		{
			name: "forwarder-drops-predicate-arg",
			files: map[string]string{
				"internal/check/forwarder.go":  "package check\n\nfunc r(obs []int) { _ = illegalTransitionFindings(obs, nil) }\n",
				"internal/cli/check/gather.go": "package check\n",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for rel, content := range tc.files {
				mustWrite(t, filepath.Join(root, rel), content)
			}
			vs, err := PolicyAcksHelperLift(root)
			if err != nil {
				t.Fatalf("%s: policy returned error: %v", tc.name, err)
			}
			if !hasPolicyViolation(vs, "acks-helper-lift") {
				t.Errorf("%s: acks-helper-lift did not fire on its fixture; got %d violations", tc.name, len(vs))
			}
		})
	}
}
