package entity

// SovereignActShape names a (kind, from, to) tuple that is FSM-legal
// but treated by the kernel as requiring an explicit sovereign-act
// gesture — a `human/` actor by default, or `--force --reason "..."`
// from a non-human actor. Exported so policy-layer code (which lives
// outside the entity package) can build per-entry regexes or other
// derived structures against the closed set.
type SovereignActShape struct {
	Kind Kind
	From string
	To   string
}

// sovereignActShapes is the closed-set list of transitions the kernel
// treats as sovereign-act-shape. Each entry's authorizing artifact is
// cited in the comment beside it.
//
// New entries land when an ADR or kernel-spec ratifies a transition
// as sovereign-act-shape. The list is consulted by:
//
//   - `requireHumanActorForSovereignAct` (internal/verb/
//     promote_sovereign_act.go) — runtime verb gate, refuses non-
//     human actors at promote time.
//   - `forcedUntraileredFindings` (internal/check/
//     fsm_history_consistent.go, M-0130/AC-3) — historical audit,
//     emits the `fsm-history-consistent/forced-untrailered` subcode
//     when a sovereign-act-shape commit lacks the `aiwf-force`
//     trailer.
//   - `auditUnforcedEpicActivate` (internal/policies/
//     aiwf_promote_epic_active_audit.go) — static CI/script audit,
//     builds one regex per entry via `entity.SovereignActShapes()` so
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
func IsSovereignActShape(k Kind, from, to string) bool {
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
