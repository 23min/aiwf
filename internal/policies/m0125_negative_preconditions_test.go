package policies

import (
	"fmt"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cellcoverage"
	"github.com/23min/aiwf/internal/workflows/spec"
)

// TestM0125_AC1_FixtureSatisfiesIllegalPreconditions exercises the
// fixture infrastructure (NewCellFixture + BringEntityToState +
// SatisfyPredicate) against every Illegal cell in spec.Rules(). It
// asserts the negative-precondition setup completes cleanly per cell —
// the actual rejection check is AC-2/AC-3's job. This test pins the
// AC-1 deliverable: the infrastructure is READY to drive negative cell
// coverage.
//
// The "self-verification" hook lives inside SatisfyPredicate's silent-
// drift guard: after each mutation the fixture is re-loaded and the
// predicate re-evaluated. If a future Illegal-cell predicate introduces
// an (Subject, Op, Value) combination the helper can't materialize,
// this test fails with a precise pointer to the broken atom.
//
// Coverage commitment: every Illegal cell in spec.Rules() yields one
// subtest; the audit-catalog inventory pins the expected count
// (currently 29 per M-0123 phase 1, computed across 12 terminalIllegal
// invocations + 17 explicit struct literals). Failure of the floor
// assertion below catches "Illegal cells silently disappeared from
// spec" — the drift protection for negative coverage.
func TestM0125_AC1_FixtureSatisfiesIllegalPreconditions(t *testing.T) {
	t.Parallel()

	cases := enumerateIllegalCases(t)
	if len(cases) == 0 {
		t.Fatal("no Illegal cells enumerated from spec.Rules(); expected at least 29")
	}
	if len(cases) < 29 {
		t.Errorf("expected at least 29 Illegal cells, got %d (spec shrank?)", len(cases))
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			satisfyIllegalPreconditions(t, tc)
		})
	}
}

type illegalCase struct {
	name string
	rule spec.Rule
}

func enumerateIllegalCases(t *testing.T) []illegalCase {
	t.Helper()
	var out []illegalCase
	rules := spec.Rules()
	for i := range rules {
		rule := rules[i]
		if rule.Outcome != spec.OutcomeIllegal {
			continue
		}
		out = append(out, illegalCase{name: illegalCaseName(rule), rule: rule})
	}
	return out
}

// illegalCaseName mirrors caseName from m0124_positive_driver_test.go
// but drops the target component — Illegal cells don't reach a target,
// the verb gets rejected. preconditionSignature still disambiguates
// cells sharing (Kind, FromState, Verb).
func illegalCaseName(rule spec.Rule) string {
	name := fmt.Sprintf("%s-%s-%s", rule.Kind, rule.FromState, rule.Verb)
	if sig := preconditionSignature(rule); sig != "" {
		name = name + "-" + sig
	}
	name = strings.ReplaceAll(name, "/", "-")
	return name
}

// satisfyIllegalPreconditions runs the precondition pipeline for one
// Illegal cell. It builds the fixture, brings the subject entity to
// the cell's FromState, and materializes each precondition by routing
// every Predicate through SatisfyPredicate. The silent-drift guard
// inside SatisfyPredicate self-verifies; any failure surfaces as a
// t.Fatalf with the (Subject, Op) pair.
//
// Unlike the M-0124 positive driver, no special-cased branches for
// self.target-state / self.addressed_by / self.superseded_by are
// needed here. Those Subjects only matter when the verb supplies the
// value as an arg or populates the field atomically — AC-1's job is
// just to confirm the fixture sets up the entity correctly. The
// driver tests (AC-2/AC-3) layer the verb-arg shaping on top.
func satisfyIllegalPreconditions(t *testing.T, tc illegalCase) {
	t.Helper()
	f := cellcoverage.NewCellFixture(t)
	opts := deriveBringOpts(tc.rule)
	id := bringEntityForCell(t, f, tc.rule, opts)

	evalCtx := spec.EvalContext{}
	for _, p := range tc.rule.Preconditions {
		f.SatisfyPredicate(t, p, id, &evalCtx)
	}
}
