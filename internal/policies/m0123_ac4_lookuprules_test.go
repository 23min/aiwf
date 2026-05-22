package policies

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/workflows/spec"
)

// TestM0123_AC4_LookupRulesHitSingle asserts a (Kind, FromState, Verb) key
// with one matching cell returns a slice of length 1.
//
// Fixture: (KindEpic, "proposed", "promote") — the legal proposed → active
// ratification cell (epicRules() entry; R-AUDIT-0001 / R-FP-0001).
func TestM0123_AC4_LookupRulesHitSingle(t *testing.T) {
	t.Parallel()

	got := spec.LookupRules(entity.KindEpic, "proposed", "promote")
	if len(got) != 1 {
		t.Fatalf("LookupRules(KindEpic, proposed, promote) length: want 1, got %d", len(got))
	}
	r := got[0]
	if r.Kind != entity.KindEpic || r.FromState != "proposed" || r.Verb != "promote" {
		t.Errorf("LookupRules returned non-matching cell: Kind=%q FromState=%q Verb=%q", r.Kind, r.FromState, r.Verb)
	}
	if r.Outcome != spec.OutcomeLegal {
		t.Errorf("Expected OutcomeLegal for proposed → active ratification, got Outcome=%d", r.Outcome)
	}
}

// TestM0123_AC4_LookupRulesHitPreconditionedPair asserts that a key with a
// legal cell AND a preconditioned illegal companion returns both. This is
// the load-bearing semantics distinguishing LookupRules (plural, slice)
// from a single-value lookup: the (Kind, FromState, Verb, Outcome) tuple
// is the uniqueness key, not (Kind, FromState, Verb).
//
// Fixture: (KindEpic, "proposed", "cancel") — Q5 / D-0003 pair:
//   - legal cell (no preconditions): generic proposed → cancelled
//   - illegal cell (precondition: any-child.status non-terminal):
//     epic-cancel-non-terminal-children
func TestM0123_AC4_LookupRulesHitPreconditionedPair(t *testing.T) {
	t.Parallel()

	got := spec.LookupRules(entity.KindEpic, "proposed", "cancel")
	if len(got) != 2 {
		t.Fatalf("LookupRules(KindEpic, proposed, cancel) length: want 2 (legal + Q5 illegal companion), got %d", len(got))
	}

	var sawLegal, sawIllegal bool
	for _, r := range got {
		if r.Kind != entity.KindEpic || r.FromState != "proposed" || r.Verb != "cancel" {
			t.Errorf("LookupRules returned non-matching cell: Kind=%q FromState=%q Verb=%q", r.Kind, r.FromState, r.Verb)
		}
		switch r.Outcome {
		case spec.OutcomeLegal:
			sawLegal = true
		case spec.OutcomeIllegal:
			sawIllegal = true
			if r.ExpectedErrorCode != "epic-cancel-non-terminal-children" {
				t.Errorf("Illegal companion ExpectedErrorCode: want %q, got %q",
					"epic-cancel-non-terminal-children", r.ExpectedErrorCode)
			}
		}
	}
	if !sawLegal {
		t.Error("LookupRules did not return the legal proposed → cancelled cell")
	}
	if !sawIllegal {
		t.Error("LookupRules did not return the Q5/D-0003 illegal companion cell")
	}
}

// TestM0123_AC4_LookupRulesMiss asserts a key with no matching cells
// returns an empty (zero-length) slice. The contract is that "miss" is
// distinguished from "hit-zero-length-by-accident" via the input space
// (the kernel FSM enumeration; spec.Rules() covers every recognized key).
func TestM0123_AC4_LookupRulesMiss(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		kind      entity.Kind
		fromState string
		verb      string
	}{
		{"unknown-from-state", entity.KindEpic, "no-such-state", "promote"},
		{"unknown-verb", entity.KindEpic, "proposed", "no-such-verb"},
		{"unknown-kind", entity.Kind("no-such-kind"), "proposed", "promote"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := spec.LookupRules(tc.kind, tc.fromState, tc.verb)
			if len(got) != 0 {
				t.Errorf("LookupRules(%q, %q, %q): want empty slice on miss, got %d cells",
					tc.kind, tc.fromState, tc.verb, len(got))
			}
		})
	}
}

// TestM0123_AC4_LookupRulesNoDuplicatesWithinResult asserts that no two
// cells in a LookupRules result share BOTH the same Outcome AND identical
// Preconditions. This mirrors AC-2's TestM0123_AC2_KeyUnique invariant
// (the real uniqueness key is (Kind, FromState, Verb, Outcome,
// Preconditions); same key + outcome with different preconditions is
// legitimate per the refined-cell pattern, e.g., AC.open.promote has both
// an open → met cell and an open → deferred cell, both legal, distinguished
// by self.target-state).
//
// LookupRules itself never introduces duplicates — it filters Rules() by
// key. The test exists to assert that LookupRules surfaces the table's
// invariant correctly under the AC-4 access path.
func TestM0123_AC4_LookupRulesNoDuplicatesWithinResult(t *testing.T) {
	t.Parallel()

	// Walk every distinct (Kind, FromState, Verb) the table references.
	seenKey := map[struct {
		k  entity.Kind
		fs string
		v  string
	}]bool{}
	for _, r := range spec.Rules() {
		k := struct {
			k  entity.Kind
			fs string
			v  string
		}{r.Kind, r.FromState, r.Verb}
		if seenKey[k] {
			continue
		}
		seenKey[k] = true

		got := spec.LookupRules(r.Kind, r.FromState, r.Verb)
		for i := 0; i < len(got); i++ {
			for j := i + 1; j < len(got); j++ {
				if got[i].Outcome == got[j].Outcome && predicateSliceEqual(got[i].Preconditions, got[j].Preconditions) {
					t.Errorf("LookupRules(%q, %q, %q) returned duplicate cells at indices %d and %d: same Outcome=%d and identical Preconditions",
						r.Kind, r.FromState, r.Verb, i, j, got[i].Outcome)
				}
			}
		}
	}
}

// TestM0123_AC4_LookupRulesMatchesAllInputs asserts the returned slice
// contains ONLY cells matching all three input keys. (Defensive: an
// implementation that filtered on Kind + FromState only would slip through
// the hit-tests above whenever the verb happens to match.)
func TestM0123_AC4_LookupRulesMatchesAllInputs(t *testing.T) {
	t.Parallel()

	for _, r := range spec.Rules() {
		got := spec.LookupRules(r.Kind, r.FromState, r.Verb)
		for _, c := range got {
			if c.Kind != r.Kind || c.FromState != r.FromState || c.Verb != r.Verb {
				t.Errorf("LookupRules(%q, %q, %q) returned cell with Kind=%q FromState=%q Verb=%q",
					r.Kind, r.FromState, r.Verb, c.Kind, c.FromState, c.Verb)
			}
		}
	}
}
