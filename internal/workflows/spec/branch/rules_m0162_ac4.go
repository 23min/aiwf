package branch

import "github.com/23min/aiwf/internal/workflows/spec"

// ac4MetaCells returns the 3 meta-cells that register the bijection
// enforcement chokepoints in the catalog itself (per M-0162/AC-4
// body §"Meta-cells registered"). The cells document that:
//
//   - branch-cell-meta-bijection-enforced — the 1:1 bijection between
//     branch.Rules() and branchtest.Pins() holds at CI time.
//   - branch-cell-meta-pin-orphan-detected — orphan Pin detection
//     (a Pin referencing a non-existent cell) produces a finding.
//   - branch-cell-meta-cell-orphan-detected — cell-orphan detection
//     (a cell with no Pin call site) produces a finding.
//
// Each meta-cell satisfies AC-4's own bijection invariant: the
// integration-package bijection meta-test pins each of these from
// a corresponding subtest, closing the meta-coverage loop (the
// bijection enforcer is itself a Pinned cell).
//
// Outcome=Legal: the meta-cells document a chokepoint, not a
// rule-firing condition. The behavioral assertion ("bijection
// holds at CI time") is pinned by integration's TestMain
// post-hook (see internal/cli/integration/bijection_*_test.go).
func ac4MetaCells() []spec.Rule {
	return []spec.Rule{
		{
			ID:      "branch-cell-meta-bijection-enforced",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		{
			ID:      "branch-cell-meta-pin-orphan-detected",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
		{
			ID:      "branch-cell-meta-cell-orphan-detected",
			Outcome: spec.OutcomeLegal,
			Sources: spec.RuleSource{Decision: "ADR-0010"},
		},
	}
}
