package entity

import (
	"fmt"
	"strconv"
	"strings"
)

// CanonicalPad is the canonical zero-pad width for every entity
// kind's id format (E-NNNN, M-NNNN, ADR-NNNN, G-NNNN, D-NNNN, C-NNNN).
// Numbers exceeding 10^pad expand naturally; the pad is a *minimum*,
// not a maximum.
//
// Per ADR-0008, the kernel emits a uniform 4-digit width across all
// kinds. Parsers (idPatterns, ParseCompositeID) keep accepting narrower
// legacy widths so pre-migration trees, branches, and commit trailers
// continue to validate without history rewrite. Display surfaces and
// new allocations always emit canonical width via Canonicalize.
const CanonicalPad = 4

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
	prefix, ok := idPrefix[k]
	if !ok {
		return fmt.Sprintf("%s%d", k, next)
	}
	return fmt.Sprintf("%s%0*d", prefix, CanonicalPad, next)
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
