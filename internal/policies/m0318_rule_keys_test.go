package policies

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/workflows/spec"
)

func TestM0318_AC1_RuleKeyDistinguishesEachDimension(t *testing.T) {
	t.Parallel()

	base := spec.Rule{
		Kind: entity.KindEpic, FromState: "active", Verb: "promote",
		ToState: "done", Outcome: spec.OutcomeLegal,
		Preconditions: []spec.Predicate{{Subject: "all-children.status", Op: "∈", Value: "terminal"}},
	}
	cases := []struct {
		name   string
		change func(*spec.Rule)
	}{
		{"kind", func(r *spec.Rule) { r.Kind = entity.KindMilestone }},
		{"origin", func(r *spec.Rule) { r.FromState = "proposed" }},
		{"verb", func(r *spec.Rule) { r.Verb = "cancel" }},
		{"target", func(r *spec.Rule) { r.ToState = "cancelled" }},
		{"outcome", func(r *spec.Rule) { r.Outcome = spec.OutcomeIllegal }},
		{"no-preconditions", func(r *spec.Rule) { r.Preconditions = nil }},
		{"different-preconditions", func(r *spec.Rule) {
			r.Preconditions = []spec.Predicate{{Subject: "all-children.status", Op: "∈", Value: "non-terminal"}}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			other := base
			tc.change(&other)
			if err := checkRuleKeys([]spec.Rule{base, other}); err != nil {
				t.Fatalf("distinct rules rejected: %v", err)
			}
		})
	}
}

func TestM0318_AC1_RuleKeyRejectsDuplicates(t *testing.T) {
	t.Parallel()

	base := spec.Rule{
		Kind: entity.KindEpic, FromState: "active", Verb: "promote",
		ToState: "done", Outcome: spec.OutcomeLegal,
	}
	conditioned := base
	conditioned.Preconditions = []spec.Predicate{{Subject: "all-children.status", Op: "∈", Value: "terminal"}}
	withMetadata := base
	withMetadata.Sources = spec.RuleSource{Audit: []string{"independent citation"}}
	withEmptyPredicates := base
	withEmptyPredicates.Preconditions = []spec.Predicate{}
	cases := []struct {
		name          string
		rules         []spec.Rule
		wantDuplicate bool
	}{
		{"empty", nil, false},
		{"singleton", []spec.Rule{base}, false},
		{"adjacent", []spec.Rule{base, base}, true},
		{"separated-by-distinct-preconditions", []spec.Rule{base, conditioned, base}, true},
		{"different-metadata", []spec.Rule{base, withMetadata}, true},
		{"nil-and-empty-preconditions", []spec.Rule{base, withEmptyPredicates}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := checkRuleKeys(tc.rules)
			if (err != nil) != tc.wantDuplicate {
				t.Fatalf("duplicate rejection: got %v, want duplicate=%t", err, tc.wantDuplicate)
			}
		})
	}
}
