package verb

import "strings"

// EntityMove is one entity's old→new repo-relative path, forward-slash,
// as planned by a file-moving verb (archive, rename, retitle,
// reallocate). RewriteLinkDestinations rewrites markdown link
// destinations that resolve to Move.From, pointing them at Move.To.
type EntityMove struct {
	From string
	To   string
}

// RewriteLinkDestinations rewrites, within body, every markdown link
// destination that resolves to one of moves' From paths, pointing it
// at the matching To path. linkingFile is the repo-relative,
// forward-slash path of the file body belongs to (M-0246/M-0247
// callers pass the file's own path; the shared primitive doesn't
// resolve it itself).
//
// Everything else is left byte-identical: prose, inline-code spans,
// fenced code blocks, URL-shaped destinations, and links whose
// destination does not resolve to a moved entity. Pure — no I/O — and
// idempotent: a destination already rewritten to a To path is not a
// From path in the same move set, so a second pass is a no-op.
//
// Masking (fence detection, inline-code-span exclusion, link-path
// region splitting) is shared with rewidth's width-rewrite via
// walkBodyLines / maskCodeSpans / splitLinkPathRegions in
// linkregion.go; only the destination-rewrite predicate below is
// specific to this primitive.
func RewriteLinkDestinations(body []byte, linkingFile string, moves []EntityMove) []byte {
	moveIndex := make(map[string]string, len(moves))
	for _, m := range moves {
		moveIndex[m.From] = m.To
	}
	return walkBodyLines(body, func(line string) string {
		return maskCodeSpans(line, func(chunk string) string {
			return rewriteLinkChunk(chunk, moveIndex)
		})
	})
}

// rewriteLinkChunk applies the move-based destination rewrite to
// every in-link-path region of chunk (as split by
// splitLinkPathRegions), leaving prose regions untouched.
func rewriteLinkChunk(chunk string, moveIndex map[string]string) string {
	var out strings.Builder
	out.Grow(len(chunk))
	for _, reg := range splitLinkPathRegions(chunk) {
		if !reg.inLinkPath {
			out.WriteString(reg.text)
			continue
		}
		out.WriteString(rewriteLinkDestination(reg.text, moveIndex))
	}
	return out.String()
}

// rewriteLinkDestination rewrites a single link-path region (the
// `(...)` destination, parens included) when its destination resolves
// to a moved entity; otherwise returns region unchanged.
//
// URL-shaped destinations (containing `://`) are never rewritten — an
// id in a URL's path is part of the URL's identity, not an entity
// reference.
func rewriteLinkDestination(region string, moveIndex map[string]string) string {
	inner := strings.TrimSuffix(strings.TrimPrefix(region, "("), ")")
	if strings.Contains(inner, "://") {
		return region
	}
	to, ok := moveIndex[inner]
	if !ok {
		return region
	}
	return "(" + to + ")"
}
