package spec

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// scopeReachTree builds an in-memory tree exercising D-0006's three
// included edges (parent-forward, composite rollup, discovered_in
// reverse), one excluded governance edge (depends_on), and a
// cross-epic no-edge case. ByID and ReachesScope iterate Entities, so
// a hand-built tree (no Load) is sufficient.
func scopeReachTree() *tree.Tree {
	return &tree.Tree{
		Root: "/test",
		Entities: []*entity.Entity{
			{ID: "E-0001", Kind: entity.KindEpic, Status: "active"},
			{
				ID: "M-0001", Kind: entity.KindMilestone, Status: "in_progress", Parent: "E-0001",
				ACs: []entity.AcceptanceCriterion{{ID: "AC-1", Status: "open"}},
			},
			{ID: "M-0002", Kind: entity.KindMilestone, Status: "draft", Parent: "E-0001", DependsOn: []string{"M-0001"}},
			{ID: "G-0001", Kind: entity.KindGap, Status: "open", DiscoveredIn: "M-0001"},
			{ID: "E-0002", Kind: entity.KindEpic, Status: "active"},
			{ID: "M-0003", Kind: entity.KindMilestone, Status: "draft", Parent: "E-0002"},
		},
	}
}

// scopeReachCases enumerates (target, scope-entity) pairs spanning
// reachable and unreachable per D-0006 — included edges and one of
// each excluded shape.
var scopeReachCases = []struct {
	name   string
	target string
	scope  string
	want   bool
}{
	{"self", "E-0001", "E-0001", true},
	{"parent forward: milestone to epic", "M-0001", "E-0001", true},
	{"composite rollup + parent: AC to epic", "M-0001/AC-1", "E-0001", true},
	{"discovered_in reverse: gap to milestone", "G-0001", "M-0001", true},
	{"depends_on excluded", "M-0002", "M-0001", false},
	{"cross-epic no edge", "M-0003", "E-0001", false},
}

// TestEvaluatePredicate_ScopeReach is M-0145/AC-1 + AC-2 + AC-3: the
// scope-reach arm evaluates without an unknown-subject error for both
// reachable and unreachable inputs (AC-1); its verdict AGREES with
// tree.ReachesScope for every case, proving it delegates rather than
// re-deriving D-0006 (AC-2); and it reads the Target + ScopeEntity
// verb-invocation context off EvalContext (AC-3).
func TestEvaluatePredicate_ScopeReach(t *testing.T) {
	t.Parallel()
	tr := scopeReachTree()

	for _, tc := range scopeReachCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{Target: tc.target, ScopeEntity: tc.scope}
			// The arm reads ctx.Target; the entity arg is unused for
			// scope-reach (it's verb-invocation context, not entity-side),
			// so resolving it is best-effort — nil for the AC composite.
			e := tr.ByID(tc.target)

			got, err := EvaluatePredicate(Predicate{Subject: "scope-reach", Op: "==", Value: "true"}, e, tr, ctx)
			if err != nil {
				t.Fatalf("AC-1: scope-reach returned error (want none) for %s→%s: %v", tc.target, tc.scope, err)
			}
			if got != tc.want {
				t.Errorf("AC-1: scope-reach(%s→%s) = %v, want %v", tc.target, tc.scope, got, tc.want)
			}
			if reach := tr.ReachesScope(tc.target, tc.scope); got != reach {
				t.Errorf("AC-2: scope-reach verdict %v disagrees with tree.ReachesScope %v for %s→%s", got, reach, tc.target, tc.scope)
			}
		})
	}
}

// TestEvaluatePredicate_ScopeReach_OpContract is M-0145/AC-3: the
// scope-reach arm's cmpBool contract — ==/!= against "true"/"false",
// with a typed error on an unknown op or a non-bool Value. This walks
// every branch of the new comparison path. Fixture target M-0001
// reaches scope E-0001 (reachability == true).
func TestEvaluatePredicate_ScopeReach_OpContract(t *testing.T) {
	t.Parallel()
	tr := scopeReachTree()
	ctx := EvalContext{Target: "M-0001", ScopeEntity: "E-0001"} // reachable: true

	cases := []struct {
		name    string
		op      string
		value   string
		want    bool
		wantErr string
	}{
		{"== true → reachable", "==", "true", true, ""},
		{"== false → reachable is not false", "==", "false", false, ""},
		{"!= true → reachable is not unequal-to-true", "!=", "true", false, ""},
		{"!= false → reachable is unequal-to-false", "!=", "false", true, ""},
		{"unknown op rejected", "non-empty", "true", false, "unknown op"},
		{"non-bool value rejected", "==", "yes", false, "true/false"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := EvaluatePredicate(Predicate{Subject: "scope-reach", Op: tc.op, Value: tc.value}, nil, tr, ctx)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("scope-reach %s %q = %v, want %v", tc.op, tc.value, got, tc.want)
			}
		})
	}
}
