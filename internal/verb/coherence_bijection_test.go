package verb

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// ruleSpecBijectionViolations reports every rule on which the
// declaration and the rules that actually fire disagree, in either
// direction, sorted so the report does not depend on map iteration
// order.
func ruleSpecBijectionViolations(specs []coherenceRuleSpec) []string {
	fired := map[string]bool{}
	for _, c := range coherenceDomainFrom(flagsFor(specs)) {
		if v := verdict(c.trailers); v != "coherent" {
			fired[v] = true
		}
	}

	declared := map[string]bool{}
	for _, rule := range declaredCoherenceRules(specs) {
		declared[rule] = true
	}

	var out []string
	for rule := range declared {
		if !fired[rule] {
			// A rule goes unreachable without being deleted: the checks
			// run in order and return the first violation, so a
			// broadened earlier condition can shadow a later rule
			// entirely. It then reads as enforced, still passes its own
			// direct-call unit test, and fires for no real trailer set.
			// A rule reading a trailer no domain point carries lands
			// here too.
			out = append(out, fmt.Sprintf(
				"%s: declared, but fires at no point in the domain — shadowed by an earlier rule, or reading a trailer the domain never varies", rule))
		}
	}
	for rule := range fired {
		if !declared[rule] {
			out = append(out, fmt.Sprintf(
				"%s: fires in the domain but is absent from the declaration, so every list derived from it omits the rule", rule))
		}
	}
	sort.Strings(out)
	return out
}

// TestCoherenceRuleSpecs_BijectWithFiringRules is M-0294/AC-3.
//
// The declaration and the rules that actually fire must name the same
// set. Each direction fails differently and both matter: a declared
// rule that fires nowhere is a rule nothing exercises — the shape that
// let the coherence guard reach one call site of four — and a rule
// firing under a name the declaration omits is a rule outside every
// list derived from it, including the subset verb.Apply enforces.
func TestCoherenceRuleSpecs_BijectWithFiringRules(t *testing.T) {
	t.Parallel()
	if got := ruleSpecBijectionViolations(coherenceRuleSpecs); len(got) != 0 {
		t.Errorf("the declaration and the rules that fire are not in bijection (%d):\n  %s",
			len(got), strings.Join(got, "\n  "))
	}
}

// TestRuleSpecBijectionViolations_NamesTheOffendingRule is AC-3's
// falsifiability half.
//
// Naming the rule is the requirement, not an embellishment: a report
// that a mismatch exists sends the reader back to the search the check
// just performed.
func TestRuleSpecBijectionViolations_NamesTheOffendingRule(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		specs    []coherenceRuleSpec
		wantRule string
	}{
		{
			// An entry dropped for a rule whose trailer another rule
			// still reads: the axis is unchanged and the golden never
			// moves, so nothing else in the suite catches it.
			name:     "a rule fires under a name the declaration omits",
			specs:    withoutSpec(coherenceRuleSpecs, CoherenceRuleAuditOnlyNonHuman),
			wantRule: CoherenceRuleAuditOnlyNonHuman,
		},
		{
			// The other direction: an entry whose rule no invocation can
			// reach. It reads as enforced and never runs.
			name: "a declared rule fires nowhere",
			specs: append(append([]coherenceRuleSpec(nil), coherenceRuleSpecs...),
				coherenceRuleSpec{Rule: "unreachable-by-construction", Reads: []string{gitops.TrailerForce}}),
			wantRule: "unreachable-by-construction",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ruleSpecBijectionViolations(tc.specs)
			if len(got) == 0 {
				t.Fatalf("accepted a declaration that does not biject with the rules that fire")
			}
			joined := strings.Join(got, "\n  ")
			if !strings.Contains(joined, tc.wantRule) {
				t.Errorf("violation does not name %q:\n  %s", tc.wantRule, joined)
			}
		})
	}
}

// withoutSpec returns a copy of in with the named rule's entry removed.
func withoutSpec(in []coherenceRuleSpec, rule string) []coherenceRuleSpec {
	out := make([]coherenceRuleSpec, 0, len(in))
	for _, s := range in {
		if s.Rule != rule {
			out = append(out, s)
		}
	}
	return out
}
