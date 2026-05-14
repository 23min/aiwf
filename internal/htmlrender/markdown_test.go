package htmlrender

import (
	"strings"
	"testing"
)

// TestMarkdownToHTML_RenderingShapes pins the load-bearing markdown
// constructs the renderer must emit correctly. Each case documents
// the input source-shape and asserts a structural marker in the
// output HTML — not just substring matches on raw text, which would
// pass even when the wrapping element is wrong.
//
// Pre-G36 the renderer dumped these as escaped raw text; the new
// helper produces real HTML. If goldmark's emit shape ever changes
// (e.g., paragraph wrapping, code-block class names), this test is
// the canary.
func TestMarkdownToHTML_RenderingShapes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		input    string
		contains []string // every fragment must appear in the rendered HTML
	}{
		{
			name:     "paragraph",
			input:    "Hello world.",
			contains: []string{"<p>Hello world.</p>"},
		},
		{
			name:  "fenced code block",
			input: "```go\nfmt.Println(\"hi\")\n```\n",
			contains: []string{
				"<pre>",
				"<code",
				"fmt.Println",
			},
		},
		{
			name:     "inline code",
			input:    "use `aiwf check` to validate",
			contains: []string{"<code>aiwf check</code>"},
		},
		{
			name:     "unordered list",
			input:    "- alpha\n- beta\n- gamma\n",
			contains: []string{"<ul>", "<li>alpha</li>", "<li>beta</li>"},
		},
		{
			name:     "ordered list",
			input:    "1. first\n2. second\n",
			contains: []string{"<ol>", "<li>first</li>", "<li>second</li>"},
		},
		{
			name:     "link",
			input:    "see [the docs](https://example.com)",
			contains: []string{`<a href="https://example.com">the docs</a>`},
		},
		{
			name:     "subheading inside section body",
			input:    "### Subhead\n\nprose under it.\n",
			contains: []string{"<h3>Subhead</h3>", "<p>prose under it.</p>"},
		},
		{
			name:     "emphasis and strong",
			input:    "this *is* **important**",
			contains: []string{"<em>is</em>", "<strong>important</strong>"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := string(markdownToHTML(tc.input))
			for _, want := range tc.contains {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q\n--- got ---\n%s", want, got)
				}
			}
		})
	}
}

// TestMarkdownToHTML_RawHTMLEscaped is the load-bearing XSS guard:
// a body containing literal HTML (notably `<script>`) must NOT pass
// through to the rendered page as live HTML. Bodies are committed
// to git and trusted by humans, but the static-site step refuses
// to upgrade that trust into "browser-executable" — that decision
// belongs to a separate authoring-policy step, not to render.
//
// Goldmark's default config is strict on this front; the test pins
// the property so a future "let's enable HTML pass-through to be
// nicer to power users" change cannot ship without flagging the
// security regression.
func TestMarkdownToHTML_RawHTMLEscaped(t *testing.T) {
	t.Parallel()
	got := string(markdownToHTML("<script>alert('xss')</script>"))
	if strings.Contains(got, "<script>") {
		t.Errorf("raw <script> tag passed through to output:\n%s", got)
	}
	// The escaped form should render as visible text with entity
	// references; either &lt;script&gt; or a comment placeholder
	// is fine. The negative is the load-bearing assertion.
}

// TestMarkdownToHTML_Empty: empty source returns empty HTML, not a
// stray paragraph wrapper or nil-deref. The renderer skips sections
// with empty content via `with` guards, but the helper itself still
// has to be safe to call.
func TestMarkdownToHTML_Empty(t *testing.T) {
	t.Parallel()
	if got := markdownToHTML(""); got != "" {
		t.Errorf("markdownToHTML(\"\") = %q, want empty", got)
	}
}
