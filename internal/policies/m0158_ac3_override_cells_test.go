package policies

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec"
)

// TestM0158_AC3_RetainedOverrideCellsPresent pins the residual
// M-0158/AC-3 claim after the M-0162/AC-1 refinement: the 2 standalone
// override-surface cells (preflight, f-nnnn-waiver) are registered as
// named cells in `branch.Rules()`.
//
// M-0158/AC-3 originally claimed all 4 override-surface mechanisms
// were registered. M-0162/AC-1 drops `branch-cell-override-cherry-pick`
// and `branch-cell-override-force-amend` as semantic duplicates of
// corner-case cells 8 and 10 (themselves also dropped); the kernel's
// underlying cherry-pick suppression and aiwf-force trailer override
// mechanisms remain implemented in the rules engine — only the
// catalog redundancy is what was redundant. The M-0158/AC-3
// promoted-met status remains valid because the original 4-cell
// catalog landed correctly at M-0158 wrap time. This test tracks
// the current catalog state — the AC-1 drop list is independently
// pinned by TestM0162_AC1_DropSet.
//
// The pre-dispatch row (session-layer PreToolUse hook) does NOT
// have a cell because it lives outside the kernel's reach —
// session-layer hooks are not legality-pertinent at the kernel
// surface, per epic §"Sovereign override surface" line 97.
func TestM0158_AC3_RetainedOverrideCellsPresent(t *testing.T) {
	t.Parallel()

	want := []string{
		"branch-cell-override-preflight",
		"branch-cell-override-f-nnnn-waiver",
	}
	byID := indexBranchRulesByID(t)
	for _, id := range want {
		if _, ok := byID[id]; !ok {
			t.Errorf("M-0158/AC-3 + M-0162/AC-1: branch.Rules() missing %q (override mechanism, retained per M-0162/AC-1 cleanup)", id)
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
