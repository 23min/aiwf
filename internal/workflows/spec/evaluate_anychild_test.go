package spec

import (
	"slices"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// TestAnyChild_ComparesParentAtCanonicalWidth: a child whose parent field
// names its epic at a different width from the epic's own id is still
// that epic's child, whichever side carries the narrow spelling.
func TestAnyChild_ComparesParentAtCanonicalWidth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		epicID      string
		childParent string
	}{
		{"narrow parent field, canonical epic", "E-0001", "E-01"},
		{"canonical parent field, legacy narrow epic", "E-02", "E-0002"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			epic := &entity.Entity{ID: tc.epicID, Kind: entity.KindEpic}
			child := &entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Parent: tc.childParent}
			stranger := &entity.Entity{ID: "M-0002", Kind: entity.KindMilestone, Parent: "E-0009"}
			tr := &tree.Tree{Root: "/test", Entities: []*entity.Entity{epic, child, stranger}}
			var seen []string
			if _, err := anyChild(tr, epic, func(c *entity.Entity) (bool, error) {
				seen = append(seen, c.ID)
				return false, nil
			}); err != nil {
				t.Fatalf("anyChild: %v", err)
			}
			if !slices.Equal(seen, []string{"M-0001"}) {
				t.Errorf("children visited = %v, want [M-0001]", seen)
			}
		})
	}
}
