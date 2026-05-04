package entity

import (
	"fmt"
	"strconv"
	"strings"
)

// canonicalPad is the minimum digit count for each kind's id format
// (matches IDFormat: E-NN, M-NNN, ADR-NNNN, G-NNN, D-NNN, C-NNN).
// Numbers exceeding 10^pad expand naturally; the pad is a *minimum*,
// not a maximum.
var canonicalPad = map[Kind]int{
	KindEpic:      2,
	KindMilestone: 3,
	KindADR:       4,
	KindGap:       3,
	KindDecision:  3,
	KindContract:  3,
}

// idPrefix is the literal prefix every id of each kind starts with.
var idPrefix = map[Kind]string{
	KindEpic:      "E-",
	KindMilestone: "M-",
	KindADR:       "ADR-",
	KindGap:       "G-",
	KindDecision:  "D-",
	KindContract:  "C-",
}

// AllocateID picks the next free id for the kind, scanning the union
// of (a) entities — the caller's working tree — and (b) trunkIDs —
// id strings already present in the configured trunk ref's tree.
// Computes max(parsed-id over both sources) + 1 and formats with the
// canonical pad width. trunkIDs may be nil when no trunk is in scope
// (e.g., a sandbox repo with no remotes); see package trunk for the
// policy that produces the slice.
//
// ids in either source whose prefix does not match k are ignored
// (cheap-and-correct: they parse to 0); ids that match the prefix
// but fail strconv are also ignored. The bookkeeping error is
// surfaced by the frontmatter-shape and id-path-consistent checks;
// the allocator does not need to refuse to start.
//
// Branch-to-branch collisions that survive both sources (two
// branches from the same trunk SHA both allocating the same id
// before either lands on trunk) are caught at merge time by the
// ids-unique check, which also reads the trunk ref, and resolved by
// `aiwf reallocate`.
func AllocateID(k Kind, entities []*Entity, trunkIDs []string) string {
	highest := 0
	for _, e := range entities {
		if e.Kind != k {
			continue
		}
		n := parseIDNumber(k, e.ID)
		if n > highest {
			highest = n
		}
	}
	for _, id := range trunkIDs {
		n := parseIDNumber(k, id)
		if n > highest {
			highest = n
		}
	}
	next := highest + 1
	pad, ok := canonicalPad[k]
	if !ok {
		return fmt.Sprintf("%s%d", idPrefix[k], next)
	}
	return fmt.Sprintf("%s%0*d", idPrefix[k], pad, next)
}

// parseIDNumber returns the numeric portion of an id whose prefix
// matches the kind. Returns 0 for non-matching or unparseable ids;
// 0 is safe as a "lowest possible value" sentinel because the
// allocator's first allocation always lands at 1.
func parseIDNumber(k Kind, id string) int {
	prefix := idPrefix[k]
	if !strings.HasPrefix(id, prefix) {
		return 0
	}
	n, err := strconv.Atoi(id[len(prefix):])
	if err != nil {
		return 0
	}
	return n
}
