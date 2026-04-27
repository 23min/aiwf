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

// AllocateID picks the next free id for the kind, given the entities
// currently in the tree. Computes max(existing ids of that kind) + 1
// and formats with the canonical pad width. The allocator only sees
// the caller's tree; cross-branch coordination is by design out of
// scope (collisions are caught by the ids-unique check and resolved
// with `aiwf reallocate`).
//
// Entities whose ID does not match the kind's expected pattern are
// ignored when computing max — the bookkeeping error is surfaced by
// the frontmatter-shape check; the allocator does not need to refuse
// to start.
func AllocateID(k Kind, entities []*Entity) string {
	max := 0
	for _, e := range entities {
		if e.Kind != k {
			continue
		}
		n := parseIDNumber(k, e.ID)
		if n > max {
			max = n
		}
	}
	next := max + 1
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
