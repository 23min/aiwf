package roadmap

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

func areaTree() *tree.Tree {
	return &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-0001", Title: "Platform epic", Status: "active", Area: "platform"},
			{Kind: entity.KindEpic, ID: "E-0002", Title: "Billing epic", Status: "active", Area: "billing"},
			{Kind: entity.KindEpic, ID: "E-0003", Title: "Untagged epic", Status: "active"},
			{Kind: entity.KindMilestone, ID: "M-0001", Title: "Cache", Status: "done", Parent: "E-0001"},
		},
	}
}

func before(t *testing.T, s, a, b string) {
	t.Helper()
	ia, ib := strings.Index(s, a), strings.Index(s, b)
	if ia < 0 || ib < 0 || ia >= ib {
		t.Errorf("expected %q before %q (got %d, %d):\n%s", a, b, ia, ib, s)
	}
}

// TestRenderGrouped_ByArea pins M-0175/AC-3: render roadmap groups epics
// into per-area sections (declared members in order, the untagged
// complement last under the default label), with epics demoted to h3
// under their area's h2 heading.
func TestRenderGrouped_ByArea(t *testing.T) {
	t.Parallel()
	got := string(RenderGrouped(areaTree(), []string{"platform", "billing"}, "Uncategorized"))

	before(t, got, "## platform", "### E-0001")
	before(t, got, "### E-0001", "## billing")
	before(t, got, "## billing", "### E-0002")
	before(t, got, "## billing", "## Uncategorized")
	before(t, got, "## Uncategorized", "### E-0003")
	// Epic heading demoted to h3 under the area h2 (the milestone table
	// still rides along under the epic).
	if !strings.Contains(got, "### E-0001 — Platform epic (active)") {
		t.Errorf("grouped epic should be an h3 heading:\n%s", got)
	}
	if !strings.Contains(got, "| M-0001 | Cache | done |") {
		t.Errorf("grouped epic should keep its milestone table:\n%s", got)
	}
}

// TestRenderGrouped_EmptyDeclaredSuppressed pins M-0175/AC-5 on the
// roadmap: a declared area with no epics is omitted; the complement is
// always shown.
func TestRenderGrouped_EmptyDeclaredSuppressed(t *testing.T) {
	t.Parallel()
	got := string(RenderGrouped(areaTree(), []string{"platform", "billing", "tooling"}, "Uncategorized"))
	if strings.Contains(got, "## tooling") {
		t.Errorf("empty declared area 'tooling' must be suppressed:\n%s", got)
	}
	if !strings.Contains(got, "## Uncategorized") {
		t.Errorf("complement must always be shown:\n%s", got)
	}
}

// TestRenderGrouped_EmptyComplementShown pins M-0175/AC-5 on the roadmap:
// when every epic is tagged, the untagged complement section is still
// rendered (with its "no epics" message), not suppressed.
func TestRenderGrouped_EmptyComplementShown(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-0001", Title: "Platform epic", Status: "active", Area: "platform"},
		},
	}
	got := string(RenderGrouped(tr, []string{"platform"}, "Uncategorized"))
	if !strings.Contains(got, "## Uncategorized") {
		t.Errorf("empty complement must still be rendered:\n%s", got)
	}
	if !strings.Contains(got, "_No epics in this area._") {
		t.Errorf("empty complement should carry the no-epics message:\n%s", got)
	}
}

// TestRenderGrouped_NoAreasMatchesFlat pins M-0175/AC-6 on the roadmap:
// with no declared members, the grouped entry point produces output
// byte-identical to the flat Render (zero-migration).
func TestRenderGrouped_NoAreasMatchesFlat(t *testing.T) {
	t.Parallel()
	tr := areaTree()
	if grouped, flat := string(RenderGrouped(tr, nil, "")), string(Render(tr)); grouped != flat {
		t.Errorf("RenderGrouped with no areas must equal flat Render:\n--- grouped ---\n%s\n--- flat ---\n%s", grouped, flat)
	}
}
