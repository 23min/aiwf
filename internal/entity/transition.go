package entity

import (
	"fmt"
	"slices"

	"github.com/23min/aiwf/internal/codes"
)

// transitions encodes the per-kind status FSM as a map from current
// status to the set of statuses you can move to via `aiwf promote` or
// `aiwf cancel`. Terminal statuses have no outgoing transitions.
//
// The PoC's FSM is deliberately one-directional — there is no "demote".
// Edit frontmatter directly if you need to back out a transition;
// markdown is the source of truth.
var transitions = map[Kind]map[Status][]Status{
	KindEpic: {
		"proposed":  {"active", "cancelled"},
		"active":    {"done", "cancelled"},
		"done":      {},
		"cancelled": {},
	},
	KindMilestone: {
		"draft":       {"in_progress", "cancelled"},
		"in_progress": {"done", "cancelled"},
		"done":        {},
		"cancelled":   {},
	},
	KindADR: {
		"proposed":   {"accepted", "rejected"},
		"accepted":   {"superseded"},
		"superseded": {},
		"rejected":   {},
	},
	KindGap: {
		"open":      {"addressed", "wontfix"},
		"addressed": {},
		"wontfix":   {},
	},
	KindDecision: {
		"proposed":   {"accepted", "rejected"},
		"accepted":   {"superseded"},
		"superseded": {},
		"rejected":   {},
	},
	KindContract: {
		"proposed":   {"accepted", "rejected"},
		"accepted":   {"deprecated", "rejected"},
		"deprecated": {"retired"},
		"retired":    {},
		"rejected":   {},
	},
}

// AllowedTransitions returns the statuses reachable from `from` for the
// given kind. Returns nil if the kind or the source status is unknown.
func AllowedTransitions(k Kind, from Status) []Status {
	kindTransitions, ok := transitions[k]
	if !ok {
		return nil
	}
	return kindTransitions[from]
}

// CodeFSMTransitionIllegal is the typed kernel-code descriptor carried by
// [FSMTransitionError]: the structured code the legal-workflow spec
// references for every illegal FSM-transition cell. It declares
// [codes.ClassLegality], the marker from which the closed legality set is
// enumerated (D-0011). Consumers see its [codes.Code.ID] string via
// [FSMTransitionError.Code].
var CodeFSMTransitionIllegal = codes.Code{ID: "fsm-transition-illegal", Class: codes.ClassLegality}

// ValidateTransition reports nil when (kind, from, to) is a legal step.
// Returns a descriptive error when from is unknown to the kind, when
// the kind itself is unknown, or when no transition from→to exists.
// An illegal transition of a *recognized* (kind, from) is reported as
// an [FSMTransitionError] carrying CodeFSMTransitionIllegal; malformed
// input (unknown kind, unrecognized from) returns a plain error.
func ValidateTransition(k Kind, from, to Status) error {
	kindTransitions, ok := transitions[k]
	if !ok {
		return fmt.Errorf("unknown kind %q", k)
	}
	allowed, knownFrom := kindTransitions[from]
	if !knownFrom {
		return fmt.Errorf("status %q is not a recognized %s state", from, k)
	}
	if slices.Contains(allowed, to) {
		return nil
	}
	return &FSMTransitionError{Kind: k, From: from, To: to, Allowed: allowed}
}

// FSMTransitionError reports an attempted status transition that the
// kind's FSM does not permit for a recognized (kind, from). It carries
// the structured CodeFSMTransitionIllegal via [FSMTransitionError.Code]
// and the transition's coordinates. An empty Allowed slice means From
// is a terminal state. Error preserves the kernel's long-standing
// message text so message-matching consumers keep working.
type FSMTransitionError struct {
	Kind    Kind
	From    Status
	To      Status
	Allowed []Status
}

