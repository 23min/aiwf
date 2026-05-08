package entity

import "testing"

func TestAllocateID_FirstOfKind(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{KindEpic, "E-01"},
		{KindMilestone, "M-001"},
		{KindADR, "ADR-0001"},
		{KindGap, "G-001"},
		{KindDecision, "D-001"},
		{KindContract, "C-001"},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			got := AllocateID(tt.kind, nil, nil)
			if got != tt.want {
				t.Errorf("AllocateID(%s, empty, empty) = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

func TestAllocateID_IncrementsMax(t *testing.T) {
	entities := []*Entity{
		{ID: "E-01", Kind: KindEpic},
		{ID: "E-03", Kind: KindEpic},
		{ID: "E-02", Kind: KindEpic},
		{ID: "M-001", Kind: KindMilestone},
	}
	if got := AllocateID(KindEpic, entities, nil); got != "E-04" {
		t.Errorf("epic allocate = %q, want E-04", got)
	}
	if got := AllocateID(KindMilestone, entities, nil); got != "M-002" {
		t.Errorf("milestone allocate = %q, want M-002", got)
	}
}

func TestAllocateID_IgnoresOtherKinds(t *testing.T) {
	// A milestone with id M-007 should not influence epic numbering.
	entities := []*Entity{
		{ID: "M-007", Kind: KindMilestone},
	}
	if got := AllocateID(KindEpic, entities, nil); got != "E-01" {
		t.Errorf("got %q, want E-01", got)
	}
}

func TestAllocateID_GrowsPastPadWidth(t *testing.T) {
	entities := []*Entity{
		{ID: "E-99", Kind: KindEpic},
	}
	// E-100 is 3 digits, exceeding the pad width of 2 — fmt.Sprintf
	// with %0*d does not truncate, so this should grow naturally.
	if got := AllocateID(KindEpic, entities, nil); got != "E-100" {
		t.Errorf("got %q, want E-100", got)
	}
}

func TestAllocateID_TolerantOfBadIds(t *testing.T) {
	// Malformed id should not crash the allocator (frontmatter-shape
	// surfaces the bookkeeping error separately).
	entities := []*Entity{
		{ID: "E-01", Kind: KindEpic},
		{ID: "not-an-id", Kind: KindEpic},
		{ID: "", Kind: KindEpic},
	}
	if got := AllocateID(KindEpic, entities, nil); got != "E-02" {
		t.Errorf("got %q, want E-02 (the bad ids should be ignored)", got)
	}
}

// G37 — trunk-aware allocator. The third argument carries ids
// observed in the configured trunk ref's tree; the allocator picks
// max+1 across the union with the working tree.

func TestAllocateID_TrunkOnly(t *testing.T) {
	// Working tree empty, trunk has E-05 — next epic id must be E-06.
	got := AllocateID(KindEpic, nil, []string{"E-05"})
	if got != "E-06" {
		t.Errorf("trunk-only allocate = %q, want E-06", got)
	}
}

func TestAllocateID_TrunkAheadOfWorkingTree(t *testing.T) {
	// Forgot-to-fetch case: working tree shows E-02 as the highest,
	// but trunk has already moved on to E-07. The allocator unions
	// both and skips past trunk.
	entities := []*Entity{{ID: "E-02", Kind: KindEpic}}
	got := AllocateID(KindEpic, entities, []string{"E-04", "E-07"})
	if got != "E-08" {
		t.Errorf("trunk-ahead allocate = %q, want E-08", got)
	}
}

func TestAllocateID_WorkingTreeAheadOfTrunk(t *testing.T) {
	// Local has already gone past trunk — common during feature work.
	// Allocator picks the local max+1, ignoring the smaller trunk
	// values.
	entities := []*Entity{
		{ID: "E-01", Kind: KindEpic},
		{ID: "E-09", Kind: KindEpic},
	}
	got := AllocateID(KindEpic, entities, []string{"E-03"})
	if got != "E-10" {
		t.Errorf("local-ahead allocate = %q, want E-10", got)
	}
}

func TestAllocateID_TrunkIDsKindFiltered(t *testing.T) {
	// Trunk ids of other kinds should not affect this kind's allocation.
	got := AllocateID(KindGap, nil, []string{"E-99", "M-99", "ADR-9999"})
	if got != "G-001" {
		t.Errorf("got %q, want G-001 (other-kind trunk ids ignored)", got)
	}
}
