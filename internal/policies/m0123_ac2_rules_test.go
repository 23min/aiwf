package policies

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/workflows/spec"
)

// TestM0123_AC2_RulesNonEmpty asserts the spec table is populated.
// Empty Rules() would be either an unfinished spec or a regression.
func TestM0123_AC2_RulesNonEmpty(t *testing.T) {
	t.Parallel()

	rules := spec.Rules()
	if len(rules) == 0 {
		t.Fatal("spec.Rules() returned empty slice; expected populated cell table")
	}
}

// TestM0123_AC2_OutcomeNotUnspecified asserts every cell carries a defined
// Outcome (the OutcomeUnspecified zero-value sentinel is a "forgot to set"
// footgun the schema invariant catches).
func TestM0123_AC2_OutcomeNotUnspecified(t *testing.T) {
	t.Parallel()

	for i, r := range spec.Rules() {
		if r.Outcome == spec.OutcomeUnspecified {
			t.Errorf("Rules()[%d] (Kind=%q, FromState=%q, Verb=%q): Outcome is OutcomeUnspecified", i, r.Kind, r.FromState, r.Verb)
		}
	}
}

// TestM0123_AC2_IllegalImpliesRejectionLayer asserts every illegal cell
// names where the rejection happens (verb-time vs check-time). Without
// this, drift policy can't pair illegal cells with their finding codes.
func TestM0123_AC2_IllegalImpliesRejectionLayer(t *testing.T) {
	t.Parallel()

	for i, r := range spec.Rules() {
		if r.Outcome == spec.OutcomeIllegal && r.RejectionLayer == spec.RejectionLayerNone {
			t.Errorf("Rules()[%d] (Kind=%q, FromState=%q, Verb=%q): Outcome=Illegal but RejectionLayer=None", i, r.Kind, r.FromState, r.Verb)
		}
	}
}

// TestM0123_AC2_VerbTimeImpliesBlockingStrict asserts that verb-time
// rejections are always blocking (the verb returns non-zero, no commit;
// there's no "advisory verb refusal" path).
func TestM0123_AC2_VerbTimeImpliesBlockingStrict(t *testing.T) {
	t.Parallel()

	for i, r := range spec.Rules() {
		if r.RejectionLayer == spec.RejectionLayerVerbTime && !r.BlockingStrict {
			t.Errorf("Rules()[%d] (Kind=%q, FromState=%q, Verb=%q): RejectionLayer=VerbTime but BlockingStrict=false", i, r.Kind, r.FromState, r.Verb)
		}
	}
}

// TestM0123_AC2_LegalImpliesNoErrorCode asserts legal cells don't carry an
// ExpectedErrorCode (which only applies to rejections).
func TestM0123_AC2_LegalImpliesNoErrorCode(t *testing.T) {
	t.Parallel()

	for i, r := range spec.Rules() {
		if r.Outcome == spec.OutcomeLegal && r.ExpectedErrorCode != "" {
			t.Errorf("Rules()[%d] (Kind=%q, FromState=%q, Verb=%q): Outcome=Legal but ExpectedErrorCode=%q", i, r.Kind, r.FromState, r.Verb, r.ExpectedErrorCode)
		}
	}
}

// TestM0123_AC2_IllegalImpliesErrorCode asserts illegal cells carry a
// non-empty ExpectedErrorCode (the closure between spec and impl needs the
// code to pair against).
func TestM0123_AC2_IllegalImpliesErrorCode(t *testing.T) {
	t.Parallel()

	for i, r := range spec.Rules() {
		if r.Outcome == spec.OutcomeIllegal && r.ExpectedErrorCode == "" {
			t.Errorf("Rules()[%d] (Kind=%q, FromState=%q, Verb=%q): Outcome=Illegal but ExpectedErrorCode is empty", i, r.Kind, r.FromState, r.Verb)
		}
	}
}

// TestM0123_AC2_KeyUnique asserts (Kind, FromState, Verb, Outcome) tuples
// are unique. Two cells with the same key+outcome would be a duplicate-row
// bug; complementary cells with different Outcomes for the same key are
// allowed (the preconditioned-legal-and-illegal pair pattern).
func TestM0123_AC2_KeyUnique(t *testing.T) {
	t.Parallel()

	type key struct {
		Kind      entity.Kind
		FromState string
		Verb      string
		Outcome   spec.Outcome
	}
	seen := map[key]int{}
	for i, r := range spec.Rules() {
		k := key{r.Kind, r.FromState, r.Verb, r.Outcome}
		if prev, ok := seen[k]; ok {
			// Two rules with the same key + outcome — only allowed if their
			// preconditions distinguish them (in which case they're not
			// "duplicates" in the operational sense). Assert at least the
			// preconditions slice differs.
			prevRule := spec.Rules()[prev]
			if predicateSliceEqual(prevRule.Preconditions, r.Preconditions) {
				t.Errorf("Rules()[%d] and Rules()[%d] are duplicates: (Kind=%q, FromState=%q, Verb=%q, Outcome=%d) with identical Preconditions",
					prev, i, r.Kind, r.FromState, r.Verb, r.Outcome)
			}
		}
		seen[k] = i
	}
}

