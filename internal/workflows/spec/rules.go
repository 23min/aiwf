package spec

import (
	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
)

// Rules returns the closed-set legal-workflow table. The enforced identity is
// (Kind, FromState, Verb, ToState, Outcome, Preconditions); distinct outcomes
// and preconditions may describe the same transition.
//
// Drift policies under internal/policies/ assert:
//   - Every (Kind, FromState) appearing in entity.transitions /
//     entity.acTransitions / entity.tddPhaseTransitions has at least one
//     corresponding cell.
//   - Every legality verb is referenced by a transition or global rule.
//   - Every legality-pertinent finding code is referenced by at least one
//     illegal-outcome cell.
//   - Every Rule satisfies the schema invariants (Outcome != Unspecified;
//     Illegal ⇒ RejectionLayer non-zero; VerbTime ⇒ BlockingStrict;
//     Legal ⇒ ExpectedErrorCode empty; Sources.Decision resolves).
func Rules() []Rule {
	var out []Rule
	out = append(out, epicRules()...)
	out = append(out, milestoneRules()...)
	out = append(out, adrRules()...)
	out = append(out, gapRules()...)
	out = append(out, decisionRules()...)
	out = append(out, contractRules()...)
	out = append(out, acRules()...)
	out = append(out, tddPhaseRules()...)
	return out
}

// GlobalRules returns the cross-cutting precondition rules that are NOT
// transition cells (ADR-0013) — kept out of [Rules] so every
// per-cell consumer (the m0124/m0125 coverage drivers, the coordinate-
// resolution drift arms, key-uniqueness) iterates cells only, with no
// per-rule exclusion. Code- and verb-oriented drift arms union
// Rules() and GlobalRules().
//
// Each entry carries its own comment naming what it refuses and the
// record that authorizes it; the cells are the enumeration, so no
// index of them is kept here to drift out of step with the slice
// below.
func GlobalRules() []Rule {
	return []Rule{
		// D-0007: autonomous-work scopes belong only to epics and milestones.
		{
			Verb: "authorize",
			Preconditions: []Predicate{
				{Subject: "self.kind", Op: "!=", Value: "epic"},
				{Subject: "self.kind", Op: "!=", Value: "milestone"},
			},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "authorize-kind-not-allowed",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0122"}, FP: []string{"R-FP-0133"}, Decision: "D-0007"},
		},
		{
			Preconditions:     []Predicate{{Subject: "scope-reach", Op: "==", Value: "false"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "provenance-authorization-out-of-scope",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Decision: "D-0006"},
		},
		{
			Verb: "authorize",
			Preconditions: []Predicate{
				{Subject: "target-agent-role", Op: "==", Value: "ai"},
				{Subject: "ritual-branch-context-present", Op: "==", Value: "false"},
				{Subject: "force", Op: "==", Value: "false"},
			},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "branch-context-required",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Decision: "ADR-0010"},
		},
		// M-0161/AC-2 (G-0201): aiwf authorize refuses opening a scope
		// on an ai/* agent when the (CurrentBranch rung, --branch rung)
		// pair is not in the legal set {(trunk, epic), (epic, milestone),
		// (milestone, patch), (epic, patch)} per ADR-0010. The single
		// rung-pair check applies regardless of whether --branch
		// references an existing local branch. Sovereign override:
		// --force --reason "...".
		{
			Verb: "authorize",
			Preconditions: []Predicate{
				{Subject: "target-agent-role", Op: "==", Value: "ai"},
				{Subject: "rung-pair-legal", Op: "==", Value: "false"},
				{Subject: "force", Op: "==", Value: "false"},
			},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "rung-pair-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Decision: "ADR-0010"},
		},
		// ADR-0047: a transition the kernel treats as a sovereign act is
		// refused for any actor that is not human/, before anything is
		// written. Cross-cutting rather than per-cell because the closed
		// set spans several (Kind, FromState, Verb) coordinates and is
		// reached by more than one verb — promote for the opening and
		// completion edges, cancel for the two cancelled ones.
		//
		// The subject names the kernel's closed set rather than
		// enumerating today's entries, so this cell cannot drift out of
		// step with entity.SovereignActShapes() as that set widens. The
		// membership itself lives in one place, in the entity package,
		// and is not copied here.
		//
		// ExpectedErrorCode is empty because the refusal carries no
		// finding code — it is a bare verb error, which is also why it
		// exits as a usage error rather than as the legality refusal it
		// is. G-0649 tracks that; until it resolves, this rule stays
		// outside the two code-oriented drift arms, which skip any rule
		// with an empty code.
		{
			Preconditions: []Predicate{
				{Subject: "sovereign-act-shape", Op: "==", Value: "true"},
				{Subject: "actor-role", Op: "!=", Value: "human"},
				{Subject: "force", Op: "==", Value: "false"},
			},
			Outcome:        OutcomeIllegal,
			RejectionLayer: RejectionLayerVerbTime,
			BlockingStrict: true,
			Sources:        RuleSource{Decision: "ADR-0047"},
		},
	}
}

