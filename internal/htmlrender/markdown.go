package htmlrender

import (
	"bytes"
	"html/template"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// markdownEngine is the package-wide goldmark renderer used to turn
// body-section markdown into HTML. Configured once at init so every
// per-section call uses the same parser options:
//
//   - GFM extensions: tables, strikethrough, autolink, task list.
//   - Hard-wrap OFF: `\n` doesn't become `<br>`. Markdown semantics
//     dictate that prose paragraphs join lines; planning bodies are
//     authored that way already.
//   - Raw HTML pass-through OFF: a body containing `<script>` (or
//     any other tag) renders as escaped text, not as HTML. Bodies
//     are committed to git but the renderer is *not* the place that
//     decides to trust them as HTML — that decision belongs to the
//     authoring policy, not the static-site step.
//   - Heading auto-IDs OFF: we don't link into body headings yet,
//     and turning auto-IDs on would create stable anchors that the
//     user could come to depend on; leave it off until a real
//     in-page deep-link surface lands.
var markdownEngine = goldmark.New(
	goldmark.WithExtensions(
		extension.Table,
		extension.Strikethrough,
		extension.Linkify,
		extension.TaskList,
	),
	goldmark.WithParserOptions(
		parser.WithAttribute(),
	),
	goldmark.WithRendererOptions(
		html.WithXHTML(),
	),
)

// markdownToHTML renders a markdown source string to HTML, returning
// a template.HTML so html/template emits the bytes verbatim instead
// of double-escaping them. Empty input returns empty HTML.
//
// Goldmark errors (extremely rare for in-memory inputs) degrade
// gracefully: the original source is escaped via template.HTMLEscaper
// and returned, so a parse failure produces ugly-but-safe output
// rather than a broken page.
//
// G36 fix.
func markdownToHTML(src string) template.HTML {
	if src == "" {
		return ""
	}
	var buf bytes.Buffer
	if err := markdownEngine.Convert([]byte(src), &buf); err != nil { //coverage:ignore goldmark.Convert never returns a non-nil error for in-memory []byte input
		// nolint:gosec // G203: input is escaped first via HTMLEscapeString; the cast wraps already-safe content.
		return template.HTML(template.HTMLEscapeString(src))
	}
	// nolint:gosec // G203: goldmark renderer is configured with raw-HTML pass-through OFF and Linkify/Table/Strikethrough/TaskList only; output is sanitized by construction (see markdownEngine config and TestMarkdownToHTML_RawHTMLEscaped).
	return template.HTML(buf.String())
}
