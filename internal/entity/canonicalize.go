package entity

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Canonicalize rewrites a recognizable entity-id string to canonical
// width: the numeric portion is left-zero-padded to CanonicalPad
// digits if it currently uses fewer. Numbers that already meet or
// exceed the canonical width are returned unchanged (so an id like
// `M-12345` keeps its natural width — CanonicalPad is a minimum,
// not a maximum).
//
// Composite ids (`M-NNN/AC-N`) recurse on the parent portion; the
// `AC-N` sub-id is left alone (the AC's id has no minimum-digit
// requirement at the grammar layer; see the docstring on
// compositeIDPattern). An input that does not parse as one of the
// six aiwf id formats — and is not a composite — is returned
// verbatim. Empty input passes through unchanged.
//
// This is the lookup-side complement to AllocateID's canonical
// emission. Callers comparing an externally-supplied id against an
// on-disk id (Tree.ByID, history-trailer matching, render
// canonicalization) run both sides through Canonicalize so a
// pre-migration narrow id (e.g. `E-22`) and a canonical id
// (`E-0022`) are treated as equivalent.
//
// Pure function. No allocations beyond the formatted output.
func Canonicalize(id string) string {
	if id == "" {
		return id
	}
	// Composite ids: recurse on the parent, leave the sub-id alone.
	if parent, sub, ok := ParseCompositeID(id); ok {
		canonParent := Canonicalize(parent)
		if canonParent == parent {
			return id
		}
		return canonParent + "/" + sub
	}
	// Bare ids: split on the kind's literal prefix and re-pad if narrower.
	for k, prefix := range idPrefix {
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		num := id[len(prefix):]
		if num == "" {
			return id
		}
		// Confirm the numeric tail matches the kind's grammar before
		// rewriting; otherwise pass through unchanged so non-id strings
		// that happen to start with `E-` etc. aren't mangled.
		if !idPatterns[k].MatchString(id) {
			return id
		}
		n, err := strconv.Atoi(num)
		if err != nil {
			//coverage:ignore strconv.Atoi only fails on non-digit input;
			// idPatterns[k].MatchString already constrained `num` to
			// `\d{N,}` so this branch is unreachable in practice. Kept
			// as a defensive fallback (returning verbatim) rather than
			// panicking on a future grammar change.
			return id
		}
		if len(num) >= CanonicalPad {
			return id
		}
		return fmt.Sprintf("%s%0*d", prefix, CanonicalPad, n)
	}
	return id
}

// IDGrepAlternation returns a POSIX-extended regex alternation that
// matches both the canonical and narrow legacy widths of id, suitable
// for `git log --grep -E ...^aiwf-entity: <pattern>$` queries that
// must continue to find pre-migration commit trailers (per AC-2 and
// AC-4 in M-081).
//
// For composite ids the parent recurses; the AC-N sub-id is anchored
// verbatim. For unrecognized ids the input is regex-quoted unchanged
// so callers always receive a syntactically-valid pattern.
//
// Concretely, an input of `E-22` returns `(E-0*22)` (any width that
// equals 22 numerically); `E-0022` returns the same. `M-22/AC-1`
// returns `(M-0*22)/AC-1`. The pattern is intended to be embedded in
// a wider regex (anchors, prefix), so it is wrapped in a single
// capture group for unambiguous concatenation.
func IDGrepAlternation(id string) string {
	if id == "" {
		return ""
	}
	if parent, sub, ok := ParseCompositeID(id); ok {
		return IDGrepAlternation(parent) + "/" + regexp.QuoteMeta(sub)
	}
	for k, prefix := range idPrefix {
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		num := id[len(prefix):]
		if num == "" {
			break
		}
		if !idPatterns[k].MatchString(id) {
			break
		}
		// Strip leading zeros to get the canonical numeric value, then
		// emit a pattern that matches any zero-padded form of it.
		// `0*` accepts both narrow (`E-22`) and canonical (`E-0022`)
		// trailers. The bare numeric portion is regex-quoted defensively
		// even though we know it's digits.
		trimmed := strings.TrimLeft(num, "0")
		if trimmed == "" {
			trimmed = "0"
		}
		return "(" + regexp.QuoteMeta(prefix) + "0*" + regexp.QuoteMeta(trimmed) + ")"
	}
	return regexp.QuoteMeta(id)
}
