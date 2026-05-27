package spec

import (
	"github.com/23min/aiwf/internal/entity"
)

// Rules returns the closed-set legal-workflow table. Per M-0123 phase 1
// concretization, every cell encodes one (Kind, FromState, Verb, Outcome)
// position. Cells may overlap on (Kind, FromState, Verb) — the (key,
// Outcome) tuple is what's required to be unique.
//
// Drift policies under internal/policies/ assert:
//   - Every (Kind, FromState) appearing in entity.transitions /
//     entity.acTransitions / entity.tddPhaseTransitions has at least one
//     corresponding cell.
//   - Every top-level Cobra verb is referenced by at least one cell.
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
	out = append(out, authorizeKindRestrictionRules()...)
	return out
}

// GlobalRules returns the cross-cutting precondition rules that are NOT
// (Kind, FromState, Verb) cells (ADR-0013) — kept out of [Rules] so every
// per-cell consumer (the m0124/m0125 coverage drivers, the coordinate-
// resolution drift arms, key-uniqueness) iterates cells only, with no
// per-rule exclusion. Only the code-oriented AC-5 drift arms union
// Rules() and GlobalRules().
//
// Today there is exactly one global rule: the scope-reach rule — an
// authorized agent's verb is refused when the target is out of scope
// (D-0006). The scope-reach predicate returns reachability (M-0145), so
// the out-of-scope violation is expressed as `scope-reach == false`. The
// coordinate fields are intentionally zero; this rule has no cell
// position. It mirrors the M-0141 runtime gate into the spec's
// bidirectional drift net and changes no runtime behavior.
func GlobalRules() []Rule {
	return []Rule{
		{
			Preconditions:     []Predicate{{Subject: "scope-reach", Op: "==", Value: "false"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "provenance-authorization-out-of-scope",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Decision: "D-0006"},
		},
	}
}

// terminalIllegal is the common shape for the per-(kind, terminal-state) cell
// that pins "this state is terminal; FSM-transition verbs from here are
// illegal." Encodes the no-outgoing-transitions truth in the spec so the
// drift policy's "every (Kind, FromState) covered" check holds without
// implicit reasoning about FSM closure.
func terminalIllegal(k entity.Kind, state string, sources RuleSource) Rule {
	return Rule{
		Kind:              k,
		FromState:         state,
		Verb:              "promote",
		Outcome:           OutcomeIllegal,
		ExpectedErrorCode: "fsm-transition-illegal",
		RejectionLayer:    RejectionLayerVerbTime,
		BlockingStrict:    true,
		Sources:           sources,
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
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0001"}, FP: []string{"R-FP-0001"}},
		},
		// proposed → cancelled
		{
			Kind:      entity.KindEpic,
			FromState: "proposed",
			Verb:      "cancel",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0002"}, FP: []string{"R-FP-0002"}},
		},
		// active → done
		{
			Kind:      entity.KindEpic,
			FromState: "active",
			Verb:      "promote",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0003"}, FP: []string{"R-FP-0003"}},
		},
		// active → cancelled
		{
			Kind:      entity.KindEpic,
			FromState: "active",
			Verb:      "cancel",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0004"}, FP: []string{"R-FP-0004"}},
		},
		// Q5 / D-0003: cancel refuses when any child milestone is non-terminal.
		// Companion to the legal cells above (different Outcome → same key still unique).
		{
			Kind:              entity.KindEpic,
			FromState:         "proposed",
			Verb:              "cancel",
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
			Preconditions:     []Predicate{{Subject: "any-child.status", Op: "∉", Value: "milestone-terminal-set"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "epic-cancel-non-terminal-children",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{FP: []string{"R-FP-0074"}, Decision: "D-0003"},
		},
		// Terminals: done and cancelled have no outgoing transitions.
		terminalIllegal(entity.KindEpic, "done", RuleSource{Audit: []string{"R-AUDIT-0005"}, FP: []string{"R-FP-0005"}}),
		terminalIllegal(entity.KindEpic, "cancelled", RuleSource{Audit: []string{"R-AUDIT-0005"}, FP: []string{"R-FP-0006"}}),
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
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0006"}, FP: []string{"R-FP-0009"}},
		},
		// draft → cancelled
		{
			Kind:      entity.KindMilestone,
			FromState: "draft",
			Verb:      "cancel",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0007"}, FP: []string{"R-FP-0010"}},
		},
		// in_progress → done (preconditioned on no open ACs per R-FP-0061)
		{
			Kind:          entity.KindMilestone,
			FromState:     "in_progress",
			Verb:          "promote",
			Preconditions: []Predicate{{Subject: "all-children-acs.status", Op: "!=", Value: "open"}},
			Outcome:       OutcomeLegal,
			Sources:       RuleSource{Audit: []string{"R-AUDIT-0008", "R-AUDIT-0049", "R-AUDIT-0081"}, FP: []string{"R-FP-0011", "R-FP-0061"}},
		},
		// in_progress → done illegal companion: any open AC fires milestone-done-incomplete-acs.
		{
			Kind:              entity.KindMilestone,
			FromState:         "in_progress",
			Verb:              "promote",
			Preconditions:     []Predicate{{Subject: "any-child-ac.status", Op: "==", Value: "open"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "milestone-done-incomplete-acs",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0049", "R-AUDIT-0081"}, FP: []string{"R-FP-0061"}},
		},
		// in_progress → cancelled
		{
			Kind:      entity.KindMilestone,
			FromState: "in_progress",
			Verb:      "cancel",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0009"}, FP: []string{"R-FP-0012"}},
		},
		// Q6 / D-0004: cancel refuses when any AC is open.
		{
			Kind:              entity.KindMilestone,
			FromState:         "draft",
			Verb:              "cancel",
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
			Preconditions:     []Predicate{{Subject: "any-child-ac.status", Op: "==", Value: "open"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "milestone-cancel-non-terminal-acs",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{FP: []string{"R-FP-0064"}, Decision: "D-0004"},
		},
		// Terminals: done and cancelled.
		terminalIllegal(entity.KindMilestone, "done", RuleSource{Audit: []string{"R-AUDIT-0010"}, FP: []string{"R-FP-0013"}}),
		terminalIllegal(entity.KindMilestone, "cancelled", RuleSource{Audit: []string{"R-AUDIT-0010"}, FP: []string{"R-FP-0014"}}),
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
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0011"}, FP: []string{"R-FP-0016"}},
		},
		// proposed → rejected
		{
			Kind:      entity.KindADR,
			FromState: "proposed",
			Verb:      "cancel",
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
			Preconditions:     []Predicate{{Subject: "self.superseded_by", Op: "==", Value: ""}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "adr-supersession-mutual",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0013"}, FP: []string{"R-FP-0018"}},
		},
		// Q3 explicit illegal: accepted → rejected is not legal (supersession only).
		{
			Kind:              entity.KindADR,
			FromState:         "accepted",
			Verb:              "cancel",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0014"}, FP: []string{"R-FP-0021"}},
		},
		// Terminals: superseded and rejected.
		terminalIllegal(entity.KindADR, "superseded", RuleSource{Audit: []string{"R-AUDIT-0014"}, FP: []string{"R-FP-0019"}}),
		terminalIllegal(entity.KindADR, "rejected", RuleSource{Audit: []string{"R-AUDIT-0014"}, FP: []string{"R-FP-0020"}}),
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
			Preconditions: []Predicate{{Subject: "self.addressed_by", Op: "non-empty"}},
			Outcome:       OutcomeLegal,
			Sources:       RuleSource{Audit: []string{"R-AUDIT-0015", "R-AUDIT-0089"}, FP: []string{"R-FP-0023", "R-FP-0087"}},
		},
		// Q8 illegal companion: gap-addressed-has-resolver fires when addressed_by is empty.
		{
			Kind:              entity.KindGap,
			FromState:         "open",
			Verb:              "promote",
			Preconditions:     []Predicate{{Subject: "self.addressed_by", Op: "==", Value: ""}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "gap-addressed-has-resolver",
			RejectionLayer:    RejectionLayerCheckTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0089"}, FP: []string{"R-FP-0087"}},
		},
		// open → wontfix
		{
			Kind:      entity.KindGap,
			FromState: "open",
			Verb:      "cancel",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0016"}, FP: []string{"R-FP-0024"}},
		},
		// Terminals: addressed and wontfix.
		terminalIllegal(entity.KindGap, "addressed", RuleSource{Audit: []string{"R-AUDIT-0017"}, FP: []string{"R-FP-0025"}}),
		terminalIllegal(entity.KindGap, "wontfix", RuleSource{Audit: []string{"R-AUDIT-0017"}, FP: []string{"R-FP-0026"}}),
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
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0018"}, FP: []string{"R-FP-0028"}},
		},
		{
			Kind:      entity.KindDecision,
			FromState: "proposed",
			Verb:      "cancel",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0019"}, FP: []string{"R-FP-0029"}},
		},
		{
			Kind:      entity.KindDecision,
			FromState: "accepted",
			Verb:      "promote",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0020"}, FP: []string{"R-FP-0030"}},
		},
		// Terminals: superseded and rejected.
		terminalIllegal(entity.KindDecision, "superseded", RuleSource{Audit: []string{"R-AUDIT-0021"}, FP: []string{"R-FP-0031"}}),
		terminalIllegal(entity.KindDecision, "rejected", RuleSource{Audit: []string{"R-AUDIT-0021"}, FP: []string{"R-FP-0032"}}),
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
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0022"}, FP: []string{"R-FP-0035"}},
		},
		// proposed → rejected
		{
			Kind:      entity.KindContract,
			FromState: "proposed",
			Verb:      "cancel",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0023"}, FP: []string{"R-FP-0036"}},
		},
		// accepted → deprecated
		{
			Kind:      entity.KindContract,
			FromState: "accepted",
			Verb:      "promote",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0024"}, FP: []string{"R-FP-0037"}},
		},
		// Q4 / D-0002: accepted → rejected is legal (Conflict resolved Pass A wins).
		{
			Kind:      entity.KindContract,
			FromState: "accepted",
			Verb:      "cancel",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0025"}, FP: []string{"R-FP-0045"}, Decision: "D-0002"},
		},
		// deprecated → retired
		{
			Kind:      entity.KindContract,
			FromState: "deprecated",
			Verb:      "promote",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0026"}, FP: []string{"R-FP-0039"}},
		},
		// Terminals: retired and rejected.
		terminalIllegal(entity.KindContract, "retired", RuleSource{Audit: []string{"R-AUDIT-0027"}, FP: []string{"R-FP-0040"}}),
		terminalIllegal(entity.KindContract, "rejected", RuleSource{Audit: []string{"R-AUDIT-0027"}, FP: []string{"R-FP-0041"}}),
	}
}

// AC sub-FSM cells: open → {met, deferred, cancelled}; met → {deferred, cancelled}.
// Q1 (deferred is terminal) is captured by absence of outgoing cells.
// Q7 / D-0005 (AC met requires --evidence) is the preconditioned met cell.
// Q2 (self-promote illegal globally) is captured by no FromState-to-same-state cells.
func acRules() []Rule {
	return []Rule{
		// open → met (preconditioned on self.evidence non-empty per
		// D-0005). The Legal cell is split on parent.tdd: when the
		// parent milestone is tdd != required, evidence alone suffices;
		// when parent.tdd == required, the kernel's acs-tdd-audit
		// (Illegal companion below) demands tdd_phase == done. Splitting
		// here keeps every Legal cell's preconditions enumerable as flat
		// AND (no implicit "and the audit doesn't fire") — closes
		// G-0152's overlapping-cells skimp.
		{
			Kind:      KindAC,
			FromState: "open",
			Verb:      "promote",
			Preconditions: []Predicate{
				{Subject: "self.evidence", Op: "non-empty"},
				{Subject: "parent.tdd", Op: "!=", Value: "required"},
			},
			Outcome: OutcomeLegal,
			Sources: RuleSource{Audit: []string{"R-AUDIT-0034", "R-AUDIT-0195"}, FP: []string{"R-FP-0046", "R-FP-0066"}, Decision: "D-0005"},
		},
		{
			Kind:      KindAC,
			FromState: "open",
			Verb:      "promote",
			Preconditions: []Predicate{
				{Subject: "self.evidence", Op: "non-empty"},
				{Subject: "parent.tdd", Op: "==", Value: "required"},
				{Subject: "self.tdd_phase", Op: "==", Value: "done"},
			},
			Outcome: OutcomeLegal,
			Sources: RuleSource{Audit: []string{"R-AUDIT-0034", "R-AUDIT-0195"}, FP: []string{"R-FP-0046", "R-FP-0066"}, Decision: "D-0005"},
		},
		// Q7 / D-0005 illegal companion: missing --evidence triggers verb-time refusal.
		{
			Kind:              KindAC,
			FromState:         "open",
			Verb:              "promote",
			Preconditions:     []Predicate{{Subject: "self.evidence", Op: "==", Value: ""}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "ac-evidence-missing",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0195"}, FP: []string{"R-FP-0066"}, Decision: "D-0005"},
		},
		// open → deferred
		{
			Kind:      KindAC,
			FromState: "open",
			Verb:      "promote",
			Preconditions: []Predicate{
				{Subject: "self.target-state", Op: "==", Value: "deferred"},
			},
			Outcome: OutcomeLegal,
			Sources: RuleSource{Audit: []string{"R-AUDIT-0035"}, FP: []string{"R-FP-0047"}},
		},
		// open → cancelled
		{
			Kind:      KindAC,
			FromState: "open",
			Verb:      "cancel",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0036"}, FP: []string{"R-FP-0048"}},
		},
		// met → deferred (scope-change after the fact)
		{
			Kind:      KindAC,
			FromState: "met",
			Verb:      "promote",
			Preconditions: []Predicate{
				{Subject: "self.target-state", Op: "==", Value: "deferred"},
			},
			Outcome: OutcomeLegal,
			Sources: RuleSource{Audit: []string{"R-AUDIT-0037"}, FP: []string{"R-FP-0049"}},
		},
		// met → cancelled (scope-change after the fact)
		{
			Kind:      KindAC,
			FromState: "met",
			Verb:      "cancel",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0038"}, FP: []string{"R-FP-0050"}},
		},
		// Q1: deferred is terminal — explicit illegal cell for clarity.
		{
			Kind:              KindAC,
			FromState:         "deferred",
			Verb:              "promote",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0039"}, FP: []string{"R-FP-0051"}},
		},
		// cancelled is terminal (AC FSM symmetric with deferred) — explicit
		// illegal cell so M-0123/AC-5's impl→spec coverage holds for every
		// AC FSM state. Surfaced by the drift test during AC-5 authoring.
		{
			Kind:              KindAC,
			FromState:         "cancelled",
			Verb:              "promote",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0036"}, FP: []string{"R-FP-0048"}},
		},
		// AC met under tdd:required requires phase=done per R-FP-0060 / R-AUDIT-0073.
		// Encoded as a precondition on met-from-open and as a check-time finding.
		{
			Kind:              KindAC,
			FromState:         "open",
			Verb:              "promote",
			Preconditions:     []Predicate{{Subject: "parent.tdd", Op: "==", Value: "required"}, {Subject: "self.tdd_phase", Op: "!=", Value: "done"}},
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "acs-tdd-audit",
			RejectionLayer:    RejectionLayerCheckTime,
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
			Kind:      KindTDDPhase,
			FromState: "",
			Verb:      "promote",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0042"}},
		},
		{
			Kind:      KindTDDPhase,
			FromState: "red",
			Verb:      "promote",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0043"}, FP: []string{"R-FP-0054"}},
		},
		{
			Kind:      KindTDDPhase,
			FromState: "green",
			Verb:      "promote",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0044", "R-AUDIT-0045"}, FP: []string{"R-FP-0055"}},
		},
		{
			Kind:      KindTDDPhase,
			FromState: "refactor",
			Verb:      "promote",
			Outcome:   OutcomeLegal,
			Sources:   RuleSource{Audit: []string{"R-AUDIT-0046"}, FP: []string{"R-FP-0056"}},
		},
		// Q2: TDD-phase done is terminal — explicit illegal cell.
		{
			Kind:              KindTDDPhase,
			FromState:         "done",
			Verb:              "promote",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "fsm-transition-illegal",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0047"}, FP: []string{"R-FP-0057"}},
		},
	}
}

// Q15 / D-0007: authorize refuses non-{epic, milestone} scope-entity kinds.
// Four illegal cells, one per disallowed Kind.
func authorizeKindRestrictionRules() []Rule {
	return []Rule{
		{
			Kind:              entity.KindGap,
			FromState:         "open",
			Verb:              "authorize",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "authorize-kind-not-allowed",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0122"}, FP: []string{"R-FP-0133"}, Decision: "D-0007"},
		},
		{
			Kind:              entity.KindDecision,
			FromState:         "proposed",
			Verb:              "authorize",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "authorize-kind-not-allowed",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0122"}, FP: []string{"R-FP-0133"}, Decision: "D-0007"},
		},
		{
			Kind:              entity.KindContract,
			FromState:         "proposed",
			Verb:              "authorize",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "authorize-kind-not-allowed",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0122"}, FP: []string{"R-FP-0133"}, Decision: "D-0007"},
		},
		{
			Kind:              entity.KindADR,
			FromState:         "proposed",
			Verb:              "authorize",
			Outcome:           OutcomeIllegal,
			ExpectedErrorCode: "authorize-kind-not-allowed",
			RejectionLayer:    RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           RuleSource{Audit: []string{"R-AUDIT-0122"}, FP: []string{"R-FP-0133"}, Decision: "D-0007"},
		},
	}
}
