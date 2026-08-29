package policies

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/workflows/spec"
)

// sovereignActShapeSubject is the predicate subject naming the kernel's
// sovereign closed set inside the legal-workflow spec.
const sovereignActShapeSubject = "sovereign-act-shape"

// TestM0324_AC6_SpecModelsTheSovereigntyGate asserts the legal-workflow
// spec carries a cross-cutting rule for the sovereignty gate.
//
// Before M-0324 the spec modelled sovereignty for no edge at all —
// including epic `proposed → active`, shipped since M-0095. All four
// epic cells were bare OutcomeLegal, and the only preconditioned ones
// were the child-cascade pair, so a reader taking the spec as the
// canonical legality surface would not learn that a human is required
// to open or close an epic.
//
// The rule is deliberately symbolic rather than an enumeration of the
// closed set's current entries. A predicate naming `sovereign-act-shape`
// stays correct as the set widens, so there is no second copy of the
// set to drift; an enumeration would buy a drift check by first
// creating the drift it detects.
//
// It lives in GlobalRules rather than as per-cell entries because the
// sovereign refusal carries no finding code, and
// TestM0123_AC2_IllegalImpliesErrorCode requires every illegal cell in
// Rules() to name one. G-0649 tracks that; until it resolves, the
// cross-cutting accessor is the only home the schema admits.
func TestM0324_AC6_SpecModelsTheSovereigntyGate(t *testing.T) {
	t.Parallel()

	_, tr := sharedRepoTree(t)

	if len(entity.SovereignActShapes()) == 0 {
		t.Fatal("the kernel's sovereign closed set is empty, so the spec rule below denotes " +
			"nothing and this test would pass while asserting no live behaviour")
	}

	var matches []spec.Rule
	for _, r := range spec.GlobalRules() {
		for _, p := range r.Preconditions {
			if p.Subject == sovereignActShapeSubject {
				matches = append(matches, r)
				break
			}
		}
	}
	if len(matches) != 1 {
		t.Fatalf("found %d GlobalRules carrying a %q precondition, want exactly 1 — the "+
			"sovereignty gate is one cross-cutting rule, not a per-edge cell",
			len(matches), sovereignActShapeSubject)
	}
	r := matches[0]

	if r.Outcome != spec.OutcomeIllegal {
		t.Errorf("Outcome = %v, want OutcomeIllegal — the rule describes the case the kernel "+
			"refuses, not the case it permits", r.Outcome)
	}
	// The refusal happens in the verb, before anything is written, which
	// is what ADR-0040 requires of a prevention rule. Recording it as
	// check-time would describe the audit that backstops the gate rather
	// than the gate.
	if r.RejectionLayer != spec.RejectionLayerVerbTime {
		t.Errorf("RejectionLayer = %v, want RejectionLayerVerbTime", r.RejectionLayer)
	}
	if !r.BlockingStrict {
		t.Error("BlockingStrict = false; a verb-time rejection is strict by the spec's own " +
			"schema invariant")
	}

	// The authorizing record is resolved rather than merely spelled.
	// TestM0123_AC6_RuleDecisionSourcesResolve does this for Rules()
	// cells but not for GlobalRules, and it could not cover this one
	// unchanged: it requires Sources.Decision to resolve to a decision
	// entity, while the cross-cutting rules cite ADRs. Widening it would
	// redefine what that field admits, which is a larger question than
	// this milestone answers.
	authorizing := tr.ByID(r.Sources.Decision)
	if authorizing == nil {
		t.Errorf("Sources.Decision=%q does not resolve via tr.ByID; the rule cites no record a "+
			"reader can follow", r.Sources.Decision)
	} else if authorizing.Status != entity.StatusAccepted {
		t.Errorf("Sources.Decision=%q resolves at status %q, want %q — a rule the kernel enforces "+
			"cannot rest on an unratified record", r.Sources.Decision, authorizing.Status, entity.StatusAccepted)
	}

	// The other two preconditions are what scope the rule to the case
	// that is actually refused: a non-human actor, unforced. Without
	// them the rule would read as refusing everyone, including the human
	// it exists to require.
	want := map[string]string{"actor-role": "!=", "force": "=="}
	got := map[string]string{}
	for _, p := range r.Preconditions {
		got[p.Subject] = p.Op
	}
	for subject, op := range want {
		if got[subject] != op {
			t.Errorf("precondition %q has op %q, want %q — present preconditions: %+v",
				subject, got[subject], op, r.Preconditions)
		}
	}
}
