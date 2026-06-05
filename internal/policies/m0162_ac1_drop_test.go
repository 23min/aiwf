package policies

import (
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec/branch"
)

// TestM0162_AC1_DropSet pins M-0162/AC-1's drop-side claim:
// 9 documentation-only / semantically-duplicate cells are ABSENT
// from `branch.Rules()` per M-0161/AC-9 body §"Part 1" and the
// M-0162/AC-1 body's enumerated drop list.
//
// Cells dropped (9):
//
//   - 5 legal-non-override documentation-only cells:
//     branch-cell-3, branch-cell-5, branch-cell-6, branch-cell-9,
//     branch-cell-11
//   - 2 legal-AND-override cells (semantic duplicates of override
//     cells): branch-cell-8, branch-cell-10
//   - 2 override-named cells (semantic duplicates of corner-case
//     cells): branch-cell-override-cherry-pick,
//     branch-cell-override-force-amend
//
// Each absence is a separate subtest so a regression that re-adds
// one of the 9 fires loudly at the offending cell.
//
// Sabotage-verifiable: re-add any of the 9 cells to `branch.Rules()`
// and the corresponding subtest fires naming the cell.
func TestM0162_AC1_DropSet(t *testing.T) {
	t.Parallel()

	dropped := []string{
		"branch-cell-3",
		"branch-cell-5",
		"branch-cell-6",
		"branch-cell-8",
		"branch-cell-9",
		"branch-cell-10",
		"branch-cell-11",
		"branch-cell-override-cherry-pick",
		"branch-cell-override-force-amend",
	}

	rules := branch.Rules()
	present := make(map[string]bool, len(rules))
	for _, r := range rules {
		present[r.ID] = true
	}

	for _, id := range dropped {
		id := id
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			if present[id] {
				t.Errorf("M-0162/AC-1: cell %q must be ABSENT from branch.Rules() (dropped per M-0161/AC-9 §\"Part 1\" + M-0162/AC-1 body)", id)
			}
		})
	}
}

// TestM0162_AC1_RetainedSet pins M-0162/AC-1's retain-side claim:
// the 7 load-bearing M-0158-era cells remain PRESENT in
// `branch.Rules()` after the AC-1 drop. Catches a future change
// that accidentally drops one of these alongside the cleanup.
//
// Retained M-0158-era cells (7):
//
//   - 5 illegal-outcome cells with real mechanical weight:
//     branch-cell-1, branch-cell-2, branch-cell-4, branch-cell-7,
//     branch-cell-12
//   - 2 standalone override cells: branch-cell-override-preflight,
//     branch-cell-override-f-nnnn-waiver
//
// Sabotage-verifiable: remove any retained cell from
// `branch.Rules()` and the corresponding subtest fires.
func TestM0162_AC1_RetainedSet(t *testing.T) {
	t.Parallel()

	retained := []string{
		"branch-cell-1",
		"branch-cell-2",
		"branch-cell-4",
		"branch-cell-7",
		"branch-cell-12",
		"branch-cell-override-preflight",
		"branch-cell-override-f-nnnn-waiver",
	}

	rules := branch.Rules()
	present := make(map[string]bool, len(rules))
	for _, r := range rules {
		present[r.ID] = true
	}

	for _, id := range retained {
		id := id
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			if !present[id] {
				t.Errorf("M-0162/AC-1: cell %q must be PRESENT in branch.Rules() (load-bearing M-0158-era cell; do not drop alongside cleanup)", id)
			}
		})
	}
}
