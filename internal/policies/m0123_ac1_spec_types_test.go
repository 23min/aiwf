package policies

import (
	"reflect"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/workflows/spec"
)

// TestM0123_AC1_SpecRuleStructShape asserts the spec.Rule struct carries the
// nine fields concretized in M-0123 phase 1, in declaration order. This is
// the load-bearing structural assertion for the Rule type — phase 2's cell
// authors and drift policies all depend on this shape.
func TestM0123_AC1_SpecRuleStructShape(t *testing.T) {
	t.Parallel()

	expected := []struct {
		name string
		typ  reflect.Type
	}{
		{"Kind", reflect.TypeOf(entity.Kind(""))},
		{"FromState", reflect.TypeOf("")},
		{"Verb", reflect.TypeOf("")},
		{"Preconditions", reflect.TypeOf([]spec.Predicate(nil))},
		{"Outcome", reflect.TypeOf(spec.Outcome(0))},
		{"ExpectedErrorCode", reflect.TypeOf("")},
		{"RejectionLayer", reflect.TypeOf(spec.RejectionLayer(0))},
		{"BlockingStrict", reflect.TypeOf(false)},
		{"Sources", reflect.TypeOf(spec.RuleSource{})},
	}

	rt := reflect.TypeOf(spec.Rule{})
	if got := rt.NumField(); got != len(expected) {
		t.Fatalf("spec.Rule field count: want %d, got %d", len(expected), got)
	}
	for i, want := range expected {
		f := rt.Field(i)
		if f.Name != want.name {
			t.Errorf("spec.Rule field %d: want name %q, got %q", i, want.name, f.Name)
		}
		if f.Type != want.typ {
			t.Errorf("spec.Rule field %s: want type %v, got %v", want.name, want.typ, f.Type)
		}
	}
}

// TestM0123_AC1_OutcomeEnum asserts the Outcome enum has Unspecified zero
// sentinel + Legal + Illegal, all distinct. The zero-value sentinel discipline
// avoids the "forgot to set the field, default went to Legal" footgun.
func TestM0123_AC1_OutcomeEnum(t *testing.T) {
	t.Parallel()

	if spec.OutcomeUnspecified != 0 {
		t.Errorf("OutcomeUnspecified: want zero value 0, got %d", spec.OutcomeUnspecified)
	}
	if spec.OutcomeLegal == spec.OutcomeUnspecified {
		t.Error("OutcomeLegal must differ from OutcomeUnspecified")
	}
	if spec.OutcomeIllegal == spec.OutcomeLegal || spec.OutcomeIllegal == spec.OutcomeUnspecified {
		t.Error("OutcomeIllegal must differ from OutcomeLegal and OutcomeUnspecified")
	}
}

// TestM0123_AC1_RejectionLayerEnum asserts the RejectionLayer enum has None
// zero sentinel (meaningful for legal cells) + VerbTime + CheckTime, all distinct.
func TestM0123_AC1_RejectionLayerEnum(t *testing.T) {
	t.Parallel()

	if spec.RejectionLayerNone != 0 {
		t.Errorf("RejectionLayerNone: want zero value 0, got %d", spec.RejectionLayerNone)
	}
	if spec.RejectionLayerVerbTime == spec.RejectionLayerNone {
		t.Error("RejectionLayerVerbTime must differ from RejectionLayerNone")
	}
	if spec.RejectionLayerCheckTime == spec.RejectionLayerVerbTime || spec.RejectionLayerCheckTime == spec.RejectionLayerNone {
		t.Error("RejectionLayerCheckTime must differ from VerbTime and None")
	}
}

