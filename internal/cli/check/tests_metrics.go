// Package check holds the check verb's body and the verb-level
// check-rule helpers that need git access. The pure-tree rules live
// in internal/check; this package composes them with history walks
// (runTestsMetricsCheck, runProvenanceCheck) and renders the
// final output envelope.
package check

import (
	"context"
	"fmt"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/history"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// RunTestsMetricsCheck emits an `acs-tdd-tests-missing` warning for
// every AC at `tdd_phase: done` under a `tdd: required` milestone
// whose `aiwf history` carries no `aiwf-tests:` trailer on any
// commit. Gated on require: when false (the default), returns nil
// without walking history — the trailer is informational metadata
// and absence is not a finding.
//
// Why the check lives here rather than in package check: the rule
// requires git access (a history walk per qualifying AC) which the
// pure-tree check.Run intentionally does not have. Composing this
// pass in the check verb's package keeps the rule's runtime cost
// scoped to invocations that opt in via aiwf.yaml.
func RunTestsMetricsCheck(ctx context.Context, root string, t *tree.Tree, require bool) ([]check.Finding, error) {
	if !require {
		return nil, nil
	}
	if !cliutil.HasCommits(ctx, root) {
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
			events, err := history.ReadHistory(ctx, root, compositeID)
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
func hasTestsTrailer(events []history.HistoryEvent) bool {
	for i := range events {
		if events[i].Tests != nil {
			return true
		}
	}
	return false
}
