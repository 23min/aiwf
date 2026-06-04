package policies

import (
	"fmt"
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec"
	"github.com/23min/aiwf/internal/workflows/spec/branch"
)

// TestM0158_AC2_RetainedCornerCellsPresent pins the residual
// M-0158/AC-2 claim after the M-0162/AC-1 refinement: the 5 corner
// cells with mechanical weight (1, 2, 4, 7, 12 — the illegal-outcome
// cells with non-empty ExpectedErrorCode) are registered as
// `branch-cell-N` in `branch.Rules()`.
//
// M-0158/AC-2 originally claimed all 12 numbered corner cases were
// registered with 1:1 id-to-number traceability. M-0162/AC-1 drops
// 7 of those (3, 5, 6, 8, 9, 10, 11) as documentation-only or
// semantic duplicates per M-0161/AC-9 §"Part 1"; the M-0158/AC-2
// promoted-met status remains valid because the original 12-cell
// catalog landed correctly at M-0158 wrap time. This test tracks
// the current catalog state — the AC-1 drop list is independently
// pinned by TestM0162_AC1_DropSet.
func TestM0158_AC2_RetainedCornerCellsPresent(t *testing.T) {
	t.Parallel()

	retainedCornerCells := []int{1, 2, 4, 7, 12}
	byID := indexBranchRulesByID(t)
	for _, n := range retainedCornerCells {
		id := fmt.Sprintf("branch-cell-%d", n)
		if _, ok := byID[id]; !ok {
			t.Errorf("M-0158/AC-2 + M-0162/AC-1: branch.Rules() missing %q (corner case %d, retained per M-0162/AC-1 cleanup)", id, n)
		}
	}
}

// TestM0158_AC2_CornerCellOutcomesMatchEpic pins the spec-side
// alignment between cell.Outcome and the epic body's enumerated
// outcomes ("preflight rejects" / "finding fires" / "finding
// silent"). A regression that flipped an Outcome (e.g., marking
// branch-cell-4 as Legal) would fire this test even before the
// behavioral tests under internal/verb/ + internal/check/ caught
// the runtime drift.
//
// The expected-outcome map is the canonical M-0158 source: it
// encodes the user's directive *"these corner cases become part
// of these verifications"* as a per-cell legality assertion.
func TestM0158_AC2_CornerCellOutcomesMatchEpic(t *testing.T) {
	t.Parallel()

	// Post-M-0162/AC-1 retained corner cells only. Entries for cells
	// 3, 5, 6, 8, 9, 10, 11 removed alongside their catalog entries;
	// the original M-0158/AC-2 met-status remains valid (the
	// outcomes-matched-epic claim landed correctly for all 12 cells
	// at M-0158 wrap time).
	want := map[int]spec.Outcome{
		1:  spec.OutcomeIllegal, // AI authorize on main no --branch → refused
		2:  spec.OutcomeIllegal, // AI authorize --branch <typo> → refused
		4:  spec.OutcomeIllegal, // AI commit on main while bound to epic → fires
		7:  spec.OutcomeIllegal, // AI commit on different epic → fires
		12: spec.OutcomeIllegal, // worktree-vs-branch mismatch → fires
	}

	byID := indexBranchRulesByID(t)
	for n, wantOutcome := range want {
		id := fmt.Sprintf("branch-cell-%d", n)
		got, ok := byID[id]
		if !ok {
			continue // covered by RetainedCornerCellsPresent above; don't double-report
		}
		if got.Outcome != wantOutcome {
			t.Errorf("M-0158/AC-2: %s.Outcome = %v; want %v (per E-0030 epic §\"Corner cases\" #%d)", id, got.Outcome, wantOutcome, n)
		}
	}
}

// TestM0158_AC2_IllegalCellsCarryExpectedErrorCode asserts that
// every Illegal corner-case cell carries an ExpectedErrorCode
// matching the cell's actual kernel emission point. The triplet
// (branch-context-required, branch-not-found, isolation-escape)
// covers the layer-4 illegal cells; a regression that left an
// Illegal cell's code empty would surface here.
func TestM0158_AC2_IllegalCellsCarryExpectedErrorCode(t *testing.T) {
	t.Parallel()

	wantCode := map[int]string{
		1:  "branch-context-required",
		2:  "branch-not-found",
		4:  "isolation-escape",
		7:  "isolation-escape",
		12: "isolation-escape",
	}

	byID := indexBranchRulesByID(t)
	for n, want := range wantCode {
		id := fmt.Sprintf("branch-cell-%d", n)
		got, ok := byID[id]
		if !ok {
			continue
		}
		if got.ExpectedErrorCode != want {
			t.Errorf("M-0158/AC-2: %s.ExpectedErrorCode = %q; want %q", id, got.ExpectedErrorCode, want)
		}
	}
}

// indexBranchRulesByID returns branch.Rules() keyed by Rule.ID.
// Helper used by all M-0158 cell-existence tests. The function
// fatals if any branch.Rules() entry has an empty ID — a
// regression there would silently mask the presence assertions
// in the tests that use this helper.
func indexBranchRulesByID(t *testing.T) map[string]spec.Rule {
	t.Helper()
	out := map[string]spec.Rule{}
	rules := branch.Rules()
	for i := range rules {
		r := &rules[i]
		if r.ID == "" {
			t.Fatalf("branch.Rules() contains entry with empty ID: %+v", r)
		}
		if _, dup := out[r.ID]; dup {
			t.Fatalf("branch.Rules() contains duplicate ID %q", r.ID)
		}
		out[r.ID] = *r
	}
	return out
}
