package entity

import "testing"

// TestIsSovereignActShape_TrueCases enumerates every entry the kernel
// currently treats as sovereign-act-shape — FSM-legal transitions a
// `human/` actor reaches with no flag and a non-human actor cannot
// reach at all. `--force` is not the non-human route: it relaxes this
// gate rather than the FSM, and the apply seam then refuses the force
// trailer it produces from a non-human actor.
//
// The set holds epic proposed → active, authorized by M-0095
// (motivated by G-0063), and the edges into a terminal epic status
// that ADR-0047 rules sovereign. New entries land alongside the ADR
// or kernel-spec that ratifies them.
func TestIsSovereignActShape_TrueCases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		kind Kind
		from Status
		to   Status
	}{
		{KindEpic, StatusProposed, StatusActive},
		{KindEpic, StatusActive, StatusDone},
		{KindEpic, StatusActive, StatusCancelled},
		{KindEpic, StatusProposed, StatusCancelled},
	}
	for _, c := range cases {
		t.Run(string(c.kind)+"/"+string(c.from)+"->"+string(c.to), func(t *testing.T) {
			t.Parallel()
			if !IsSovereignActShape(c.kind, c.from, c.to) {
				t.Errorf("IsSovereignActShape(%s, %q, %q) = false, want true", c.kind, c.from, c.to)
			}
		})
	}
}

// TestIsSovereignActShape_FalseCases pins what is NOT sovereign-act-
// shape — most legal transitions, all illegal transitions, and any
// tuple with unknown kind or unknown status.
//
// Legal-but-not-sovereign-act-shape cases (the "negative space" the
// predicate must NOT fire on) get explicit coverage because they are
// the failure mode that matters most: a too-broad predicate would
// refuse routine transitions the kernel permits. Every such case is
// now another kind's — the epic FSM's four legal transitions are all
// sovereign, so epics contribute no negative space.
func TestIsSovereignActShape_FalseCases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		kind Kind
		from Status
		to   Status
	}{
		// No legal epic transition appears here. ADR-0047 rules every
		// edge into a terminal epic status sovereign, and with
		// proposed → active already in the set from M-0095, the epic
		// FSM's four legal transitions are now sovereign without
		// exception. The negative space this table guards has moved
		// entirely to other kinds.
		//
		// Other kinds — their activation/acceptance edges are
		// explicitly out of scope per M-0095's spec body ("separate
		// open question, deferred at planning time").
		{"milestone draft->in_progress", KindMilestone, StatusDraft, StatusInProgress},
		{"milestone in_progress->done", KindMilestone, StatusInProgress, StatusDone},
		{"adr proposed->accepted", KindADR, StatusProposed, StatusAccepted},
		{"decision proposed->accepted", KindDecision, StatusProposed, StatusAccepted},
		{"contract proposed->accepted", KindContract, StatusProposed, StatusAccepted},
		{"contract accepted->rejected", KindContract, StatusAccepted, StatusRejected},
		{"gap open->addressed", KindGap, StatusOpen, StatusAddressed},
		// Illegal transitions — set membership is independent of FSM
		// legality, but no illegal transition can be in the set by
		// construction (see TestSovereignActShapes_AllFSMLegal).
		{"epic skip-ahead proposed->done", KindEpic, StatusProposed, StatusDone},
		{"milestone backwards", KindMilestone, StatusInProgress, StatusDraft},
		// Unknown kind / unknown status.
		{"unknown kind", Kind("widget"), StatusProposed, StatusActive},
		{"unknown from", KindEpic, "weird", StatusActive},
		{"unknown to", KindEpic, StatusProposed, "weird"},
		{"empty kind", Kind(""), StatusProposed, StatusActive},
		{"empty from", KindEpic, "", StatusActive},
		{"empty to", KindEpic, StatusProposed, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if IsSovereignActShape(c.kind, c.from, c.to) {
				t.Errorf("IsSovereignActShape(%s, %q, %q) = true, want false", c.kind, c.from, c.to)
			}
		})
	}
}

// TestSovereignActShapes_DefensiveCopy asserts the exported accessor
// returns a slice callers can mutate without altering the kernel's
// closed set. Without the copy, a downstream consumer iterating and
// reassigning entries could silently rewrite kernel state.
func TestSovereignActShapes_DefensiveCopy(t *testing.T) {
	t.Parallel()
	got := SovereignActShapes()
	if len(got) == 0 {
		t.Fatal("SovereignActShapes returned empty slice; expected at least one entry")
	}
	original := got[0]
	got[0] = SovereignActShape{Kind: Kind("tampered"), From: "x", To: "y"}
	// The predicate over the original entry must still match.
	if !IsSovereignActShape(original.Kind, original.From, original.To) {
		t.Error("mutating SovereignActShapes() result altered IsSovereignActShape behavior — accessor is not returning a defensive copy")
	}
}

// TestSovereignActShapes_AllFSMLegal asserts the closed-set invariant
// promised in D-0008: every SovereignActShape entry must be a legal
// FSM transition. Sovereign-act-shape is a property *over* legal
// transitions — never below them. If a future entry sneaks in that is
// not FSM-legal, this test fires and the entry must be either removed
// or the FSM extended to admit it.
func TestSovereignActShapes_AllFSMLegal(t *testing.T) {
	t.Parallel()
	for _, s := range SovereignActShapes() {
		t.Run(string(s.Kind)+"/"+string(s.From)+"->"+string(s.To), func(t *testing.T) {
			t.Parallel()
			if err := ValidateTransition(s.Kind, s.From, s.To); err != nil {
				t.Errorf("SovereignActShape entry (%s, %q, %q) is not FSM-legal: %v", s.Kind, s.From, s.To, err)
			}
		})
	}
}
