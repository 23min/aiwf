package policies

import (
	"regexp"
	"strings"

	"github.com/23min/aiwf/internal/check"
)

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
		out = append(out, strings.TrimRight(m[1], ".,;:"))
	}
	return out
}
