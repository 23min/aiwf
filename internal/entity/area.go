package entity

// AreaGlobal is the reserved area value for inherently-cross-cutting
// entities (ADRs, decisions, seam contracts) — the affirmative, never-
// inferred not-1:1 escape valve (ADR-0021, E-0044 / M-0184). It is one
// value of the single-valued area dimension, not a second axis; a
// declared areas.members entry may not be named it.
const AreaGlobal = "global"

// IsValidAreaValue reports whether v is an acceptable non-empty area tag
// given the declared member names: the reserved AreaGlobal sentinel, or
// any declared member. It is the single definition of "valid area value"
// (ADR-0021, M-0184) — area-unknown, set-area, add --area, and the read-
// filter note all route their membership decision through it.
//
// With no declared members the area dimension is inert (M-0171), so
// NOTHING is valid — not even the reserved global sentinel (Position A,
// M-0184): `global` is feature-gated and unavailable until an areas block
// exists. This gate lives in the predicate itself, the SSOT, rather than
// depending on each caller's pre-guard (the callers keep their guards for
// clearer messages; this is the correctness backstop).
//
// An empty v is not valid here (absence is area-required's concern,
// handled before this is called).
func IsValidAreaValue(v string, members []string) bool {
	if v == "" {
		return false
	}
	if len(members) == 0 {
		// Area dimension inert until an areas block is declared (M-0171);
		// global included (Position A, M-0184).
		return false
	}
	if v == AreaGlobal {
		return true
	}
	for _, m := range members {
		if m == v {
			return true
		}
	}
	return false
}
