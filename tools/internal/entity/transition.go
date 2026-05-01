package entity

import "fmt"

// transitions encodes the per-kind status FSM as a map from current
// status to the set of statuses you can move to via `aiwf promote` or
// `aiwf cancel`. Terminal statuses have no outgoing transitions.
//
// The PoC's FSM is deliberately one-directional — there is no "demote".
// Edit frontmatter directly if you need to back out a transition;
// markdown is the source of truth.
var transitions = map[Kind]map[string][]string{
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
func AllowedTransitions(k Kind, from string) []string {
	kindTransitions, ok := transitions[k]
	if !ok {
		return nil
	}
	return kindTransitions[from]
}

// ValidateTransition reports nil when (kind, from, to) is a legal step.
// Returns a descriptive error when from is unknown to the kind, when
// the kind itself is unknown, or when no transition from→to exists.
func ValidateTransition(k Kind, from, to string) error {
	kindTransitions, ok := transitions[k]
	if !ok {
		return fmt.Errorf("unknown kind %q", k)
	}
	allowed, knownFrom := kindTransitions[from]
	if !knownFrom {
		return fmt.Errorf("status %q is not a recognized %s state", from, k)
	}
	for _, candidate := range allowed {
		if candidate == to {
			return nil
		}
	}
	if len(allowed) == 0 {
		return fmt.Errorf("%s status %q is terminal; cannot transition to %q", k, from, to)
	}
	return fmt.Errorf("%s status %q cannot transition to %q (allowed: %v)", k, from, to, allowed)
}

// CancelTarget returns the kind's terminal-cancel status — the one
// `aiwf cancel` promotes any non-terminal entity to. Used by the cancel
// verb to know which terminal status maps to "discarded": cancelled
// for epic/milestone, rejected for adr/decision/contract, wontfix for
// gap.
func CancelTarget(k Kind) string {
	switch k {
	case KindEpic, KindMilestone:
		return "cancelled"
	case KindADR, KindDecision, KindContract:
		return "rejected"
	case KindGap:
		return "wontfix"
	}
	return ""
}

// acTransitions encodes the per-status FSM for an acceptance criterion.
// `open → met` is the normal completion path; `open → deferred` and
// `open → cancelled` are the two terminal removals. `met → deferred`
// and `met → cancelled` cover scope changes after the AC was already
// done. `deferred` and `cancelled` are terminal.
var acTransitions = map[string][]string{
	"open":      {"met", "deferred", "cancelled"},
	"met":       {"deferred", "cancelled"},
	"deferred":  {},
	"cancelled": {},
}

// IsLegalACTransition reports whether (from, to) is a legal AC status
// transition under the FSM. Self-transitions, unknown `from`, and
// unknown `to` all return false. The verb-projection finding
// `acs-transition` (Step 6) consults this; `--force --reason` (Step 4)
// is what relaxes it.
func IsLegalACTransition(from, to string) bool {
	for _, allowed := range acTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
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
	for _, allowed := range tddPhaseTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
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
		if ac.Status == "open" {
			openACs = append(openACs, ac.ID)
		}
	}
	return len(openACs) == 0, openACs
}
