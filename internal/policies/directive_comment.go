package policies

import "strings"

// hasDirectiveComment reports whether a raw comment is the named escape
// directive carrying a non-empty reason. It is the shared matcher behind
// //history:ok and //exec:ok, so the two conventions cannot drift apart.
//
// The marker must open the comment, directive-style (`//<marker> why`).
// Matching it anywhere in the text would let prose that merely mentions the
// escape — including these policies' own doc comments — silently exempt a
// neighbouring line, which is the one thing an escape hatch on a blocking
// gate must not do.
//
// Whitespace must separate the marker from the reason, so a longer word
// opening with the marker's letters (`//exec:okay`) reads as a different
// comment rather than as the directive with "ay" for a reason. A bare marker
// is not an escape either: the reason is the point.
//
// The `//` prefix is required, and callers pass a whole comment rather than
// one source line of one, so a marker inside a /* */ block is text rather
// than a directive. Go reads //go:build the same way: a build constraint
// written inside a block comment does not apply.
func hasDirectiveComment(raw, marker string) bool {
	rest, found := strings.CutPrefix(raw, "//"+marker)
	if !found {
		return false
	}
	reason := strings.TrimLeft(rest, " \t")
	if len(reason) == len(rest) {
		// Nothing separates the marker from what follows: either the
		// comment is the bare marker, or the marker is a prefix of a
		// longer word. Neither is an escape.
		return false
	}
	return strings.TrimSpace(reason) != ""
}
