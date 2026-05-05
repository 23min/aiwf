package main

import (
	"context"
	"fmt"

	"github.com/23min/ai-workflow-v2/internal/check"
	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// runTestsMetricsCheck emits an `acs-tdd-tests-missing` warning for
// every AC at `tdd_phase: done` under a `tdd: required` milestone
// whose `aiwf history` carries no `aiwf-tests:` trailer on any
// commit. Gated on require: when false (the default), returns nil
// without walking history — the trailer is informational metadata
// and absence is not a finding.
//
// Why the check lives here rather than in package check: the rule
// requires git access (a history walk per qualifying AC) which the
// pure-tree check.Run intentionally does not have. Composing this
// pass in cmd/aiwf is the same shape as runProvenanceCheck: both
// take (root, tree) and produce findings, and both are stitched into
// runCheck's findings slice before sorting.
func runTestsMetricsCheck(ctx context.Context, root string, t *tree.Tree, require bool) ([]check.Finding, error) {
	if !require {
		return nil, nil
	}
	if !hasCommits(ctx, root) {
		return nil, nil
	}
	var findings []check.Finding
	for _, m := range t.ByKind(entity.KindMilestone) {
		if m.TDD != "required" {
			continue
		}
		for i := range m.ACs {
			ac := m.ACs[i]
			if ac.TDDPhase != entity.TDDPhaseDone {
				continue
			}
			compositeID := m.ID + "/" + ac.ID
			events, err := readHistory(ctx, root, compositeID)
			if err != nil {
				return nil, fmt.Errorf("history for %s: %w", compositeID, err)
			}
			if hasTestsTrailer(events) {
				continue
			}
			findings = append(findings, check.Finding{
				Code:     "acs-tdd-tests-missing",
				Severity: check.SeverityWarning,
				EntityID: compositeID,
				Path:     m.Path,
				Message: fmt.Sprintf(
					"%s is at tdd_phase: done but no commit in its history carries an aiwf-tests trailer (require_test_metrics: true)",
					compositeID),
				Hint: "re-run the cycle through `aiwf promote --phase ... --tests \"pass=N fail=N skip=N\"`, or set `tdd.require_test_metrics: false` in aiwf.yaml to silence the warning",
			})
		}
	}
	return findings, nil
}

// hasTestsTrailer reports whether any event in the history carries a
// non-nil Tests pointer (i.e. the commit's aiwf-tests trailer parsed
// successfully through the tolerant reader).
func hasTestsTrailer(events []HistoryEvent) bool {
	for i := range events {
		if events[i].Tests != nil {
			return true
		}
	}
	return false
}
