package policies

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/workflows/spec"
)

func TestM0321_AC2_ApplicableCoordinatesHaveRules(t *testing.T) {
	t.Parallel()
	missing, count := missingRuleCoordinates(spec.Rules(), spec.Applicabilities())
	for _, c := range missing {
		t.Errorf("missing rule at (%s, %q, %s)", c.kind, c.state, c.verb)
	}
	t.Logf("checked %d applicable coordinates", count)
}

func TestM0321_AC2_TotalityNamesRemovedCoordinate(t *testing.T) {
	t.Parallel()
	for _, coordinate := range []ruleCoordinate{
		{entity.KindEpic, "done", "cancel"},
		{spec.KindAC, "deferred", "cancel"},
		{spec.KindTDDPhase, "", "promote"},
	} {
		t.Run(string(coordinate.kind), func(t *testing.T) {
			t.Parallel()
			rules := slices.DeleteFunc(spec.Rules(), func(r spec.Rule) bool {
				return ruleCoordinate{r.Kind, r.FromState, r.Verb} == coordinate
			})
			missing, _ := missingRuleCoordinates(rules, spec.Applicabilities())
			if diff := cmp.Diff([]ruleCoordinate{coordinate}, missing, cmp.AllowUnexported(ruleCoordinate{})); diff != "" {
				t.Errorf("missing coordinates (-want +got):\n%s", diff)
			}
		})
	}
}

// A coordinate can have several targets or conditional outcomes; totality
// requires a representative cell, while the outcome drivers judge its behavior.
func TestM0321_AC2_CompanionCellKeepsCoordinateCovered(t *testing.T) {
	t.Parallel()
	rules := spec.Rules()
	for i, r := range rules {
		if r.Kind == entity.KindEpic && r.FromState == "proposed" && r.Verb == "cancel" {
			rules = slices.Delete(rules, i, i+1)
			break
		}
	}
	if missing, _ := missingRuleCoordinates(rules, spec.Applicabilities()); len(missing) != 0 {
		t.Fatalf("companion cell should preserve coverage: %+v", missing)
	}
}

type ruleCoordinate struct {
	kind  entity.Kind
	state string
	verb  string
}

func missingRuleCoordinates(rules []spec.Rule, applicability []spec.Applicability) (missing []ruleCoordinate, count int) {
	covered := map[ruleCoordinate]bool{}
	for i := range rules {
		r := &rules[i]
		covered[ruleCoordinate{r.Kind, r.FromState, r.Verb}] = true
	}
	for _, a := range applicability {
		if !a.Applies {
			continue
		}
		for _, state := range workflowStates(a.Kind) {
			count++
			coordinate := ruleCoordinate{a.Kind, state, a.Verb}
			if !covered[coordinate] {
				missing = append(missing, coordinate)
			}
		}
	}
	return missing, count
}

func workflowStates(kind entity.Kind) []string {
	if kind == spec.KindTDDPhase {
		return append([]string{""}, entity.AllowedTDDPhases()...)
	}
	states := entity.AllowedStatuses(kind)
	if kind == spec.KindAC {
		states = entity.AllowedACStatuses()
	}
	var out []string
	for _, state := range states {
		out = append(out, string(state))
	}
	return out
}
