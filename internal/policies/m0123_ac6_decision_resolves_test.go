package policies

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/workflows/spec"
)

// TestM0123_AC6_RuleDecisionSourcesResolve asserts every spec.Rules() cell
// with non-empty Sources.Decision names a D-NNNN that resolves to an
// existing decision entity in the planning tree via tree.Load + Tree.ByID.
//
// This is the load-bearing closure for the FP-only and Conflict
// reconciliation classes (per M-0123 body §"Rule sources"): the spec
// table cites D-NNNN entities as the resolution path for cells where the
// Audit-derived and FP-derived sources diverge OR where FP surfaced a
// pattern not in the Audit. AC-2's invariant
// (TestM0123_AC2_DecisionSourcesPopulatedForFPOnlyAndConflict) restricts
// the cited set to {D-0002..D-0007}; AC-6 closes the loop by asserting
// each cited D-NNNN actually exists.
//
// Per CLAUDE.md §"Policy tests that read entity files must resolve via
// the loader": Tree.ByID transparently resolves active and archive paths
// per ADR-0004, so the test stays correct when a decision entity later
// reaches terminal status and is archive-swept.
func TestM0123_AC6_RuleDecisionSourcesResolve(t *testing.T) {
	t.Parallel()

	_, tr := sharedRepoTree(t)

	rules := spec.Rules()
	for i := range rules {
		r := &rules[i]
		if r.Sources.Decision == "" {
			continue
		}
		e := tr.ByID(r.Sources.Decision)
		if e == nil {
			t.Errorf("Rules()[%d] (Kind=%q, FromState=%q, Verb=%q): Sources.Decision=%q does not resolve via tr.ByID",
				i, r.Kind, r.FromState, r.Verb, r.Sources.Decision)
			continue
		}
		if e.Kind != entity.KindDecision {
			t.Errorf("Rules()[%d]: Sources.Decision=%q resolves to a %s entity, expected decision",
				i, r.Sources.Decision, e.Kind)
		}
	}
}

// TestM0123_AC6_AntiRuleDecisionSourcesResolve asserts the same closure
// for spec.AntiRules() entries. Today's anti-rules carry only FP sources
// (none reference a D-NNNN), but the test exists to police future
// additions: an anti-rule landing with a non-empty Sources.Decision must
// resolve like a Rule does.
func TestM0123_AC6_AntiRuleDecisionSourcesResolve(t *testing.T) {
	t.Parallel()

	_, tr := sharedRepoTree(t)

	anti := spec.AntiRules()
	for i := range anti {
		a := &anti[i]
		if a.Sources.Decision == "" {
			continue
		}
		e := tr.ByID(a.Sources.Decision)
		if e == nil {
			t.Errorf("AntiRules()[%d] (ID=%q): Sources.Decision=%q does not resolve via tr.ByID",
				i, a.ID, a.Sources.Decision)
			continue
		}
		if e.Kind != entity.KindDecision {
			t.Errorf("AntiRules()[%d] (ID=%q): Sources.Decision=%q resolves to a %s entity, expected decision",
				i, a.ID, a.Sources.Decision, e.Kind)
		}
	}
}
