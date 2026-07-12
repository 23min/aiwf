package verb

import (
	"path"
	"strings"
)

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
// resolve it itself). A relative destination (`../work/…`, any `../`
// depth) is resolved against linkingFile's own directory; a
// destination already rooted at a known entity directory (`work/…`,
// `docs/adr/…`) is treated as root-relative and compared as-is —
// matching rewidth's existing convention for `work/`-prefixed links.
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
	dir := path.Dir(linkingFile)
	return walkBodyLines(body, func(line string) string {
		return maskCodeSpans(line, func(chunk string) string {
			return rewriteLinkChunk(chunk, dir, moveIndex)
		})
	})
}

// rewriteLinkChunk applies the move-based destination rewrite to
// every in-link-path region of chunk (as split by
// splitLinkPathRegions), leaving prose regions untouched.
func rewriteLinkChunk(chunk, dir string, moveIndex map[string]string) string {
	var out strings.Builder
	out.Grow(len(chunk))
	for _, reg := range splitLinkPathRegions(chunk) {
		if !reg.inLinkPath {
			out.WriteString(reg.text)
			continue
		}
		out.WriteString(rewriteLinkDestination(reg.text, dir, moveIndex))
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
//
// A `#fragment` / `?query` suffix (M-0251) is split off before
// resolution, so it never participates in the move-index lookup, and
// reattached verbatim on a rewrite — an anchored or query-bearing
// entity link survives a move exactly like a bare path link already
// does. A destination whose bare-path portion doesn't match a move is
// returned unchanged, suffix included.
func rewriteLinkDestination(region, dir string, moveIndex map[string]string) string {
	inner := strings.TrimSuffix(strings.TrimPrefix(region, "("), ")")
	if strings.Contains(inner, "://") {
		return region
	}
	bare, suffix := splitDestinationSuffix(inner)
	resolved, rootRelative := resolveLinkDestination(bare, dir)
	to, ok := moveIndex[resolved]
	if !ok {
		return region
	}
	return "(" + newDestination(to, dir, rootRelative) + suffix + ")"
}

// splitDestinationSuffix splits inner into its bare-path portion and
// a trailing `#fragment` / `?query` suffix, if present — query before
// fragment, per the ordering a relative reference uses (RFC 3986
// §4.2), so the first `#` or `?` in inner marks the suffix's start.
// suffix carries its leading `#`/`?` and everything after it
// verbatim, including a combined `?query#fragment`; when inner has
// neither character, suffix is empty.
func splitDestinationSuffix(inner string) (bare, suffix string) {
	idx := strings.IndexAny(inner, "#?")
	if idx < 0 {
		return inner, ""
	}
	return inner[:idx], inner[idx:]
}

// resolveLinkDestination resolves a link destination to a repo-
// relative, forward-slash path comparable against EntityMove.From,
// plus whether the destination was root-relative (rooted at a known
// entity directory, e.g. `work/gaps/...`) as opposed to relative to
// dir (`../work/gaps/...`).
//
// Path arithmetic uses the stdlib "path" package rather than
// "path/filepath": these are markdown-embedded destinations, not
// filesystem paths, so the math is pure forward-slash string
// manipulation regardless of host OS.
func resolveLinkDestination(inner, dir string) (resolved string, rootRelative bool) {
	if isEntityRootRelative(inner) {
		return path.Clean(inner), true
	}
	return path.Clean(path.Join(dir, inner)), false
}

// newDestination renders to (the move's new path) in the same flavor
// as the original destination: unchanged when the original was
// root-relative, or recomputed relative to dir otherwise.
func newDestination(to, dir string, rootRelative bool) string {
	if rootRelative {
		return to
	}
	return relativeFromDir(dir, to)
}

// entityRootPrefixes returns the deduplicated, trailing-slash root
// directories under which entity files live, derived from
// activeKindLayouts (rewidth.go) so the root-relative set has a
// single source of truth shared with the rewidth verb.
func entityRootPrefixes() []string {
	seen := make(map[string]bool)
	var out []string
	for _, l := range activeKindLayouts() {
		prefix := l.rootDir + "/"
		if seen[prefix] {
			continue
		}
		seen[prefix] = true
		out = append(out, prefix)
	}
	return out
}

// isEntityRootRelative reports whether inner is rooted at one of the
// known entity directories (`work/gaps/...`, `docs/adr/...`, etc.),
// as opposed to a `../`-relative or same-directory destination.
func isEntityRootRelative(inner string) bool {
	for _, p := range entityRootPrefixes() {
		if strings.HasPrefix(inner, p) {
			return true
		}
	}
	return false
}

// relativeFromDir returns the forward-slash relative path from dir to
// target, both already-clean repo-relative paths (dir "." means the
// repo root). Pure segment arithmetic rather than filepath.Rel, for
// the same reason as resolveLinkDestination: these are markdown path
// strings, not filesystem paths.
func relativeFromDir(dir, target string) string {
	dirParts := pathSegments(dir)
	targetParts := pathSegments(target)

	common := 0
	for common < len(dirParts) && common < len(targetParts) && dirParts[common] == targetParts[common] {
		common++
	}

	segs := make([]string, 0, (len(dirParts)-common)+(len(targetParts)-common))
	for i := common; i < len(dirParts); i++ {
		segs = append(segs, "..")
	}
	segs = append(segs, targetParts[common:]...)
	if len(segs) == 0 {
		return "."
	}
	return strings.Join(segs, "/")
}

// pathSegments splits a clean repo-relative forward-slash path into
// its components. "" and "." (the repo root) split to nil.
func pathSegments(p string) []string {
	if p == "" || p == "." {
		return nil
	}
	return strings.Split(strings.Trim(p, "/"), "/")
}
