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
			got := AllocateID(tt.kind, nil)
			if got != tt.want {
				t.Errorf("AllocateID(%s, empty) = %q, want %q", tt.kind, got, tt.want)
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
	if got := AllocateID(KindEpic, entities); got != "E-04" {
		t.Errorf("epic allocate = %q, want E-04", got)
	}
	if got := AllocateID(KindMilestone, entities); got != "M-002" {
		t.Errorf("milestone allocate = %q, want M-002", got)
	}
}

func TestAllocateID_IgnoresOtherKinds(t *testing.T) {
	// A milestone with id M-007 should not influence epic numbering.
	entities := []*Entity{
		{ID: "M-007", Kind: KindMilestone},
	}
	if got := AllocateID(KindEpic, entities); got != "E-01" {
		t.Errorf("got %q, want E-01", got)
	}
}

func TestAllocateID_GrowsPastPadWidth(t *testing.T) {
	entities := []*Entity{
		{ID: "E-99", Kind: KindEpic},
	}
	// E-100 is 3 digits, exceeding the pad width of 2 — fmt.Sprintf
	// with %0*d does not truncate, so this should grow naturally.
	if got := AllocateID(KindEpic, entities); got != "E-100" {
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
	if got := AllocateID(KindEpic, entities); got != "E-02" {
		t.Errorf("got %q, want E-02 (the bad ids should be ignored)", got)
	}
}
