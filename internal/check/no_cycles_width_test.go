package check

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/entity"
)

// TestNoCycles_ComparesIDsAtCanonicalWidth: a depends_on cycle is found
// whichever side of an edge is stored at a legacy width, and every
// finding names its milestone at canonical width.
func TestNoCycles_ComparesIDsAtCanonicalWidth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		entities []*entity.Entity
	}{
		{"narrow edge, canonical node", []*entity.Entity{
			{ID: "M-0002", Kind: entity.KindMilestone, DependsOn: []string{"M-003"}, Path: "2.md"},
			{ID: "M-0003", Kind: entity.KindMilestone, DependsOn: []string{"M-0002"}, Path: "3.md"},
		}},
		{"canonical edge, narrow node", []*entity.Entity{
			{ID: "M-0002", Kind: entity.KindMilestone, DependsOn: []string{"M-0003"}, Path: "2.md"},
			{ID: "M-003", Kind: entity.KindMilestone, DependsOn: []string{"M-0002"}, Path: "3.md"},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var ids []string
			for _, f := range noCycles(makeTree(tc.entities...)) {
				ids = append(ids, f.EntityID)
			}
			sort.Strings(ids)
			if diff := cmp.Diff([]string{"M-0002", "M-0003"}, ids); diff != "" {
				t.Errorf("cycle findings (-want +got):\n%s", diff)
			}
		})
	}
}
