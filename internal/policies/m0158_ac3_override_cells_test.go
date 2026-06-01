package policies

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec"
)

// TestM0158_AC3_FourOverrideCellsPresent pins M-0158/AC-3: every
// override-surface row from E-0030 §"Sovereign override surface"
// (pre-dispatch / at-dispatch / post-hoc / at-check) is registered
// as a named cell with id `branch-cell-override-<mechanism>`.
//
// The 4 mechanisms map to the layered defense in depth E-0030
// commits to:
//
//   - preflight    — M-0103 verb-time --force --reason override
//   - cherry-pick  — M-0106 check-time committer-vs-actor + marker
//   - force-amend  — M-0106 check-time aiwf-force trailer amend
//   - f-nnnn-waiver — at-check ADR-0003 waiver pattern
//
// The pre-dispatch row (session-layer PreToolUse hook) does NOT
// have a cell because it lives outside the kernel's reach —
// session-layer hooks are not legality-pertinent at the kernel
// surface, per epic §"Sovereign override surface" line 97.
func TestM0158_AC3_FourOverrideCellsPresent(t *testing.T) {
	t.Parallel()

	want := []string{
		"branch-cell-override-preflight",
		"branch-cell-override-cherry-pick",
		"branch-cell-override-force-amend",
		"branch-cell-override-f-nnnn-waiver",
	}
	byID := indexBranchRulesByID(t)
	for _, id := range want {
		if _, ok := byID[id]; !ok {
			t.Errorf("M-0158/AC-3: branch.Rules() missing %q (override mechanism per E-0030 epic §\"Sovereign override surface\")", id)
		}
	}
}

// TestM0158_AC3_AllOverrideCellsAreLegalOutcome asserts the
// structural invariant that every override cell carries
// OutcomeLegal: an override IS the legitimate acceptance path,
// not an additional illegal cell. A regression that marked an
// override cell as Illegal would invert the catalog's semantic.
func TestM0158_AC3_AllOverrideCellsAreLegalOutcome(t *testing.T) {
	t.Parallel()

	byID := indexBranchRulesByID(t)
	for id, rule := range byID {
		if !strings.HasPrefix(id, "branch-cell-override-") {
			continue
		}
		if rule.Outcome != spec.OutcomeLegal {
			t.Errorf("M-0158/AC-3: %s.Outcome = %v; want %v (override cells are legitimate acceptance paths, not illegal)", id, rule.Outcome, spec.OutcomeLegal)
		}
		if rule.ExpectedErrorCode != "" {
			t.Errorf("M-0158/AC-3: %s.ExpectedErrorCode = %q; want empty (Legal cells must leave ExpectedErrorCode empty)", id, rule.ExpectedErrorCode)
		}
		if rule.RejectionLayer != spec.RejectionLayerNone {
			t.Errorf("M-0158/AC-3: %s.RejectionLayer = %v; want %v (Legal cells have no rejection)", id, rule.RejectionLayer, spec.RejectionLayerNone)
		}
	}
}
