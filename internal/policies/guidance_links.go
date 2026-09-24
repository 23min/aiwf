package policies

import (
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"github.com/23min/aiwf/internal/check"
)

// guidanceMarkdown parses guidance documents for the references they make.
var guidanceMarkdown = goldmark.New()

// markdownLinks returns the destinations of a document's links as CommonMark
// reads them: inline, reference-style, with or without a title, in angle
// brackets. Images and autolinks are not references to read.
func markdownLinks(doc string) []string {
	src := []byte(doc)
	var out []string
	root := guidanceMarkdown.Parser().Parse(text.NewReader(src))
	// The walker never returns an error: the callback has no error path.
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if link, ok := n.(*ast.Link); ok && entering {
			out = append(out, string(link.Destination))
		}
		return ast.WalkContinue, nil
	})
	return out
}

// importToken matches an `@path` import in prose: an at-sign opening a
// whitespace-delimited token, so an address such as a@b is not one.
var importToken = regexp.MustCompile(`(?:^|[\s(])@([^\s)]+)`)

// markdownImports returns the `@path` imports in a document's prose, where
// Claude Code expands them: inline or on a line of their own, but not inside
// a code span or block. Trailing sentence punctuation is not part of the
// path.
func markdownImports(doc string) []string {
	var out []string
	for _, m := range importToken.FindAllStringSubmatch(check.ProseMask([]byte(doc)), -1) {
		if target := strings.TrimRight(m[1], ".,;:"); target != "" {
			out = append(out, target)
		}
	}
	return out
}

// resolveReference turns a reference made from the document at doc into a
// repository path. A leading slash is the repository root, as it is on the
// forge; anything else resolves against the document's directory. The
// anchor and query are dropped and percent-escapes decoded. A reference
// with a scheme, a bare anchor, or one leaving the repository is not a
// repository path.
func resolveReference(doc, target string) (string, bool) {
	target, _, _ = strings.Cut(target, "#")
	target, _, _ = strings.Cut(target, "?")
	if target == "" || strings.Contains(target, ":") {
		return "", false
	}
	if decoded, err := url.PathUnescape(target); err == nil {
		target = decoded
	}
	var p string
	if rooted, ok := strings.CutPrefix(target, "/"); ok {
		p = path.Clean(rooted)
	} else {
		p = path.Clean(path.Join(path.Dir(doc), target))
	}
	if p == "." || p == ".." || strings.HasPrefix(p, "../") {
		return "", false
	}
	return p, true
}
