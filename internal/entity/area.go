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
// filter note all route their membership decision through it. An empty v
// is not valid here (absence is area-required's concern, handled before
// this is called).
func IsValidAreaValue(v string, members []string) bool {
	if v == "" {
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