// Epic FSM cells: proposed → {active, cancelled}; active → {done, cancelled}.
// Plus the Q5 / D-0003 preconditioned-cancel illegal cell.
// Plus terminal-state coverage (done, cancelled) per R-FP-0005, R-FP-0006.
func epicRules() []Rule {
	return []Rule{
		// proposed → active (ratification)
		{
			Kind:      entity.KindEpic,
			FromState: "proposed",
			Verb:      "promote",
			ToState:   "active",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0001"}, FP: []string{"R-FP-0001"}},
		},
		{
			Kind:      entity.KindEpic,
			FromState: "proposed",
			Verb:      "promote",
			ToState:   "cancelled",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0001"}, FP: []string{"R-FP-0001"}},
		},
		// proposed → cancelled
		{
			Kind:      entity.KindEpic,
			FromState: "proposed",
			Verb:      "cancel",
			ToState:   "cancelled",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0002"}, FP: []string{"R-FP-0002"}},
		},
		// active → done
		{
			Kind:      entity.KindEpic,
			FromState: "active",
			Verb:      "promote",
			ToState:   "done",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0003"}, FP: []string{"R-FP-0003"}},
		},
		{
			Kind:      entity.KindEpic,
			FromState: "active",
			Verb:      "promote",
			ToState:   "cancelled",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0003"}, FP: []string{"R-FP-0003"}},
		},
		// active → cancelled
		{
			Kind:      entity.KindEpic,
			FromState: "active",
			Verb:      "cancel",
			ToState:   "cancelled",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0004"}, FP: []string{"R-FP-0004"}},
		},
		// Q5 / D-0003: cancel refuses when any child milestone is non-terminal.
		// Companion to the legal cells above (different Outcome → same key still unique).
		{
			Kind:              entity.KindEpic,
			FromState:         "proposed",
			Verb:              "cancel",
			ToState:           "cancelled",
			Preconditions:     []Predicate{{Subject: "any-child.status", Op: "∉", Value: "milestone-terminal-set"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "epic-cancel-non-terminal-children",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{FP: []string{"R-FP-0074"}, Decision: "D-0003"},
		},
		{
			Kind:              entity.KindEpic,
			FromState:         "active",
			Verb:              "cancel",
			ToState:           "cancelled",
			Preconditions:     []Predicate{{Subject: "any-child.status", Op: "∉", Value: "milestone-terminal-set"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "epic-cancel-non-terminal-children",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{FP: []string{"R-FP-0074"}, Decision: "D-0003"},
		},
		// G-0393: promote refuses reaching a terminal status the same
		// way cancel does, when any child milestone is non-terminal.
		// Companion to the "active → done" legal cell above (different
		// Outcome → same key still unique).
		//
		// Sources is deliberately empty rather than citing D-0003: that
		// decision's own "Spec cell" section names Verb: "cancel"
		// explicitly (it ratified the cancel-verb refuse pattern), and
		// D-0003 is `status: accepted` -- durable, not something later
		// scope additions rewrite in place (CLAUDE.md's aiwfx-record-
		// decision constraint). A fresh decision for the promote-side
		// generalization is not an option either: AC-2's own hardcoded
		// M-0123 audit set (TestM0123_AC2_DecisionSourcesPopulatedFor-
		// FPOnlyAndConflict) only accepts {D-0002..D-0007}, all already
		// spoken for. Absent a genuinely-scoped decision to cite, an
		// empty Sources is more honest than borrowing one that does not
		// name this verb. (G-0394 independently proposed the same cell,
		// scoped to `done` only; this cell supersedes it, covering both
		// of Promote's terminal targets for KindEpic.)
		{
			Kind:              entity.KindEpic,
			FromState:         "active",
			Verb:              "promote",
			ToState:           "done",
			Preconditions:     []Predicate{{Subject: "any-child.status", Op: "∉", Value: "milestone-terminal-set"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "epic-promote-non-terminal-children",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
		},
		{
			Kind:              entity.KindEpic,
			FromState:         "active",
			Verb:              "promote",
			ToState:           "cancelled",
			Preconditions:     []Predicate{{Subject: "any-child.status", Op: "∉", Value: "milestone-terminal-set"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "epic-promote-non-terminal-children",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
		},
		// Terminals: done and cancelled have no outgoing transitions.
		{
			Kind:              entity.KindEpic,
			FromState:         "done",
			Verb:              "promote",
			ToState:           "proposed",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0005"}, FP: []string{"R-FP-0005"}},
		},
		{
			Kind:              entity.KindEpic,
			FromState:         "done",
			Verb:              "promote",
			ToState:           "active",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0005"}, FP: []string{"R-FP-0005"}},
		},
		{
			Kind:      entity.KindEpic,
			FromState: "done",
			Verb:      "promote",
			ToState:   "done",
			Outcome:   OutcomeNoOp,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0005"}, FP: []string{"R-FP-0005"}},
		},
		{
			Kind:              entity.KindEpic,
			FromState:         "done",
			Verb:              "promote",
			ToState:           "cancelled",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0005"}, FP: []string{"R-FP-0005"}},
		},
		{
			Kind:              entity.KindEpic,
			FromState:         "cancelled",
			Verb:              "promote",
			ToState:           "proposed",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0005"}, FP: []string{"R-FP-0006"}},
		},
		{
			Kind:              entity.KindEpic,
			FromState:         "cancelled",
			Verb:              "promote",
			ToState:           "active",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0005"}, FP: []string{"R-FP-0006"}},
		},
		{
			Kind:              entity.KindEpic,
			FromState:         "cancelled",
			Verb:              "promote",
			ToState:           "done",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0005"}, FP: []string{"R-FP-0006"}},
		},
		{
			Kind:      entity.KindEpic,
			FromState: "cancelled",
			Verb:      "promote",
			ToState:   "cancelled",
			Outcome:   OutcomeNoOp,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0005"}, FP: []string{"R-FP-0006"}},
		},
	}
}

// Milestone FSM cells: draft → {in_progress, cancelled}; in_progress → {done, cancelled}.
// Plus the Q6 / D-0004 preconditioned-cancel illegal cell.
// Plus the R-FP-0061 milestone-done-requires-no-open-acs precondition.
func milestoneRules() []Rule {
	return []Rule{
		// draft → in_progress
		{
			Kind:      entity.KindMilestone,
			FromState: "draft",
			Verb:      "promote",
			ToState:   "in_progress",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0006"}, FP: []string{"R-FP-0009"}},
		},
		{
			Kind:      entity.KindMilestone,
			FromState: "draft",
			Verb:      "promote",
			ToState:   "cancelled",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0006"}, FP: []string{"R-FP-0009"}},
		},
		// draft → cancelled
		{
			Kind:      entity.KindMilestone,
			FromState: "draft",
			Verb:      "cancel",
			ToState:   "cancelled",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0007"}, FP: []string{"R-FP-0010"}},
		},
		// in_progress → done (preconditioned on no open ACs per R-FP-0061)
		{
			Kind:          entity.KindMilestone,
			FromState:     "in_progress",
			Verb:          "promote",
			ToState:       "done",
			Preconditions: []Predicate{{Subject: "all-children-acs.status", Op: "!=", Value: "open"}},
			Outcome:       OutcomeLegal,
			Sources:       RuleSource{Audit: []string{"R-AUDIT-0008", "R-AUDIT-0049", "R-AUDIT-0081"}, FP: []string{"R-FP-0011", "R-FP-0061"}},
		},
		{
			Kind:          entity.KindMilestone,
			FromState:     "in_progress",
			Verb:          "promote",
			ToState:       "cancelled",
			Preconditions: []Predicate{{Subject: "all-children-acs.status", Op: "!=", Value: "open"}},
			Outcome:       OutcomeLegal,
			Sources:       RuleSource{Audit: []string{"R-AUDIT-0008", "R-AUDIT-0049", "R-AUDIT-0081"}, FP: []string{"R-FP-0011", "R-FP-0061"}},
		},
		// in_progress → done illegal companion: any open AC fires
		// milestone-done-incomplete-acs.
		{
			Kind:              entity.KindMilestone,
			FromState:         "in_progress",
			Verb:              "promote",
			ToState:           "done",
			Preconditions:     []Predicate{{Subject: "any-child-ac.status", Op: "==", Value: "open"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: check.CodeMilestoneDoneIncompleteACs,
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0049", "R-AUDIT-0081"}, FP: []string{"R-FP-0061"}},
		},
		{
			Kind:              entity.KindMilestone,
			FromState:         "in_progress",
			Verb:              "promote",
			ToState:           "cancelled",
			Preconditions:     []Predicate{{Subject: "any-child-ac.status", Op: "==", Value: "open"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "milestone-promote-non-terminal-acs",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0049", "R-AUDIT-0081"}, FP: []string{"R-FP-0061"}},
		},
		// in_progress → cancelled
		{
			Kind:      entity.KindMilestone,
			FromState: "in_progress",
			Verb:      "cancel",
			ToState:   "cancelled",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0009"}, FP: []string{"R-FP-0012"}},
		},
		// Q6 / D-0004: cancel refuses when any AC is open.
		{
			Kind:              entity.KindMilestone,
			FromState:         "draft",
			Verb:              "cancel",
			ToState:           "cancelled",
			Preconditions:     []Predicate{{Subject: "any-child-ac.status", Op: "==", Value: "open"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "milestone-cancel-non-terminal-acs",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{FP: []string{"R-FP-0064"}, Decision: "D-0004"},
		},
		{
			Kind:              entity.KindMilestone,
			FromState:         "in_progress",
			Verb:              "cancel",
			ToState:           "cancelled",
			Preconditions:     []Predicate{{Subject: "any-child-ac.status", Op: "==", Value: "open"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "milestone-cancel-non-terminal-acs",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{FP: []string{"R-FP-0064"}, Decision: "D-0004"},
		},
		// Promotion to cancelled refuses while any AC is open.
		{
			Kind:              entity.KindMilestone,
			FromState:         "draft",
			Verb:              "promote",
			ToState:           "cancelled",
			Preconditions:     []Predicate{{Subject: "any-child-ac.status", Op: "==", Value: "open"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "milestone-promote-non-terminal-acs",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
		},
		// Terminals: done and cancelled.
		{
			Kind:              entity.KindMilestone,
			FromState:         "done",
			Verb:              "promote",
			ToState:           "draft",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0010"}, FP: []string{"R-FP-0013"}},
		},
		{
			Kind:              entity.KindMilestone,
			FromState:         "done",
			Verb:              "promote",
			ToState:           "in_progress",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0010"}, FP: []string{"R-FP-0013"}},
		},
		{
			Kind:      entity.KindMilestone,
			FromState: "done",
			Verb:      "promote",
			ToState:   "done",
			Outcome:   OutcomeNoOp,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0010"}, FP: []string{"R-FP-0013"}},
		},
		{
			Kind:              entity.KindMilestone,
			FromState:         "done",
			Verb:              "promote",
			ToState:           "cancelled",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0010"}, FP: []string{"R-FP-0013"}},
		},
		{
			Kind:              entity.KindMilestone,
			FromState:         "cancelled",
			Verb:              "promote",
			ToState:           "draft",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0010"}, FP: []string{"R-FP-0014"}},
		},
		{
			Kind:              entity.KindMilestone,
			FromState:         "cancelled",
			Verb:              "promote",
			ToState:           "in_progress",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0010"}, FP: []string{"R-FP-0014"}},
		},
		{
			Kind:              entity.KindMilestone,
			FromState:         "cancelled",
			Verb:              "promote",
			ToState:           "done",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0010"}, FP: []string{"R-FP-0014"}},
		},
		{
			Kind:      entity.KindMilestone,
			FromState: "cancelled",
			Verb:      "promote",
			ToState:   "cancelled",
			Outcome:   OutcomeNoOp,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0010"}, FP: []string{"R-FP-0014"}},
		},
	}
}

// ADR FSM cells: proposed → {accepted, rejected}; accepted → superseded.
// Q3 (accepted → rejected illegal) is implicit in the FSM and captured by
// the absence of a Legal cell for that transition — but we add an Illegal
// cell to make the discipline explicit and to ground the drift policy's
// reference for "rejected from accepted is not legal."
func adrRules() []Rule {
	return []Rule{
		// proposed → accepted
		{
			Kind:      entity.KindADR,
			FromState: "proposed",
			Verb:      "promote",
			ToState:   "accepted",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0011"}, FP: []string{"R-FP-0016"}},
		},
		{
			Kind:      entity.KindADR,
			FromState: "proposed",
			Verb:      "promote",
			ToState:   "rejected",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0011"}, FP: []string{"R-FP-0016"}},
		},
		// proposed → rejected
		{
			Kind:      entity.KindADR,
			FromState: "proposed",
			Verb:      "cancel",
			ToState:   "rejected",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0012"}, FP: []string{"R-FP-0017"}},
		},
		// accepted → superseded (preconditioned on self.superseded_by
		// non-empty per adr-supersession-mutual; mirrors the gap
		// open→addressed shape from R-AUDIT-0089). G-0152 records the
		// spec-vs-kernel drift this pair closes.
		{
			Kind:          entity.KindADR,
			FromState:     "accepted",
			Verb:          "promote",
			ToState:       "superseded",
			Preconditions: []Predicate{{Subject: "self.superseded_by", Op: "non-empty"}},
			Outcome:       OutcomeLegal,
			Sources:       RuleSource{Audit: []string{"R-AUDIT-0013"}, FP: []string{"R-FP-0018"}},
		},
		// adr-supersession-mutual illegal companion: missing
		// --superseded-by triggers verb-time refusal. Surfaced via
		// M-0124/AC-3's per-cell positive driver (gap G-0152).
		{
			Kind:              entity.KindADR,
			FromState:         "accepted",
			Verb:              "promote",
			ToState:           "superseded",
			Preconditions:     []Predicate{{Subject: "self.superseded_by", Op: "==", Value: ""}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: check.CodeADRSupersessionMutual,
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0013"}, FP: []string{"R-FP-0018"}},
		},
		// Q3 explicit illegal: accepted → rejected is not legal (supersession only).
		{
			Kind:              entity.KindADR,
			FromState:         "accepted",
			Verb:              "cancel",
			ToState:           "rejected",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0014"}, FP: []string{"R-FP-0021"}},
		},
		// Terminals: superseded and rejected.
		{
			Kind:              entity.KindADR,
			FromState:         "superseded",
			Verb:              "promote",
			ToState:           "proposed",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0014"}, FP: []string{"R-FP-0019"}},
		},
		{
			Kind:              entity.KindADR,
			FromState:         "superseded",
			Verb:              "promote",
			ToState:           "accepted",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0014"}, FP: []string{"R-FP-0019"}},
		},
		{
			Kind:      entity.KindADR,
			FromState: "superseded",
			Verb:      "promote",
			ToState:   "superseded",
			Outcome:   OutcomeNoOp,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0014"}, FP: []string{"R-FP-0019"}},
		},
		{
			Kind:              entity.KindADR,
			FromState:         "superseded",
			Verb:              "promote",
			ToState:           "rejected",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0014"}, FP: []string{"R-FP-0019"}},
		},
		{
			Kind:              entity.KindADR,
			FromState:         "rejected",
			Verb:              "promote",
			ToState:           "proposed",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0014"}, FP: []string{"R-FP-0020"}},
		},
		{
			Kind:              entity.KindADR,
			FromState:         "rejected",
			Verb:              "promote",
			ToState:           "accepted",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0014"}, FP: []string{"R-FP-0020"}},
		},
		{
			Kind:              entity.KindADR,
			FromState:         "rejected",
			Verb:              "promote",
			ToState:           "superseded",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0014"}, FP: []string{"R-FP-0020"}},
		},
		{
			Kind:      entity.KindADR,
			FromState: "rejected",
			Verb:      "promote",
			ToState:   "rejected",
			Outcome:   OutcomeNoOp,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0014"}, FP: []string{"R-FP-0020"}},
		},
	}
}

// Gap FSM cells: open → {addressed, wontfix}.
// Q8 (gap addressed requires addressed_by reference) is the preconditioned cell.
func gapRules() []Rule {
	return []Rule{
		// open → addressed (preconditioned on addressed_by non-empty per R-AUDIT-0089)
		{
			Kind:          entity.KindGap,
			FromState:     "open",
			Verb:          "promote",
			ToState:       "addressed",
			Preconditions: []Predicate{{Subject: "self.addressed_by", Op: "non-empty"}},
			Outcome:       OutcomeLegal,
			Sources:       RuleSource{Audit: []string{"R-AUDIT-0015", "R-AUDIT-0089"}, FP: []string{"R-FP-0023", "R-FP-0087"}},
		},
		// Q8 illegal companion: gap-addressed-has-resolver fires when addressed_by is empty.
		{
			Kind:              entity.KindGap,
			FromState:         "open",
			Verb:              "promote",
			ToState:           "addressed",
			Preconditions:     []Predicate{{Subject: "self.addressed_by", Op: "==", Value: ""}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: check.CodeGapAddressedHasResolver,
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0089"}, FP: []string{"R-FP-0087"}},
		},
		// open → wontfix
		{
			Kind:      entity.KindGap,
			FromState: "open",
			Verb:      "cancel",
			ToState:   "wontfix",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0016"}, FP: []string{"R-FP-0024"}},
		},
		// Terminals: addressed and wontfix.
		{
			Kind:              entity.KindGap,
			FromState:         "addressed",
			Verb:              "promote",
			ToState:           "open",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0017"}, FP: []string{"R-FP-0025"}},
		},
		{
			Kind:      entity.KindGap,
			FromState: "addressed",
			Verb:      "promote",
			ToState:   "addressed",
			Outcome:   OutcomeNoOp,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0017"}, FP: []string{"R-FP-0025"}},
		},
		{
			Kind:              entity.KindGap,
			FromState:         "addressed",
			Verb:              "promote",
			ToState:           "wontfix",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0017"}, FP: []string{"R-FP-0025"}},
		},
		{
			Kind:              entity.KindGap,
			FromState:         "wontfix",
			Verb:              "promote",
			ToState:           "open",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0017"}, FP: []string{"R-FP-0026"}},
		},
		{
			Kind:              entity.KindGap,
			FromState:         "wontfix",
			Verb:              "promote",
			ToState:           "addressed",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0017"}, FP: []string{"R-FP-0026"}},
		},
		{
			Kind:      entity.KindGap,
			FromState: "wontfix",
			Verb:      "promote",
			ToState:   "wontfix",
			Outcome:   OutcomeNoOp,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0017"}, FP: []string{"R-FP-0026"}},
		},
	}
}

// Decision FSM cells: proposed → {accepted, rejected}; accepted → superseded.
// Structurally identical to ADR.
func decisionRules() []Rule {
	return []Rule{
		{
			Kind:      entity.KindDecision,
			FromState: "proposed",
			Verb:      "promote",
			ToState:   "accepted",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0018"}, FP: []string{"R-FP-0028"}},
		},
		{
			Kind:      entity.KindDecision,
			FromState: "proposed",
			Verb:      "promote",
			ToState:   "rejected",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0018"}, FP: []string{"R-FP-0028"}},
		},
		{
			Kind:      entity.KindDecision,
			FromState: "proposed",
			Verb:      "cancel",
			ToState:   "rejected",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0019"}, FP: []string{"R-FP-0029"}},
		},
		{
			Kind:      entity.KindDecision,
			FromState: "accepted",
			Verb:      "promote",
			ToState:   "superseded",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0020"}, FP: []string{"R-FP-0030"}},
		},
		// Terminals: superseded and rejected.
		{
			Kind:              entity.KindDecision,
			FromState:         "superseded",
			Verb:              "promote",
			ToState:           "proposed",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0021"}, FP: []string{"R-FP-0031"}},
		},
		{
			Kind:              entity.KindDecision,
			FromState:         "superseded",
			Verb:              "promote",
			ToState:           "accepted",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0021"}, FP: []string{"R-FP-0031"}},
		},
		{
			Kind:      entity.KindDecision,
			FromState: "superseded",
			Verb:      "promote",
			ToState:   "superseded",
			Outcome:   OutcomeNoOp,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0021"}, FP: []string{"R-FP-0031"}},
		},
		{
			Kind:              entity.KindDecision,
			FromState:         "superseded",
			Verb:              "promote",
			ToState:           "rejected",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0021"}, FP: []string{"R-FP-0031"}},
		},
		{
			Kind:              entity.KindDecision,
			FromState:         "rejected",
			Verb:              "promote",
			ToState:           "proposed",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0021"}, FP: []string{"R-FP-0032"}},
		},
		{
			Kind:              entity.KindDecision,
			FromState:         "rejected",
			Verb:              "promote",
			ToState:           "accepted",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0021"}, FP: []string{"R-FP-0032"}},
		},
		{
			Kind:              entity.KindDecision,
			FromState:         "rejected",
			Verb:              "promote",
			ToState:           "superseded",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0021"}, FP: []string{"R-FP-0032"}},
		},
		{
			Kind:      entity.KindDecision,
			FromState: "rejected",
			Verb:      "promote",
			ToState:   "rejected",
			Outcome:   OutcomeNoOp,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0021"}, FP: []string{"R-FP-0032"}},
		},
	}
}

// Contract FSM cells: proposed → {accepted, rejected};
// accepted → {deprecated, rejected}; deprecated → retired.
// Q4 / D-0002: accepted → rejected IS legal (asymmetric to ADR, deliberate per D-0002).
func contractRules() []Rule {
	return []Rule{
		// proposed → accepted
		{
			Kind:      entity.KindContract,
			FromState: "proposed",
			Verb:      "promote",
			ToState:   "accepted",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0022"}, FP: []string{"R-FP-0035"}},
		},
		{
			Kind:      entity.KindContract,
			FromState: "proposed",
			Verb:      "promote",
			ToState:   "rejected",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0022"}, FP: []string{"R-FP-0035"}},
		},
		// proposed → rejected
		{
			Kind:      entity.KindContract,
			FromState: "proposed",
			Verb:      "cancel",
			ToState:   "rejected",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0023"}, FP: []string{"R-FP-0036"}},
		},
		// accepted → deprecated
		{
			Kind:      entity.KindContract,
			FromState: "accepted",
			Verb:      "promote",
			ToState:   "deprecated",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0024"}, FP: []string{"R-FP-0037"}},
		},
		{
			Kind:      entity.KindContract,
			FromState: "accepted",
			Verb:      "promote",
			ToState:   "rejected",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0024"}, FP: []string{"R-FP-0037"}},
		},
		// Q4 / D-0002: accepted → rejected is legal (Conflict resolved Pass A wins).
		{
			Kind:      entity.KindContract,
			FromState: "accepted",
			Verb:      "cancel",
			ToState:   "rejected",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0025"}, FP: []string{"R-FP-0045"}, Decision: "D-0002"},
		},
		// deprecated → retired
		{
			Kind:      entity.KindContract,
			FromState: "deprecated",
			Verb:      "promote",
			ToState:   "retired",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0026"}, FP: []string{"R-FP-0039"}},
		},
		// Terminals: retired and rejected.
		{
			Kind:              entity.KindContract,
			FromState:         "retired",
			Verb:              "promote",
			ToState:           "proposed",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0027"}, FP: []string{"R-FP-0040"}},
		},
		{
			Kind:              entity.KindContract,
			FromState:         "retired",
			Verb:              "promote",
			ToState:           "accepted",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0027"}, FP: []string{"R-FP-0040"}},
		},
		{
			Kind:              entity.KindContract,
			FromState:         "retired",
			Verb:              "promote",
			ToState:           "deprecated",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0027"}, FP: []string{"R-FP-0040"}},
		},
		{
			Kind:      entity.KindContract,
			FromState: "retired",
			Verb:      "promote",
			ToState:   "retired",
			Outcome:   OutcomeNoOp,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0027"}, FP: []string{"R-FP-0040"}},
		},
		{
			Kind:              entity.KindContract,
			FromState:         "retired",
			Verb:              "promote",
			ToState:           "rejected",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0027"}, FP: []string{"R-FP-0040"}},
		},
		{
			Kind:              entity.KindContract,
			FromState:         "rejected",
			Verb:              "promote",
			ToState:           "proposed",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0027"}, FP: []string{"R-FP-0041"}},
		},
		{
			Kind:              entity.KindContract,
			FromState:         "rejected",
			Verb:              "promote",
			ToState:           "accepted",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0027"}, FP: []string{"R-FP-0041"}},
		},
		{
			Kind:              entity.KindContract,
			FromState:         "rejected",
			Verb:              "promote",
			ToState:           "deprecated",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0027"}, FP: []string{"R-FP-0041"}},
		},
		{
			Kind:              entity.KindContract,
			FromState:         "rejected",
			Verb:              "promote",
			ToState:           "retired",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0027"}, FP: []string{"R-FP-0041"}},
		},
		{
			Kind:      entity.KindContract,
			FromState: "rejected",
			Verb:      "promote",
			ToState:   "rejected",
			Outcome:   OutcomeNoOp,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0027"}, FP: []string{"R-FP-0041"}},
		},
	}
}

// AC sub-FSM cells: open → {met, deferred, cancelled}; met → {deferred, cancelled}.
// Q1 (deferred is terminal) permits no state-changing legal cell.
//
// Terminal status requests converge on the current status and reject other
// targets; the explicit rows distinguish those outcomes.
func acRules() []Rule {
	return []Rule{
		// open → met. The Legal cell is split on parent.tdd: when the
		// parent milestone is tdd != required, promotion is unconditioned;
		// when parent.tdd == required, the kernel's acs-tdd-audit
		// (Illegal companion below) demands tdd_phase == done. Splitting
		// here keeps every Legal cell's preconditions enumerable as flat
		// AND (no implicit "and the audit doesn't fire") — closes
		// G-0152's overlapping-cells skimp.
		{
			Kind:      KindAC,
			FromState: "open",
			Verb:      "promote",
			ToState:   "met",
			Preconditions: []Predicate{
				{Subject: "parent.tdd", Op: "!=", Value: "required"},
			},
			Outcome: OutcomeLegal,
			Sources: RuleSource{Audit: []string{"R-AUDIT-0034"}, FP: []string{"R-FP-0046"}},
		},
		{
			Kind:      KindAC,
			FromState: "open",
			Verb:      "promote",
			ToState:   "met",
			Preconditions: []Predicate{
				{Subject: "parent.tdd", Op: "==", Value: "required"},
				{Subject: "self.tdd_phase", Op: "==", Value: "done"},
			},
			Outcome: OutcomeLegal,
			Sources: RuleSource{Audit: []string{"R-AUDIT-0034"}, FP: []string{"R-FP-0046"}},
		},
		// open → deferred
		{
			Kind:      KindAC,
			FromState: "open",
			Verb:      "promote",
			ToState:   "deferred",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0035"}, FP: []string{"R-FP-0047"}},
		},
		// open → cancelled
		{
			Kind:      KindAC,
			FromState: "open",
			Verb:      "cancel",
			ToState:   "cancelled",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0036"}, FP: []string{"R-FP-0048"}},
		},
		// met → deferred (scope-change after the fact)
		{
			Kind:      KindAC,
			FromState: "met",
			Verb:      "promote",
			ToState:   "deferred",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0037"}, FP: []string{"R-FP-0049"}},
		},
		// met → cancelled (scope-change after the fact)
		{
			Kind:      KindAC,
			FromState: "met",
			Verb:      "cancel",
			ToState:   "cancelled",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0038"}, FP: []string{"R-FP-0050"}},
		},
		// Q1: deferred is terminal — self-convergence and other-target refusals.
		//
		// The self-target converges; every other target is refused.
		{
			Kind:              KindAC,
			FromState:         "deferred",
			Verb:              "promote",
			ToState:           "open",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0039"}, FP: []string{"R-FP-0051"}},
		},
		{
			Kind:              KindAC,
			FromState:         "deferred",
			Verb:              "promote",
			ToState:           "met",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0039"}, FP: []string{"R-FP-0051"}},
		},
		{
			Kind:      KindAC,
			FromState: "deferred",
			Verb:      "promote",
			ToState:   "deferred",
			Outcome:   OutcomeNoOp,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0039"}, FP: []string{"R-FP-0051"}},
		},
		{
			Kind:              KindAC,
			FromState:         "deferred",
			Verb:              "promote",
			ToState:           "cancelled",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0039"}, FP: []string{"R-FP-0051"}},
		},
		// Cancelled has the same terminal behavior as deferred.
		{
			Kind:              KindAC,
			FromState:         "cancelled",
			Verb:              "promote",
			ToState:           "open",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0036"}, FP: []string{"R-FP-0048"}},
		},
		{
			Kind:              KindAC,
			FromState:         "cancelled",
			Verb:              "promote",
			ToState:           "met",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0036"}, FP: []string{"R-FP-0048"}},
		},
		{
			Kind:              KindAC,
			FromState:         "cancelled",
			Verb:              "promote",
			ToState:           "deferred",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0036"}, FP: []string{"R-FP-0048"}},
		},
		{
			Kind:      KindAC,
			FromState: "cancelled",
			Verb:      "promote",
			ToState:   "cancelled",
			Outcome:   OutcomeNoOp,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0036"}, FP: []string{"R-FP-0048"}},
		},
		// AC met under tdd:required requires phase=done per R-FP-0060 / R-AUDIT-0073.
		// Encoded as a precondition on met-from-open and as a check-time finding.
		{
			Kind:              KindAC,
			FromState:         "open",
			Verb:              "promote",
			ToState:           "met",
			Preconditions:     []Predicate{{Subject: "parent.tdd", Op: "==", Value: "required"}, {Subject: "self.tdd_phase", Op: "!=", Value: "done"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: check.CodeACsTDDAudit,
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0073"}, FP: []string{"R-FP-0060"}},
		},
	}
}

// TDD-phase sub-FSM cells: "" → red; red → green; green → {refactor, done};
// refactor → done.
func tddPhaseRules() []Rule {
	return []Rule{
		{
			Kind:          KindTDDPhase,
			FromState:     "red",
			Verb:          "promote",
			ToState:       "red",
			Preconditions: []Predicate{{Subject: "self.tests", Op: "==", Value: ""}},
			Outcome:       OutcomeNoOp,
		},
		{
			Kind:              KindTDDPhase,
			FromState:         "red",
			Verb:              "promote",
			ToState:           "red",
			Preconditions:     []Predicate{{Subject: "self.tests", Op: "non-empty"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
		},
		{
			Kind:          KindTDDPhase,
			FromState:     "green",
			Verb:          "promote",
			ToState:       "green",
			Preconditions: []Predicate{{Subject: "self.tests", Op: "==", Value: ""}},
			Outcome:       OutcomeNoOp,
		},
		{
			Kind:              KindTDDPhase,
			FromState:         "green",
			Verb:              "promote",
			ToState:           "green",
			Preconditions:     []Predicate{{Subject: "self.tests", Op: "non-empty"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
		},
		{
			Kind:          KindTDDPhase,
			FromState:     "refactor",
			Verb:          "promote",
			ToState:       "refactor",
			Preconditions: []Predicate{{Subject: "self.tests", Op: "==", Value: ""}},
			Outcome:       OutcomeNoOp,
		},
		{
			Kind:              KindTDDPhase,
			FromState:         "refactor",
			Verb:              "promote",
			ToState:           "refactor",
			Preconditions:     []Predicate{{Subject: "self.tests", Op: "non-empty"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
		},
		{
			Kind:          KindTDDPhase,
			FromState:     "done",
			Verb:          "promote",
			ToState:       "done",
			Preconditions: []Predicate{{Subject: "self.tests", Op: "==", Value: ""}},
			Outcome:       OutcomeNoOp,
		},
		{
			Kind:              KindTDDPhase,
			FromState:         "done",
			Verb:              "promote",
			ToState:           "done",
			Preconditions:     []Predicate{{Subject: "self.tests", Op: "non-empty"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
		},
		{
			Kind:      KindTDDPhase,
			FromState: "",
			Verb:      "promote",
			ToState:   "red",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0042"}},
		},
		{
			Kind:      KindTDDPhase,
			FromState: "red",
			Verb:      "promote",
			ToState:   "green",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0043"}, FP: []string{"R-FP-0054"}},
		},
		{
			Kind:      KindTDDPhase,
			FromState: "green",
			Verb:      "promote",
			ToState:   "refactor",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0044", "R-AUDIT-0045"}, FP: []string{"R-FP-0055"}},
		},
		{
			Kind:      KindTDDPhase,
			FromState: "green",
			Verb:      "promote",
			ToState:   "done",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0044", "R-AUDIT-0045"}, FP: []string{"R-FP-0055"}},
		},
		{
			Kind:      KindTDDPhase,
			FromState: "refactor",
			Verb:      "promote",
			ToState:   "done",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0046"}, FP: []string{"R-FP-0056"}},
		},
		// TDD-phase done refuses every other phase.
		{
			Kind:              KindTDDPhase,
			FromState:         "done",
			Verb:              "promote",
			ToState:           "red",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0047"}, FP: []string{"R-FP-0057"}},
		},
		{
			Kind:              KindTDDPhase,
			FromState:         "done",
			Verb:              "promote",
			ToState:           "green",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0047"}, FP: []string{"R-FP-0057"}},
		},
		{
			Kind:              KindTDDPhase,
			FromState:         "done",
			Verb:              "promote",
			ToState:           "refactor",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0047"}, FP: []string{"R-FP-0057"}},
		},
	}
}
