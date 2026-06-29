package tree

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/trunk"
)

// AllocationIDs (M-0212) is the allocator's broadened cross-branch
// view: the union of the configured trunk ref's ids and every local
// branch ref's ids. It is deliberately broader than TrunkIDs alone,
// which the ids-unique check keeps reading directly (E-0052 decision:
// the widened set feeds allocation only, never the uniqueness check).

func TestTree_AllocationIDs_UnionsTrunkAndLocalRefs(t *testing.T) {
	t.Parallel()
	tr := &Tree{
		TrunkIDs:    []trunk.ID{{Kind: entity.KindGap, ID: "G-0003", Path: "work/gaps/G-0003-x.md"}},
		LocalRefIDs: []string{"G-0007", "G-0003"},
	}
	got := tr.AllocationIDs()
	// trunk ids first, then local-ref ids; duplicates are harmless
	// because AllocateID takes the max.
	want := []string{"G-0003", "G-0007", "G-0003"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("AllocationIDs mismatch (-want +got):\n%s", diff)
	}
}

func TestTree_AllocationIDs_UnionsTrunkLocalAndRemoteRefs(t *testing.T) {
	t.Parallel()
	tr := &Tree{
		TrunkIDs:     []trunk.ID{{Kind: entity.KindGap, ID: "G-0003", Path: "work/gaps/G-0003-x.md"}},
		LocalRefIDs:  []string{"G-0007"},
		RemoteRefIDs: []string{"G-0011"},
	}
	got := tr.AllocationIDs()
	// trunk, then local-ref, then remote-ref ids; duplicates harmless.
	want := []string{"G-0003", "G-0007", "G-0011"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("AllocationIDs mismatch (-want +got):\n%s", diff)
	}
}

func TestTree_AllocationIDs_NoLocalRefs_IsTrunkOnly(t *testing.T) {
	t.Parallel()
	tr := &Tree{
		TrunkIDs: []trunk.ID{{Kind: entity.KindGap, ID: "G-0003", Path: "work/gaps/G-0003-x.md"}},
	}
	got := tr.AllocationIDs()
	want := []string{"G-0003"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("AllocationIDs mismatch (-want +got):\n%s", diff)
	}
}

func TestTree_AllocationIDs_FeedsAllocatorPastSiblingID(t *testing.T) {
	t.Parallel()
	// Working tree carries G-0001; a sibling local branch carries
	// G-0009. The allocator must skip past the sibling id to G-0010,
	// not hand back G-0002.
	tr := &Tree{
		Entities:    []*entity.Entity{{Kind: entity.KindGap, ID: "G-0001"}},
		LocalRefIDs: []string{"G-0009"},
	}
	got := entity.AllocateID(entity.KindGap, tr.Entities, tr.AllocationIDs())
	if got != "G-0010" {
		t.Errorf("AllocateID = %q, want G-0010 (past sibling-branch G-0009)", got)
	}
}
