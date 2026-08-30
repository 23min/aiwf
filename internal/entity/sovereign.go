package entity

// SovereignActShape names a (kind, from, to) tuple that is FSM-legal
// but treated by the kernel as requiring an explicit sovereign-act
// gesture: a `human/` actor. `--force --reason "..."` does not stand in
// for one — it relaxes this gate rather than the FSM (these tuples are
// FSM-legal, and a human reaches them with no flag at all), and a force
// trailer from a non-human actor is itself refused at the apply seam,
// so both routes into these transitions require a human. Exported so
// policy-layer code (which lives outside the entity package) can build
// per-entry regexes or other derived structures against the closed set.
type SovereignActShape struct {
	Kind Kind
	From Status
	To   Status
}

// sovereignActShapes is the closed-set list of transitions the kernel
// treats as sovereign-act-shape. Each entry's authorizing artifact is
// cited in the comment beside it.
//
// New entries land when an ADR or kernel-spec ratifies a transition
// as sovereign-act-shape. The list is consulted by:
//
//   - `requireHumanActorForSovereignAct` (internal/verb/
//     promote_sovereign_act.go) — runtime verb gate, refuses a non-
//     human actor before anything is written. Called from both
//     `Promote` and `Cancel`, since a transition is named by the
//     state it reaches rather than by the verb that reaches it.
//   - `forcedUntraileredFindings` (internal/check/
//     fsm_history_consistent.go, M-0130/AC-3) — historical audit,
//     emits the `fsm-history-consistent/forced-untrailered` subcode
//     when a sovereign-act-shape commit lacks the `aiwf-force`
//     trailer.
//   - `auditUnforcedSovereignActPromote` (internal/policies/
//     aiwf_promote_epic_active_audit.go) — static CI/script audit,
//     derives its patterns from `entity.SovereignActShapes()` so
//     adding a new entry here automatically widens the audit's reach.
//
// D-0008 promises a closed-set invariant: every entry here must be a
// legal FSM transition (sovereign-act-shape is a property *over* legal
// transitions, never below them). The invariant is pinned by
// `TestSovereignActShapes_AllFSMLegal` in sovereign_test.go.
var sovereignActShapes = []SovereignActShape{
	// epic proposed → active. Authorized by M-0095 (motivated by
	// G-0063). M-0095's spec body frames other kinds' activation /
	// acceptance edges as a "separate open question, deferred at
	// planning time" — they remain open candidates pending their
	// own authorizing ADRs.
	{KindEpic, StatusProposed, StatusActive},
	// epic active → done. Authorized by ADR-0047, which rules every
	// edge into a terminal epic status sovereign, arguing from
	// irreversibility rather than effort: `done` has no outgoing
	// edges, and the kernel answers an unwanted terminal status with
	// a new entity rather than a transition back. Milestones share
	// the terminal shape and are deliberately excluded — an epic is
	// the unit a scope is opened on, a milestone is work inside a
	// scope someone already holds.
	{KindEpic, StatusActive, StatusDone},
	// epic → cancelled, from either state it is reachable from. Same
	// authorization (ADR-0047), which declines to treat cancelling a
	// proposed epic as the lesser act: discarding a plan that never
	// became work invites that reading, but `cancelled` is terminal
	// whatever state it is reached from, so the two are equally
	// unrecoverable. Both are reached by `aiwf cancel` rather than by
	// promote, which is why that verb consults this set too.
	{KindEpic, StatusActive, StatusCancelled},
	{KindEpic, StatusProposed, StatusCancelled},
}

// IsSovereignActShape reports whether (k, from, to) names a transition
// the kernel treats as sovereign-act-shape — set-membership only, no
// FSM-legality check. Callers that need to distinguish "legal but
// sovereign" from "illegal" call ValidateTransition separately; the
// `fsm-history-consistent` check rule (M-0130) layers the two checks
// to produce its disjoint `illegal-transition` and `forced-untrailered`
// subcodes per D-0008.
//
// Returns false for unknown kinds, unknown statuses, and any tuple not
// in the sovereignActShapes closed set.
func IsSovereignActShape(k Kind, from, to Status) bool {
	for _, s := range sovereignActShapes {
		if s.Kind == k && s.From == from && s.To == to {
			return true
		}
	}
	return false
}

// SovereignActShapes returns a defensive copy of the kernel's closed-
// set sovereign-act-shape list. Callers iterate to build derived
// structures (regex lists, doc tables, drift checks) without exposing
// the package-level slice to mutation.
func SovereignActShapes() []SovereignActShape {
	out := make([]SovereignActShape, len(sovereignActShapes))
	copy(out, sovereignActShapes)
	return out
}
