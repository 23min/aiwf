package gitops

import "strings"

// coauthorTrailerKey is git's conventional co-author trailer, as the
// convention spells it. Matching against it is case-insensitive.
const coauthorTrailerKey = "Co-Authored-By"

// RefusedCoauthorIn returns the first address a `Co-Authored-By:` trailer in
// block names that refused holds, or "" when none does. Addresses in refused
// are expected lowercased; coauthorAddress lowercases what it reads.
//
// The key is matched case-insensitively, unlike the `aiwf-*` keys elsewhere:
// git compares trailer keys without regard to case, and both `Co-Authored-By`
// and `Co-authored-by` are spellings its tooling produces, so both name this
// trailer and a case-sensitive match would wave one of them through.
func RefusedCoauthorIn(block string, refused map[string]bool) string {
	for _, tr := range ParseTrailers(block) {
		if !strings.EqualFold(tr.Key, coauthorTrailerKey) {
			continue
		}
		if addr := coauthorAddress(tr.Value); refused[addr] {
			return addr
		}
	}
	return ""
}

// coauthorAddress returns the address a `Co-Authored-By:` value names,
// lowercased for comparison. The conventional form is `Name <address>`; a
// value carrying no angle-bracketed address is taken whole, which is what lets
// a configured address match a trailer written without a display name.
func coauthorAddress(v string) string {
	v = strings.TrimSpace(v)
	if open := strings.LastIndexByte(v, '<'); open >= 0 {
		if closeAt := strings.IndexByte(v[open+1:], '>'); closeAt >= 0 {
			return strings.ToLower(strings.TrimSpace(v[open+1 : open+1+closeAt]))
		}
	}
	return strings.ToLower(v)
}
