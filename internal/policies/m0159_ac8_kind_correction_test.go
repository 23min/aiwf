package policies

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/workflows/spec/branch"
)

// m0159_ac8_kind_correction_test.go — M-0159/AC-8: pin the
// branch-cell-override-f-nnnn-waiver cell's Kind value at
// "finding" per ADR-0003's seventh-kind declaration.
//
// Context: M-0158 introduced the four override cells in
// internal/workflows/spec/branch/rules.go. Three carried the
// intended kind directly (epic, milestone, etc.); the fourth —
// the F-NNNN waiver cell — incorrectly carried `Kind: "gap"` with
// an inline comment claiming "F-NNNN is registered under the gap
// kind." Per ADR-0003 §"Decision", "finding" IS the seventh
// entity kind, stored at work/findings/F-NNNN-*.md. The kind
// itself is not implemented in the PoC (entity.AllKinds() returns
// only the six existing kinds at entity.go:34), but the spec
// table's job is to catalog the override surface CORRECTLY so
// consumers reading it see the right surface name. A reader
// consulting the catalog today would learn the wrong shape; a
// future PoC milestone implementing the finding kind would have
// to track down and rename every consumer that took "gap" at face
// value.
//
// M-0158 should have landed this fix; the patch was prepared but
// never committed. M-0159/AC-8 surfaces the correctness
// regression with this pin, then lands the rules.go change to
// make it green.
//
// The test is structural — it asserts the cell's Kind value
// directly. No fixture, no git, no subprocess.

// TestM0159_AC8_FNNNNWaiverCellKindIsFinding asserts that the
// branch-cell-override-f-nnnn-waiver cell in
// internal/workflows/spec/branch/rules.go carries Kind ==
// entity.Kind("finding"), forward-declaring the correct surface
// name per ADR-0003 §"Decision" even though the kind itself is
// not yet implemented in the PoC's entity package.
func TestM0159_AC8_FNNNNWaiverCellKindIsFinding(t *testing.T) {
	t.Parallel()

	const targetID = "branch-cell-override-f-nnnn-waiver"
	const wantKind = entity.Kind("finding")

	var found bool
	for _, cell := range branch.Rules() {
		if cell.ID != targetID {
			continue
		}
		found = true
		if cell.Kind != wantKind {
			t.Errorf("M-0159/AC-8: cell %q .Kind = %q; want %q (ADR-0003 §\"Decision\" declares finding as the seventh kind, stored at work/findings/F-NNNN-*.md; the spec table forward-declares this correct surface name so future consumers don't have to re-trace the catalog after the entity-side implementation lands)",
				targetID, cell.Kind, wantKind)
		}
		break
	}
	if !found {
		t.Fatalf("M-0159/AC-8: no cell with ID %q in branch.Rules() (M-0158 introduced this cell as one of four overrides — a missing cell is a regression in the override-cell coverage that M-0158/AC-3 pins)",
			targetID)
	}
}
