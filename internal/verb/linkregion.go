package verb

import "strings"

// walkBodyLines splits body into lines, leaves fenced code blocks
// (```...```) untouched, and passes every non-fenced line through
// rewriteLine. Shared by rewidth's width-rewrite and the move-based
// link-destination rewrite (M-0245) — fence detection is identical
// for both.
func walkBodyLines(body []byte, rewriteLine func(line string) string) []byte {
	src := string(body)
	var out strings.Builder
	out.Grow(len(src))

	lines := strings.Split(src, "\n")
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			out.WriteString(line)
			if i < len(lines)-1 {
				out.WriteByte('\n')
			}
			inFence = !inFence
			continue
		}
		if inFence {
			out.WriteString(line)
			if i < len(lines)-1 {
				out.WriteByte('\n')
			}
			continue
		}
		out.WriteString(rewriteLine(line))
		if i < len(lines)-1 {
			out.WriteByte('\n')
		}
	}
	return []byte(out.String())
}

// maskCodeSpans walks line honoring inline-code-span (`text`)
// boundaries: content between backticks passes through verbatim,
// content outside is handed to rewriteChunk. Shared by rewidth's
// width-rewrite and the move-based link-destination rewrite.
//
// An unterminated span on a line (an unmatched opening backtick) is
// treated as in-span (verbatim) per markdown convention: an unmatched
// backtick does not open a real code span, but treating the tail as
// prose to rewrite risks mangling a genuine typo'd id mention, so the
// conservative choice is to leave it untouched.
func maskCodeSpans(line string, rewriteChunk func(chunk string) string) string {
	var out strings.Builder
	out.Grow(len(line))
	inSpan := false
	var buf strings.Builder
	flushOutside := func() {
		if buf.Len() == 0 {
			return
		}
		out.WriteString(rewriteChunk(buf.String()))
		buf.Reset()
	}
	for _, r := range line {
		if r == '`' {
			if inSpan {
				// Closing backtick — flush the in-span buffer verbatim.
				out.WriteString(buf.String())
				buf.Reset()
				out.WriteRune(r)
				inSpan = false
				continue
			}
			// Opening backtick — flush the out-of-span buffer with
			// rewriting applied.
			flushOutside()
			out.WriteRune(r)
			inSpan = true
			continue
		}
		buf.WriteRune(r)
	}
	if inSpan {
		out.WriteString(buf.String())
	} else {
		flushOutside()
	}
	return out.String()
}

// linkPathRegion is a contiguous run inside or outside a markdown
// link-path `](...)` literal. inLinkPath=true regions include the
// surrounding `(` and `)` so callers can match against the
// destination directly; outside regions are pure prose.
type linkPathRegion struct {
	text       string
	inLinkPath bool
}

// splitLinkPathRegions walks s and splits it into alternating
// in-link-path and outside-link-path regions. A link-path region
// starts at `](` (immediately after the `]`) and ends at the matching
// `)`. Nesting and escapes are not handled — markdown's link-path
// grammar disallows unescaped `)` inside the path, and the callers'
// inputs don't include escaped link paths.
func splitLinkPathRegions(s string) []linkPathRegion {
	var out []linkPathRegion
	var buf strings.Builder
	i := 0
	for i < len(s) {
		// Look for `](` starting at i.
		idx := strings.Index(s[i:], "](")
		if idx < 0 {
			buf.WriteString(s[i:])
			break
		}
		abs := i + idx
		// Everything up to (but not including) `]` goes into the
		// outside region. We also include the `]` itself in outside,
		// since it's not part of the link-path region.
		buf.WriteString(s[i : abs+1])
		out = append(out, linkPathRegion{text: buf.String(), inLinkPath: false})
		buf.Reset()
		// Now find the matching `)`. Start at the `(` immediately
		// after `]`.
		closeRel := strings.Index(s[abs+2:], ")")
		if closeRel < 0 {
			// Unbalanced — treat the rest of the string as outside,
			// per "conservative: don't rewrite" approach for malformed
			// markdown.
			out = append(out, linkPathRegion{text: s[abs+1:], inLinkPath: false})
			break
		}
		closeAbs := abs + 2 + closeRel
		// link-path region includes `(` and `)`.
		out = append(out, linkPathRegion{text: s[abs+1 : closeAbs+1], inLinkPath: true})
		i = closeAbs + 1
	}
	if buf.Len() > 0 {
		out = append(out, linkPathRegion{text: buf.String(), inLinkPath: false})
	}
	return out
}
