package policies

import (
	"fmt"
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec"
)

// TestM0124_AC4_LegalCellsAllCovered asserts that every Legal cell is
// enumerated by the driver. Coverage identity includes the declared target
// and precondition signature so split cells cannot hide one another.
func TestM0124_AC4_LegalCellsAllCovered(t *testing.T) {
	t.Parallel()

	enumerated := enumeratedCellKeys(t)
	for _, rule := range spec.Rules() {
		if rule.Outcome != spec.OutcomeLegal {
			continue
		}
		key := cellKey(rule)
		if _, ok := enumerated[key]; !ok {
			t.Errorf("Legal cell missing from driver enumeration: %s\n  rule: kind=%s from=%s verb=%s preconditions=%+v",
				key, rule.Kind, rule.FromState, rule.Verb, rule.Preconditions)
		}
	}
}

// TestM0124_AC4_NoExtraEnumerations is the converse of the above:
// every enumerated case corresponds to a Legal cell in spec.Rules().
// Catches the case where enumerateLegalCases accidentally includes
// non-Legal cells (e.g. an Outcome filter change) or fabricates
// cases not grounded in the spec.
func TestM0124_AC4_NoExtraEnumerations(t *testing.T) {
	t.Parallel()

	legalKeys := map[string]bool{}
	for _, rule := range spec.Rules() {
		if rule.Outcome == spec.OutcomeLegal {
			legalKeys[cellKey(rule)] = true
		}
	}
	for _, c := range enumerateLegalCases(t) {
		key := cellKey(c.rule)
		if !legalKeys[key] {
			t.Errorf("enumeration includes case not grounded in a Legal spec cell: %s (case name %q)", key, c.name)
		}
	}
}

// TestM0124_AC4_SubtestNamesUnique asserts case names are unique.
// t.Run on a duplicate name silently shadows the second subtest
// under most test runners (the result is reported but the
// disambiguation is lost). The case-name function appends a
// precondition signature when (Kind, FromState, Verb, target)
// collides; this test confirms the signature is sufficient.
func TestM0124_AC4_SubtestNamesUnique(t *testing.T) {
	t.Parallel()

	seen := map[string]int{}
	cases := enumerateLegalCases(t)
	for _, c := range cases {
		seen[c.name]++
	}
	for name, count := range seen {
		if count > 1 {
			t.Errorf("subtest name %q assigned to %d distinct cases — case-name disambiguation insufficient", name, count)
		}
	}
}

// TestM0124_AC4_EveryCaseHasTargets asserts each driver case executes the
// cell's non-empty declared target.
func TestM0124_AC4_EveryCaseHasTargets(t *testing.T) {
	t.Parallel()

	for _, c := range enumerateLegalCases(t) {
		if c.target == "" || c.target != c.rule.ToState {
			t.Errorf("case %q target=%q, want the cell's declared target %q", c.name, c.target, c.rule.ToState)
		}
	}
}

func TestM0124_AC4_LegalCellKeysUnique(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}
	for _, rule := range spec.Rules() {
		if rule.Outcome != spec.OutcomeLegal {
			continue
		}
		key := cellKey(rule)
		if seen[key] {
			t.Errorf("distinct legal cells share coverage key %q", key)
		}
		seen[key] = true
	}
}

// enumeratedCellKeys collects the set of cell keys from the driver's
// enumerateLegalCases. Used by the coverage assertion.
func enumeratedCellKeys(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	cases := enumerateLegalCases(t)
	for i := range cases {
		out[cellKey(cases[i].rule)] = true
	}
	return out
}

// cellKey is the (Kind, FromState, Verb, ToState, preconditions) identity of
// a spec cell — the smallest tuple that distinguishes overlapping
// Legal cells (e.g. AC.met's split on parent.tdd). Uses
// preconditionSignature from the driver test file for the
// preconditions component so the key shape stays in step with
// AC-3's case naming.
func cellKey(rule spec.Rule) string {
	from := rule.FromState
	if from == "" {
		from = "empty"
	}
	key := fmt.Sprintf("%s/%s/%s/%s", rule.Kind, from, rule.Verb, rule.ToState)
	if sig := preconditionSignature(rule); sig != "" {
		key += "[" + sig + "]"
	}
	return key
}
