package policies

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// body_section_gate_not_tree_wide_test.go — M-0331/AC-3. The push-seam
// membership gate rides a commit range, not the tree, so `aiwf check`'s
// tree-wide output is what it was before the gate landed and no entity already
// carrying an omission gains a finding.
//
// The live tree is the fixture because that is what the criterion claims about,
// and it carries the debt: entities whose bodies omit a required section, none
// of which any rule reports. An absence assertion over a tree with nothing to
// find would pass for the wrong reason, so the count is asserted alongside it —
// the two together are the claim.

// entitiesOmittingARequiredSection returns how many non-archived entities in t
// omit at least one section their kind requires, read through the same helper
// the gate and both write seams use.
func entitiesOmittingARequiredSection(t *testing.T, tr *tree.Tree) int {
	t.Helper()
	n := 0
	for _, e := range tr.Entities {
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(tr.Root, e.Path))
		if err != nil {
			continue
		}
		_, body, ok := entity.Split(raw)
		if !ok {
			continue
		}
		if len(check.AbsentRequiredSections(e.Kind, body)) > 0 {
			n++
		}
	}
	return n
}

func TestPolicy_BodySectionGateIsNotTreeWide(t *testing.T) {
	t.Parallel()
	_, tr := sharedRepoTree(t)

	omitting := entitiesOmittingARequiredSection(t, tr)
	if omitting == 0 {
		t.Fatalf("the live tree carries no entity omitting a required section, so the " +
			"absence assertion below would hold against a gate that had joined check.Run; " +
			"re-derive the claim rather than trusting this test")
	}

	for _, f := range check.Run(tr, sharedRepoTreeLoadErrs(t)) {
		if f.Code == check.CodeEntityBodySectionDropped.ID {
			t.Errorf("the tree-wide pass emitted %s on %s (%s) — the gate rides a commit "+
				"range and must not join check.Run; %d live entities omit a required "+
				"section and every one of them would gain a finding",
				f.Code, f.EntityID, f.Path, omitting)
		}
	}
}