// TestM0123_AC1_PredicateStructShape asserts spec.Predicate carries Subject,
// Op, Value fields (in that order). Closed-set vocabulary validation lives in
// AC-2's cell-level tests, not here.
func TestM0123_AC1_PredicateStructShape(t *testing.T) {
	t.Parallel()

	expected := []string{"Subject", "Op", "Value"}
	rt := reflect.TypeOf(spec.Predicate{})
	if got := rt.NumField(); got != len(expected) {
		t.Fatalf("spec.Predicate field count: want %d, got %d", len(expected), got)
	}
	for i, name := range expected {
		if f := rt.Field(i); f.Name != name {
			t.Errorf("spec.Predicate field %d: want %q, got %q", i, name, f.Name)
		}
	}
}

// TestM0123_AC1_RuleSourceStructShape asserts spec.RuleSource carries Audit,
// FP, Decision fields with the expected types. The reconciliation-class
// invariants (Agreement / Audit-only / FP-only / Conflict population
// signatures) are AC-6's concern, not here.
func TestM0123_AC1_RuleSourceStructShape(t *testing.T) {
	t.Parallel()

	expected := []struct {
		name string
		typ  reflect.Type
	}{
		{"Audit", reflect.TypeOf([]string(nil))},
		{"FP", reflect.TypeOf([]string(nil))},
		{"Decision", reflect.TypeOf("")},
	}
	rt := reflect.TypeOf(spec.RuleSource{})
	if got := rt.NumField(); got != len(expected) {
		t.Fatalf("spec.RuleSource field count: want %d, got %d", len(expected), got)
	}
	for i, want := range expected {
		f := rt.Field(i)
		if f.Name != want.name {
			t.Errorf("spec.RuleSource field %d: want name %q, got %q", i, want.name, f.Name)
		}
		if f.Type != want.typ {
			t.Errorf("spec.RuleSource field %s: want type %v, got %v", want.name, want.typ, f.Type)
		}
	}
}

// TestM0123_AC1_AntiRuleStructShape asserts spec.AntiRule carries ID,
// Statement, Reasoning, Sources fields with the expected types.
func TestM0123_AC1_AntiRuleStructShape(t *testing.T) {
	t.Parallel()

	expected := []struct {
		name string
		typ  reflect.Type
	}{
		{"ID", reflect.TypeOf("")},
		{"Statement", reflect.TypeOf("")},
		{"Reasoning", reflect.TypeOf("")},
		{"Sources", reflect.TypeOf(spec.RuleSource{})},
	}
	rt := reflect.TypeOf(spec.AntiRule{})
	if got := rt.NumField(); got != len(expected) {
		t.Fatalf("spec.AntiRule field count: want %d, got %d", len(expected), got)
	}
	for i, want := range expected {
		f := rt.Field(i)
		if f.Name != want.name {
			t.Errorf("spec.AntiRule field %d: want name %q, got %q", i, want.name, f.Name)
		}
		if f.Type != want.typ {
			t.Errorf("spec.AntiRule field %s: want type %v, got %v", want.name, want.typ, f.Type)
		}
	}
}

// TestM0123_AC1_KindExtensions asserts the spec package declares the KindAC
// and KindTDDPhase constants (entity.Kind-typed extensions for sub-FSM cells
// — composite-id ACs and TDD-phase transitions don't have first-class kinds).
func TestM0123_AC1_KindExtensions(t *testing.T) {
	t.Parallel()

	if string(spec.KindAC) != "ac" {
		t.Errorf("spec.KindAC: want value %q, got %q", "ac", string(spec.KindAC))
	}
	if string(spec.KindTDDPhase) != "tdd-phase" {
		t.Errorf("spec.KindTDDPhase: want value %q, got %q", "tdd-phase", string(spec.KindTDDPhase))
	}
	if reflect.TypeOf(spec.KindAC) != reflect.TypeOf(entity.Kind("")) {
		t.Errorf("spec.KindAC: want type entity.Kind, got %v", reflect.TypeOf(spec.KindAC))
	}
	if reflect.TypeOf(spec.KindTDDPhase) != reflect.TypeOf(entity.Kind("")) {
		t.Errorf("spec.KindTDDPhase: want type entity.Kind, got %v", reflect.TypeOf(spec.KindTDDPhase))
	}
}
