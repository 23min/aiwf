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