// Error implements error, preserving the kernel's established phrasing
// for terminal vs. not-allowed refusals.
func (e *FSMTransitionError) Error() string {
	if len(e.Allowed) == 0 {
		return fmt.Sprintf("%s status %q is terminal; cannot transition to %q", e.Kind, e.From, e.To)
	}
	return fmt.Sprintf("%s status %q cannot transition to %q (allowed: %v)", e.Kind, e.From, e.To, e.Allowed)
}

// Code returns CodeFSMTransitionIllegal's ID, satisfying [Coded].
func (e *FSMTransitionError) Code() string { return CodeFSMTransitionIllegal.ID }

// IsTerminal reports whether (kind, status) names a terminal state in
// the kind's FSM — i.e., a state with no outgoing transitions. Returns
// false for unknown kinds and unknown statuses, so downstream checks
// keep firing on junk-status entities rather than silently exempting
// them.
//
// Derives terminality from the FSM rather than a parallel hardcoded
// list to keep one source of truth: if the FSM grows or shrinks a
// state's outgoing edges, IsTerminal tracks it automatically.
func IsTerminal(k Kind, status Status) bool {
	kindTransitions, ok := transitions[k]
	if !ok {
		return false
	}
	outgoing, known := kindTransitions[status]
	if !known {
		return false
	}
	return len(outgoing) == 0
}

// CancelTarget returns the kind's terminal-cancel status — the one
// `aiwf cancel` promotes a non-terminal entity to. Used by the cancel
// verb to know which terminal status maps to "discarded".
//
// Three kinds are status-agnostic; their `currentStatus` argument is
// ignored:
//   - epic / milestone → "cancelled"
//   - gap              → "wontfix"
//
// ADR, Decision, and Contract are state-aware. The FSM does not permit
// every non-terminal state to be cancelled — ADR and Decision's
// `accepted` exits only via `promote → superseded`, and Contract's
// `deprecated` exits via `retired` (not `rejected`). Returning the
// FSM-illegal target unconditionally would route an illegal projection
// through the cancel verb (G-0131 for Contract, G-0163 for ADR/Decision);
// returning "" surfaces it to the caller as "no cancel target."
//
//	adr.proposed       → rejected
//	adr.accepted       → ""   (exits only via promote → superseded)
//	decision.proposed  → rejected
//	decision.accepted  → ""   (exits only via promote → superseded)
//	contract.proposed  → rejected
//	contract.accepted  → rejected
//	contract.deprecated → retired
//	contract.{retired,rejected,unknown} → ""  (caller surfaces "no
//	    cancel target" rather than picking an FSM-illegal target)
//
// Unknown kinds return "" — defensive against future kind additions
// where CancelTarget hasn't been wired yet.
func CancelTarget(k Kind, currentStatus Status) Status {
	switch k {
	case KindEpic, KindMilestone:
		return StatusCancelled
	case KindADR, KindDecision:
		if currentStatus == StatusProposed {
			return StatusRejected
		}
		return ""
	case KindGap:
		return StatusWontfix
	case KindContract:
		switch currentStatus {
		case StatusProposed, StatusAccepted:
			return StatusRejected
		case StatusDeprecated:
			return StatusRetired
		}
		return ""
	}
	return ""
}

// acTransitions encodes the per-status FSM for an acceptance criterion.
// `open → met` is the normal completion path; `open → deferred` and
// `open → cancelled` are the two terminal removals. `met → deferred`
// and `met → cancelled` cover scope changes after the AC was already
// done. `deferred` and `cancelled` are terminal.
//
// Two properties of this shape are load-bearing elsewhere and easy to
// undo by accident:
//
// `met` is deliberately NOT terminal, unlike a kind's `done`. An AC is
// a claim inside a contract that can still be rescoped, so a met
// criterion may legitimately be descoped while its parent milestone
// runs; an epic is a closed unit of work. That is why cancelling a met
// AC does real work while cancelling a done epic converges.
//
// Both terminals are removal-class: `deferred` and `cancelled` each
// mean "off the milestone's contract", and neither claims the criterion
// succeeded. The AC FSM therefore has no success-terminal, which is
// what lets `cancel` converge on any terminal AC without absorbing a
// success outcome. A future success-class terminal would have to
// revisit that (see cancelAC).
var acTransitions = map[Status][]Status{
	"open":      {"met", "deferred", "cancelled"},
	"met":       {"deferred", "cancelled"},
	"deferred":  {},
	"cancelled": {},
}

