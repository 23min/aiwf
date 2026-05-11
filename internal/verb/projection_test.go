package verb

import (
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// TestProjectionFindings_PreExistingFiltered confirms that errors
// already present on the original tree do not surface as "introduced
// by this verb." This is the load-bearing behavior for item 15: a
// verb shouldn't refuse to run because of unrelated pre-existing
// breakage.
func TestProjectionFindings_PreExistingFiltered(t *testing.T) {
	original := &tree.Tree{
		Entities: []*entity.Entity{
			// Pre-existing refs-resolve error: gap points to non-existent milestone.
			{
				ID: "G-0001", Kind: entity.KindGap, Title: "Broken", Status: "open",
				DiscoveredIn: "M-0999", Path: "g.md",
			},
		},
	}
	projected := &tree.Tree{
		Entities: []*entity.Entity{
			{
				ID: "G-0001", Kind: entity.KindGap, Title: "Broken", Status: "open",
				DiscoveredIn: "M-0999", Path: "g.md",
			},
			// Unrelated new epic — should not be blocked. Paired with a
			// drafted milestone so the M-0094 rule
			// (epic-active-no-drafted-milestones) does not fire and
			// muddy this test's premise.
			{
				ID: "E-0001", Kind: entity.KindEpic, Title: "Foundations", Status: "active",
				Path: "e.md",
			},
			{
				ID: "M-0001", Kind: entity.KindMilestone, Title: "Queued", Status: "draft",
				Parent: "E-0001", Path: "m.md",
			},
		},
	}

	got := projectionFindings(original, projected)
	if len(got) != 0 {
		t.Errorf("expected zero introduced findings, got: %+v", got)
	}
}

// TestProjectionFindings_NewErrorIntroduced is the complement: a
// finding present only in the projected tree surfaces as introduced.
func TestProjectionFindings_NewErrorIntroduced(t *testing.T) {
	original := &tree.Tree{
		Entities: []*entity.Entity{
			{
				ID: "E-0001", Kind: entity.KindEpic, Title: "OK", Status: "active",
				Path: "work/epics/E-01-ok/epic.md",
			},
		},
	}
	projected := &tree.Tree{
		Entities: []*entity.Entity{
			{
				ID: "E-0001", Kind: entity.KindEpic, Title: "OK", Status: "active",
				Path: "work/epics/E-01-ok/epic.md",
			},
			// New milestone with a bad parent — error introduced by the verb.
			{
				ID: "M-0001", Kind: entity.KindMilestone, Title: "Bad ref", Status: "draft",
				Parent: "E-0099", Path: "work/epics/E-01-ok/M-001.md",
			},
		},
	}

	got := projectionFindings(original, projected)
	if !check.HasErrors(got) {
		t.Errorf("expected introduced error, got: %+v", got)
	}
}

// TestProjectionFindings_PreExistingPlusNew confirms that a tree with
// existing problems plus a verb that introduces a *new* error surfaces
// only the new one (the existing ones are filtered).
func TestProjectionFindings_PreExistingPlusNew(t *testing.T) {
	original := &tree.Tree{
		Entities: []*entity.Entity{
			// Pre-existing issue.
			{
				ID: "G-0001", Kind: entity.KindGap, Title: "Old broken", Status: "open",
				DiscoveredIn: "M-0999", Path: "g.md",
			},
		},
	}
	projected := &tree.Tree{
		Entities: []*entity.Entity{
			{
				ID: "G-0001", Kind: entity.KindGap, Title: "Old broken", Status: "open",
				DiscoveredIn: "M-0999", Path: "g.md",
			},
			// New issue: a different gap, also broken.
			{
				ID: "G-0002", Kind: entity.KindGap, Title: "New broken", Status: "open",
				DiscoveredIn: "M-0888", Path: "g2.md",
			},
		},
	}

	got := projectionFindings(original, projected)
	if len(got) != 1 {
		t.Fatalf("expected exactly one introduced finding, got %d: %+v", len(got), got)
	}
	if got[0].EntityID != "G-0002" {
		t.Errorf("introduced finding entity = %q, want G-002", got[0].EntityID)
	}
}
