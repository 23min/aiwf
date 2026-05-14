package entity

import "testing"

func TestAllocateID_FirstOfKind(t *testing.T) {
	t.Parallel()
	// Per AC-1 in M-081 (canonicalized via ADR-0008), the allocator
	// emits canonical 4-digit width for every kind on the first
	// allocation.
	tests := []struct {
		kind Kind
		want string
	}{
		{KindEpic, "E-0001"},
		{KindMilestone, "M-0001"},
		{KindADR, "ADR-0001"},
		{KindGap, "G-0001"},
		{KindDecision, "D-0001"},
		{KindContract, "C-0001"},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			t.Parallel()
			got := AllocateID(tt.kind, nil, nil)
			if got != tt.want {
				t.Errorf("AllocateID(%s, empty, empty) = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

func TestAllocateID_IncrementsMax(t *testing.T) {
	t.Parallel()
	// Narrow legacy on-disk ids are tolerated by the allocator's
	// parseIDNumber (AC-2 parser-tolerance); the emitted next-id is
	// canonical (AC-1).
	entities := []*Entity{
		{ID: "E-01", Kind: KindEpic},
		{ID: "E-03", Kind: KindEpic},
		{ID: "E-02", Kind: KindEpic},
		{ID: "M-001", Kind: KindMilestone},
	}
	if got := AllocateID(KindEpic, entities, nil); got != "E-0004" {
		t.Errorf("epic allocate = %q, want E-0004", got)
	}
	if got := AllocateID(KindMilestone, entities, nil); got != "M-0002" {
		t.Errorf("milestone allocate = %q, want M-0002", got)
	}
}

func TestAllocateID_IgnoresOtherKinds(t *testing.T) {
	t.Parallel()
	// A milestone with id M-007 should not influence epic numbering.
	entities := []*Entity{
		{ID: "M-007", Kind: KindMilestone},
	}
	if got := AllocateID(KindEpic, entities, nil); got != "E-0001" {
		t.Errorf("got %q, want E-0001", got)
	}
}

func TestAllocateID_GrowsPastPadWidth(t *testing.T) {
	t.Parallel()
	entities := []*Entity{
		{ID: "E-99", Kind: KindEpic},
	}
	// E-0100 — past the pad width but emitted at natural width.
	// fmt.Sprintf with %0*d does not truncate; the canonical pad is
	// a minimum, not a maximum.
	if got := AllocateID(KindEpic, entities, nil); got != "E-0100" {
		t.Errorf("got %q, want E-0100", got)
	}
}

func TestAllocateID_TolerantOfBadIds(t *testing.T) {
	t.Parallel()
	// Malformed id should not crash the allocator (frontmatter-shape
	// surfaces the bookkeeping error separately).
	entities := []*Entity{
		{ID: "E-01", Kind: KindEpic},
		{ID: "not-an-id", Kind: KindEpic},
		{ID: "", Kind: KindEpic},
	}
	if got := AllocateID(KindEpic, entities, nil); got != "E-0002" {
		t.Errorf("got %q, want E-0002 (the bad ids should be ignored)", got)
	}
}

// G37 — trunk-aware allocator. The third argument carries ids
// observed in the configured trunk ref's tree; the allocator picks
// max+1 across the union with the working tree.

func TestAllocateID_TrunkOnly(t *testing.T) {
	t.Parallel()
	// Working tree empty, trunk has E-05 — next epic id must be
	// canonical E-0006 per AC-1 in M-081 (allocator always emits
	// canonical 4-digit width). Narrow legacy trunk ids are
	// tolerated by parseIDNumber (AC-2 parser-tolerance).
	got := AllocateID(KindEpic, nil, []string{"E-05"})
	if got != "E-0006" {
		t.Errorf("trunk-only allocate = %q, want E-0006", got)
	}
}

func TestAllocateID_TrunkAheadOfWorkingTree(t *testing.T) {
	t.Parallel()
	// Forgot-to-fetch case: working tree shows E-02 as the highest,
	// but trunk has already moved on to E-07. The allocator unions
	// both and skips past trunk; canonical emission per AC-1.
	entities := []*Entity{{ID: "E-02", Kind: KindEpic}}
	got := AllocateID(KindEpic, entities, []string{"E-04", "E-07"})
	if got != "E-0008" {
		t.Errorf("trunk-ahead allocate = %q, want E-0008", got)
	}
}

func TestAllocateID_WorkingTreeAheadOfTrunk(t *testing.T) {
	t.Parallel()
	// Local has already gone past trunk — common during feature work.
	// Allocator picks the local max+1, ignoring the smaller trunk
	// values. Canonical emission per AC-1.
	entities := []*Entity{
		{ID: "E-01", Kind: KindEpic},
		{ID: "E-09", Kind: KindEpic},
	}
	got := AllocateID(KindEpic, entities, []string{"E-03"})
	if got != "E-0010" {
		t.Errorf("local-ahead allocate = %q, want E-0010", got)
	}
}

func TestAllocateID_TrunkIDsKindFiltered(t *testing.T) {
	t.Parallel()
	// Trunk ids of other kinds should not affect this kind's allocation.
	got := AllocateID(KindGap, nil, []string{"E-99", "M-99", "ADR-9999"})
	if got != "G-0001" {
		t.Errorf("got %q, want G-0001 (other-kind trunk ids ignored)", got)
	}
}