// IsLegalACTransition reports whether (from, to) is a legal AC status
// transition under the FSM. Self-transitions, unknown `from`, and
// unknown `to` all return false. The AC-promoting and AC-cancelling
// verb paths consult this; `--force --reason` (Step 4) is what relaxes
// it.
func IsLegalACTransition(from, to Status) bool {
	return slices.Contains(acTransitions[from], to)
}

// IsTerminalACStatus reports whether status names a terminal state in
// the acceptance-criterion FSM — a state with no outgoing transitions.
// The AC analogue of [IsTerminal], and deliberately the same shape:
// derived from acTransitions rather than a parallel hardcoded list, so
// the answer tracks the FSM if its edges change.
//
// Returns false for a status the FSM does not know, matching
// IsTerminal, so a junk status is never silently treated as disposed —
// a verb reaching this with an unrecognized status still has to consult
// IsLegalACTransition, which refuses it.
//
// Distinct from [MilestoneCanGoDone], which asks a different
// AC-disposal question ("is every AC out of `open`?") for the
// milestone-completion guard.
func IsTerminalACStatus(status Status) bool {
	outgoing, known := acTransitions[status]
	if !known {
		return false
	}
	return len(outgoing) == 0
}

// tddPhaseTransitions encodes the linear FSM for an AC's `tdd_phase`.
// `red → green → (refactor →) done`. `refactor` is optional — `green`
// may go directly to `done`. The linearity prevents a "green without
// red" claim that the audit hook (`acs-tdd-audit`, Step 6) would
// otherwise have to reconcile after the fact.
//
// The empty string is a "pre-cycle" entry state: an AC with no
// tdd_phase yet (added before I2, or under a non-required milestone)
// may start a TDD cycle by advancing to red. Entering at green or
// later from absent is intentionally not allowed — that would
// bypass red and undermine the audit's "met requires done" rule.
var tddPhaseTransitions = map[string][]string{
	"":         {"red"},
	"red":      {"green"},
	"green":    {"refactor", "done"},
	"refactor": {"done"},
	"done":     {},
}

// IsLegalTDDPhaseTransition reports whether (from, to) is a legal
// transition along an AC's TDD phase FSM. Self-transitions, unknown
// `from`, and unknown `to` all return false.
func IsLegalTDDPhaseTransition(from, to string) bool {
	return slices.Contains(tddPhaseTransitions[from], to)
}

// MilestoneCanGoDone reports whether the milestone's ACs are in a
// state that permits the milestone itself to transition to `done`.
// Returns (true, nil) when no AC has `status: open`; returns (false,
// openACs) listing the bare AC ids (`AC-N`) that are still open.
//
// This is the AC-level precondition; the per-status milestone FSM
// (`in_progress → done`) is a separate check that ValidateTransition
// already covers. Step 6's `milestone-done-incomplete-acs` finding
// surfaces this on every `aiwf check` pass; Step 7's promote verb
// wires it into the projection.
//
// The function is milestone-specific by intent. Calling it on other
// kinds returns (true, nil) trivially — non-milestone Entities never
// carry ACs in the schema.
func MilestoneCanGoDone(m *Entity) (canGoDone bool, openACs []string) {
	if m == nil {
		return true, nil
	}
	for _, ac := range m.ACs {
		if ac.Status == StatusOpen {
			openACs = append(openACs, ac.ID)
		}
	}
	return len(openACs) == 0, openACs
}
