package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"strings"
)

// trailerLineStart matches the start of a synthesized commit-message
// trailer line: `aiwf-<name>: ` at the beginning of a format string.
var trailerLineStart = regexp.MustCompile(`^\s*aiwf-[a-z][a-z0-9-]*:\s`)

// PolicyNoTrailerStringComposition forbids fmt format strings that
// look like they are SYNTHESIZING a commit-message trailer line —
// where the format BEGINS with `aiwf-<name>: ` (the literal
// trailer-line shape). Trailer values must be assembled via
// gitops.Trailer{Key: ..., Value: ...} struct literals so the
// per-trailer write-time validators (gitops.ValidateTrailer) can
// run; a Sprintf'd trailer block bypasses that check.
//
// Diagnostic / error messages that name a trailer in passing
// (e.g. "commit %s: aiwf-actor: %q must match …") are legitimate
// and not flagged — the trailer name appears mid-string for
// human readability, not as the start of a synthesized line.
//
// Test files are exempt (tests synthesize commit messages with
// inline trailer text). The policies package is excluded by
// WalkGoFiles already.
func PolicyNoTrailerStringComposition(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		// Skip the gitops package — it owns trailer construction
		// helpers and CommitMessage which legitimately formats
		// trailer lines.
		if strings.HasPrefix(f.Path, "internal/gitops/") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			// Only check fmt.Sprintf / fmt.Errorf / fmt.Fprintf calls.
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "fmt" {
				return true
			}
			switch sel.Sel.Name {
			case "Sprintf", "Errorf", "Fprintf", "Printf":
			default:
				return true
			}
			// First arg is the format string. Check that it doesn't
			// embed an aiwf-* trailer-name token.
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			val, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			// Only flag formats that BEGIN with the trailer-line
			// shape `aiwf-<name>: ` — these look like synthesized
			// commit-message trailer lines. Diagnostic messages
			// that mention a trailer name mid-string are legitimate.
			if !trailerLineStart.MatchString(val) {
				return true
			}
			out = append(out, Violation{
				Policy: "no-trailer-string-composition",
				File:   f.Path,
				Line:   fset.Position(call.Pos()).Line,
				Detail: "fmt." + sel.Sel.Name + " format embeds an aiwf-* trailer name; assemble trailers via gitops.Trailer{} so ValidateTrailer can check shape",
			})
			return true
		})
	}
	return out, nil
}
