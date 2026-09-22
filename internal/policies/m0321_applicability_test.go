package policies

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/workflows/spec"
)

func TestM0321_AC1_ApplicabilityIsTotal(t *testing.T) {
	t.Parallel()
	var verbs []string
	for _, r := range spec.Rules() {
		verbs = append(verbs, r.Verb)
	}
	slices.Sort(verbs)
	verbs = slices.Compact(verbs)
	if err := checkApplicability(spec.Applicabilities(), workflowKinds(), verbs); err != nil {
		t.Fatal(err)
	}
}

func TestM0321_AC1_OnlyPhaseCancellationIsInapplicable(t *testing.T) {
	t.Parallel()
	for _, a := range spec.Applicabilities() {
		want := a.Kind != spec.KindTDDPhase || a.Verb != "cancel"
		if a.Applies != want {
			t.Errorf("(%s, %s) applies = %v, want %v", a.Kind, a.Verb, a.Applies, want)
		}
	}
}

func TestM0321_AC1_ApplicabilityValidation(t *testing.T) {
	t.Parallel()
	valid := []spec.Applicability{
		{Kind: entity.KindEpic, Verb: "cancel", Applies: true},
		{Kind: spec.KindTDDPhase, Verb: "cancel", Reason: "The phase ladder has no cancellation operation."},
	}
	tests := []struct {
		name    string
		entries []spec.Applicability
		kinds   []entity.Kind
		verbs   []string
		want    string
	}{
		{name: "complete", entries: valid},
		{name: "missing pair", entries: valid[:1], want: "missing applicability for (tdd-phase, cancel)"},
		{name: "new kind", entries: valid, kinds: []entity.Kind{entity.KindEpic, spec.KindTDDPhase, entity.KindGap}, want: "missing applicability for (gap, cancel)"},
		{name: "new verb", entries: valid, verbs: []string{"cancel", "promote"}, want: "missing applicability for (epic, promote)"},
		{name: "unknown kind", entries: []spec.Applicability{{Kind: entity.KindGap, Verb: "cancel", Applies: true}}, want: "unexpected applicability for (gap, cancel)"},
		{name: "unknown verb", entries: []spec.Applicability{{Kind: entity.KindEpic, Verb: "promote", Applies: true}}, want: "unexpected applicability for (epic, promote)"},
		{name: "duplicate", entries: append(slices.Clone(valid), valid[0]), want: "duplicate applicability for (epic, cancel)"},
		{name: "blank reason", entries: []spec.Applicability{{Kind: entity.KindEpic, Verb: "cancel", Reason: " \t"}}, want: "inapplicable pair (epic, cancel) needs a reason"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			kinds := tt.kinds
			if kinds == nil {
				kinds = []entity.Kind{entity.KindEpic, spec.KindTDDPhase}
			}
			verbs := tt.verbs
			if verbs == nil {
				verbs = []string{"cancel"}
			}
			err := checkApplicability(tt.entries, kinds, verbs)
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tt.want {
				t.Errorf("validation = %q, want %q", got, tt.want)
			}
		})
	}
}

func workflowKinds() []entity.Kind {
	return append(entity.AllKinds(), spec.KindAC, spec.KindTDDPhase)
}

type applicabilityKey struct {
	kind entity.Kind
	verb string
}

func checkApplicability(entries []spec.Applicability, kinds []entity.Kind, verbs []string) error {
	seen := map[applicabilityKey]bool{}
	for _, a := range entries {
		key := applicabilityKey{a.Kind, a.Verb}
		if !slices.Contains(kinds, a.Kind) || !slices.Contains(verbs, a.Verb) {
			return fmt.Errorf("unexpected applicability for (%s, %s)", a.Kind, a.Verb)
		}
		if seen[key] {
			return fmt.Errorf("duplicate applicability for (%s, %s)", a.Kind, a.Verb)
		}
		if !a.Applies && strings.TrimSpace(a.Reason) == "" {
			return fmt.Errorf("inapplicable pair (%s, %s) needs a reason", a.Kind, a.Verb)
		}
		seen[key] = true
	}
	for _, kind := range kinds {
		for _, verb := range verbs {
			if !seen[applicabilityKey{kind, verb}] {
				return fmt.Errorf("missing applicability for (%s, %s)", kind, verb)
			}
		}
	}
	return nil
}
