package policies

import (
	"fmt"
	"slices"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/workflows/spec"
)

type workflowTransition struct {
	kind entity.Kind
	from string
	to   string
}

func workflowTransitions() []workflowTransition {
	var edges []workflowTransition
	for _, kind := range entity.AllKinds() {
		for _, from := range entity.AllowedStatuses(kind) {
			for _, to := range entity.AllowedTransitions(kind, from) {
				edges = append(edges, workflowTransition{kind, string(from), string(to)})
			}
		}
	}
	for _, from := range entity.AllowedACStatuses() {
		for _, to := range entity.AllowedACStatuses() {
			if entity.IsLegalACTransition(from, to) {
				edges = append(edges, workflowTransition{spec.KindAC, string(from), string(to)})
			}
		}
	}
	for _, from := range append([]string{""}, entity.AllowedTDDPhases()...) {
		for _, to := range entity.AllowedTDDPhases() {
			if entity.IsLegalTDDPhaseTransition(from, to) {
				edges = append(edges, workflowTransition{spec.KindTDDPhase, from, to})
			}
		}
	}
	return edges
}

func checkLegalRuleTargets(rules []spec.Rule, edges []workflowTransition) error {
	covered := map[workflowTransition]bool{}
	for i := range rules {
		rule := &rules[i]
		if rule.Outcome != spec.OutcomeLegal {
			continue
		}
		edge := workflowTransition{rule.Kind, rule.FromState, rule.ToState}
		if !slices.Contains(edges, edge) {
			return fmt.Errorf("legal rule %d (%s, %q, %s, %q) names no allowed FSM edge",
				i, rule.Kind, rule.FromState, rule.Verb, rule.ToState)
		}
		covered[edge] = true
	}
	for _, edge := range edges {
		if !covered[edge] {
			return fmt.Errorf("FSM edge (%s, %q, %q) has no legal rule", edge.kind, edge.from, edge.to)
		}
	}
	return nil
}

func TestM0318_AC2_LegalTargetsAgreeWithFSM(t *testing.T) {
	t.Parallel()

	if err := checkLegalRuleTargets(spec.Rules(), workflowTransitions()); err != nil {
		t.Fatal(err)
	}
}

func TestM0318_AC2_TargetDriftIsRejectedInBothDirections(t *testing.T) {
	t.Parallel()

	edge := workflowTransition{kind: entity.KindEpic, from: "active", to: "done"}
	otherEdge := workflowTransition{kind: entity.KindEpic, from: "active", to: "cancelled"}
	rule := spec.Rule{
		Kind: edge.kind, FromState: edge.from, Verb: "promote",
		ToState: edge.to, Outcome: spec.OutcomeLegal,
	}
	illegal := rule
	illegal.Outcome = spec.OutcomeIllegal
	missingTarget := rule
	missingTarget.ToState = ""
	wrongTarget := rule
	wrongTarget.ToState = "cancelled"
	conditioned := rule
	conditioned.Preconditions = []spec.Predicate{{Subject: "all-children.status", Op: "∈", Value: "terminal"}}
	cases := []struct {
		name      string
		rules     []spec.Rule
		edges     []workflowTransition
		wantDrift bool
	}{
		{"empty", nil, nil, false},
		{"matching-edge", []spec.Rule{rule}, []workflowTransition{edge}, false},
		{"complementary-cells", []spec.Rule{rule, illegal, conditioned}, []workflowTransition{edge}, false},
		{"missing-target", []spec.Rule{missingTarget}, []workflowTransition{edge}, true},
		{"target-not-in-fsm", []spec.Rule{wrongTarget}, []workflowTransition{edge}, true},
		{"extra-cell-beyond-fsm", []spec.Rule{rule, wrongTarget}, []workflowTransition{edge}, true},
		{"edge-without-cell", nil, []workflowTransition{edge}, true},
		{"illegal-cell-does-not-cover-edge", []spec.Rule{illegal}, []workflowTransition{edge}, true},
		{"added-fsm-edge", []spec.Rule{rule}, []workflowTransition{edge, otherEdge}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := checkLegalRuleTargets(tc.rules, tc.edges)
			if (err != nil) != tc.wantDrift {
				t.Fatalf("target drift: got %v, want drift=%t", err, tc.wantDrift)
			}
		})
	}
}