// TestM0123_AC2_EveryEntityFSMFromStateCovered asserts every (Kind, FromState)
// in the kernel's entity.transitions table is referenced by at least one
// Rule in the spec. This is the impl→spec arm of the drift policy at the
// FSM-state level (AC-5 will land the full bidirectional policy with
// verb-set and finding-code coverage too).
func TestM0123_AC2_EveryEntityFSMFromStateCovered(t *testing.T) {
	t.Parallel()

	// Enumerate every (Kind, FromState) the kernel FSM knows about.
	kinds := []entity.Kind{
		entity.KindEpic,
		entity.KindMilestone,
		entity.KindADR,
		entity.KindGap,
		entity.KindDecision,
		entity.KindContract,
	}

	// Build coverage map from the spec.
	covered := map[entity.Kind]map[string]bool{}
	for _, r := range spec.Rules() {
		if _, ok := covered[r.Kind]; !ok {
			covered[r.Kind] = map[string]bool{}
		}
		covered[r.Kind][r.FromState] = true
	}

	// For each known kind, walk entity.AllowedTransitions to find every
	// FromState the FSM recognizes.
	for _, k := range kinds {
		// We need to enumerate states. Build by probing all known status
		// strings from the entity package's transition functions. The
		// authoritative list is in entity.transitions but it's unexported;
		// the closest public surface is entity.IsTerminal + a known-states
		// catalog. For this AC, the per-kind canonical states are:
		fromStates := canonicalFromStates(k)
		for _, fs := range fromStates {
			if !covered[k][fs] {
				t.Errorf("spec.Rules() missing coverage for (Kind=%q, FromState=%q): no cell references this FSM position", k, fs)
			}
		}
	}
}

// canonicalFromStates returns the closed-set list of from-states recognized
// by entity.transitions for the given kind. This mirrors the FSM table; if
// the kernel's FSM gains a new state, this list — and the spec — must grow
// together (the drift policy in AC-5 will be the chokepoint).
func canonicalFromStates(k entity.Kind) []string {
	switch k {
	case entity.KindEpic:
		return []string{"proposed", "active", "done", "cancelled"}
	case entity.KindMilestone:
		return []string{"draft", "in_progress", "done", "cancelled"}
	case entity.KindADR:
		return []string{"proposed", "accepted", "superseded", "rejected"}
	case entity.KindGap:
		return []string{"open", "addressed", "wontfix"}
	case entity.KindDecision:
		return []string{"proposed", "accepted", "superseded", "rejected"}
	case entity.KindContract:
		return []string{"proposed", "accepted", "deprecated", "retired", "rejected"}
	}
	return nil
}

// TestM0123_AC2_DecisionSourcesPopulatedForFPOnlyAndConflict asserts that
// every cell carrying a non-empty Sources.Decision is one we expect (D-0002
// through D-0007). Cells in Agreement / Audit-only class have empty
// Decision. AC-6 will assert the D-NNNN ids resolve to real entities in
// the planning tree.
func TestM0123_AC2_DecisionSourcesPopulatedForFPOnlyAndConflict(t *testing.T) {
	t.Parallel()

	expectedDecisions := map[string]bool{
		"D-0002": true, // Q4 Conflict
		"D-0003": true, // Q5 FP-only
		"D-0004": true, // Q6 FP-only
		"D-0005": true, // Q7 FP-only
		"D-0006": true, // Q14 FP-only (predicate not cell; may or may not show in Rules)
		"D-0007": true, // Q15 FP-only
	}

	for i, r := range spec.Rules() {
		if r.Sources.Decision == "" {
			continue
		}
		if !expectedDecisions[r.Sources.Decision] {
			t.Errorf("Rules()[%d] (Kind=%q, FromState=%q, Verb=%q): Sources.Decision=%q is not in the expected M-0123 set",
				i, r.Kind, r.FromState, r.Verb, r.Sources.Decision)
		}
	}
}

// predicateSliceEqual compares two []Predicate slices by value.
// Returns true iff lengths and every element match field-by-field.
func predicateSliceEqual(a, b []spec.Predicate) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Subject != b[i].Subject || a[i].Op != b[i].Op || a[i].Value != b[i].Value {
			return false
		}
	}
	return true
}
