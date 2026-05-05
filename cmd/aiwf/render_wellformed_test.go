package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestRender_AllPagesAreWellFormed exercises every HTML file in a
// realistic fixture render (epic + milestone + ACs + phase
// history + authorize scope) and asserts each parses with no
// errors via the local well-formedness checker.
//
// "Well-formed" here is a narrower property than full HTML5
// validation: every opened tag is closed in the right order, no
// unclosed `<a>` / `<section>` / `<table>` / `<ul>` / `<ol>` /
// `<li>` / `<tr>` / `<td>` / `<th>` / `<thead>` / `<tbody>` /
// `<p>` / `<span>` / `<code>` / `<nav>` / `<main>` / `<h1>` /
// `<h2>` / `<h3>`. Catches the regression class where a template
// loses a `</section>` and the page still kind of renders but
// every browser falls back to error-recovery heuristics that
// produce different DOMs.
func TestRender_AllPagesAreWellFormed(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test")
	mustRun(t, "add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "epic", "--title", "Adoption", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "Schema parser", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "Tree loader", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "ac", "--root", root, "--actor", "human/test", "M-001", "--title", "Parses YAML")
	mustRun(t, "add", "ac", "--root", root, "--actor", "human/test", "M-001", "--title", "Reports errors")
	mustRun(t, "promote", "--root", root, "--actor", "human/test", "M-001/AC-1", "met")
	mustRun(t, "promote", "--root", root, "--actor", "human/test", "M-001/AC-2", "--phase", "red")
	mustRun(t, "promote", "--root", root, "--actor", "human/test", "M-001/AC-2", "--phase", "green",
		"--tests", "pass=8 fail=0 skip=1")
	mustRun(t, "promote", "--root", root, "--actor", "human/test", "M-001/AC-2", "--phase", "done")
	mustRun(t, "promote", "--root", root, "--actor", "human/test", "M-001", "in_progress")
	mustRun(t, "authorize", "--root", root, "--actor", "human/test", "M-002", "--to", "ai/claude")

	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)

	for _, name := range []string{"index.html", "E-01.html", "E-02.html", "M-001.html", "M-002.html"} {
		t.Run(name, func(t *testing.T) {
			body := readFileT(t, filepath.Join(out, name))
			if err := assertWellFormed(body); err != nil {
				t.Errorf("%s: %v", name, err)
			}
		})
	}
}

// assertWellFormed walks the input and verifies that every opening
// tag (in trackedTags) is matched by a closing tag in LIFO order.
// Self-closing void elements (meta, link, br, hr, etc.) are
// skipped, as are HTML comments and the doctype declaration.
//
// Returns a non-nil error naming the offending tag and position
// when the document is malformed; nil when balanced.
func assertWellFormed(html string) error {
	tracked := map[string]bool{
		"a": true, "section": true, "table": true, "tbody": true, "thead": true,
		"tr": true, "td": true, "th": true, "ul": true, "ol": true, "li": true,
		"p": true, "span": true, "code": true, "nav": true, "main": true,
		"h1": true, "h2": true, "h3": true, "html": true, "body": true,
		"head": true, "title": true,
	}
	var stack []string
	cursor := 0
	for cursor < len(html) {
		open := strings.IndexByte(html[cursor:], '<')
		if open < 0 {
			break
		}
		open += cursor
		// Skip comments.
		if strings.HasPrefix(html[open:], "<!--") {
			closeC := strings.Index(html[open:], "-->")
			if closeC < 0 {
				return errAt("unterminated comment", open)
			}
			cursor = open + closeC + len("-->")
			continue
		}
		// Skip doctype.
		if strings.HasPrefix(strings.ToLower(html[open:]), "<!doctype") {
			gt := strings.IndexByte(html[open:], '>')
			if gt < 0 {
				return errAt("unterminated doctype", open)
			}
			cursor = open + gt + 1
			continue
		}
		gt := strings.IndexByte(html[open:], '>')
		if gt < 0 {
			return errAt("unterminated tag", open)
		}
		tag := html[open+1 : open+gt]
		cursor = open + gt + 1
		closing := strings.HasPrefix(tag, "/")
		if closing {
			tag = tag[1:]
		}
		// Strip attributes.
		if i := strings.IndexByte(tag, ' '); i >= 0 {
			tag = tag[:i]
		}
		tag = strings.ToLower(strings.TrimSuffix(tag, "/"))
		if !tracked[tag] {
			continue
		}
		if closing {
			if len(stack) == 0 {
				return errAt("close </"+tag+"> with empty stack", open)
			}
			top := stack[len(stack)-1]
			if top != tag {
				return errAt("close </"+tag+"> when top of stack is <"+top+">", open)
			}
			stack = stack[:len(stack)-1]
			continue
		}
		stack = append(stack, tag)
	}
	if len(stack) > 0 {
		return errAt("unclosed tags at EOF: "+strings.Join(stack, ", "), len(html))
	}
	return nil
}

// errAt is a small helper that formats a position-tagged error so
// failures are easier to triage.
func errAt(msg string, pos int) error {
	return &wellFormedError{msg: msg, pos: pos}
}

type wellFormedError struct {
	msg string
	pos int
}

func (e *wellFormedError) Error() string {
	return e.msg + " (at byte " + itoa(e.pos) + ")"
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
