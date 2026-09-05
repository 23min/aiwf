package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// sectionScanPackages are the packages that decide, about an entity
// body, whether a section is there. Nothing else in the module reads an
// entity body for its `##` headings: the roadmap renderer and the
// policy suite scan ROADMAP.md and CLAUDE.md, which are documents no
// entity rule judges.
var sectionScanPackages = []string{
	filepath.Join("internal", "check"),
	filepath.Join("internal", "verb"),
}

// sectionParserFile is the one file allowed to walk a body's lines
// looking for section headings. Every section question in the packages
// above resolves to what it returns, which is what keeps two rules from
// answering differently about the same body.
var sectionParserFile = "internal/entity/body.go"

// sectionScanExempt names the functions whose `##` test asks something
// other than which sections a body has, each of which reads a section
// the caller has already located:
//
//   - isAllWhitespaceOrHeadings classifies content as written or not.
//   - scanACBodies uses the heading as a terminator, bounding where one
//     AC's body stops.
var sectionScanExempt = map[string]bool{
	"isAllWhitespaceOrHeadings": true,
	"scanACBodies":              true,
}

// prefixTests are the calls a hand-rolled heading scan is written from.
var prefixTests = map[string]bool{
	"HasPrefix": true, "TrimPrefix": true, "CutPrefix": true,
}

// PolicySectionAbsenceSinglePredicate asserts that no package deciding
// whether an entity-body section is present carries a heading scan of
// its own.
//
// Two rules ask that question — the write-time guards in internal/verb
// and milestone-done-empty-release-note in internal/check — and they
// once asked it two ways. One matched a regexp tolerant of any
// whitespace after `##`, the other the literal `"## "`. Measured on
// `##\tGoal`: the first reported the section present and the second
// reported it absent, about the same body. A rule keyed on that answer
// is only as trustworthy as there being one of it.
//
// The ban is on the construct rather than on a call graph, because a
// heading scan is unmistakable in source: a `##` prefix test, or a
// regexp anchored on one. A new rule that rolls its own is caught where
// it is written rather than when someone notices the two disagree.
// Building on entity.ParseBodySections instead costs nothing and is
// what every other caller already does.
//
// It judges the use, not the literal. `"## " + name` renders a heading
// into an operator-facing message and decides nothing, and `^###` is
// the AC-heading pattern, which asks which criteria a milestone body
// carries rather than which sections.
func PolicySectionAbsenceSinglePredicate(root string) ([]Violation, error) {
	var vs []Violation
	for _, pkg := range sectionScanPackages {
		dir := filepath.Join(root, pkg)
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		fset := token.NewFileSet()
		for _, de := range entries {
			if de.IsDir() || !strings.HasSuffix(de.Name(), ".go") || strings.HasSuffix(de.Name(), "_test.go") {
				continue
			}
			f, err := parser.ParseFile(fset, filepath.Join(dir, de.Name()), nil, 0)
			if err != nil {
				return nil, err
			}
			rel := filepath.ToSlash(filepath.Join(pkg, de.Name()))
			vs = append(vs, headingScansIn(f, fset, rel)...)
		}
	}
	return vs, nil
}

// headingScansIn reports every heading scan in one parsed file.
func headingScansIn(f *ast.File, fset *token.FileSet, rel string) []Violation {
	var vs []Violation
	for _, decl := range f.Decls {
		fn, isFunc := decl.(*ast.FuncDecl)
		if isFunc && sectionScanExempt[fn.Name.Name] {
			continue
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !isHeadingScanCall(call) {
				return true
			}
			vs = append(vs, Violation{
				Policy: "section-absence-single-predicate",
				File:   rel,
				Line:   fset.Position(call.Pos()).Line,
				Detail: "this package decides whether an entity-body section is present, so it may carry no `##` heading scan of its own — two scans answer differently about the same body (measured: a tab after the hashes). Ask entity.ParseBodySections, which " + sectionParserFile + " owns, or check.SectionsAbsent for the absence question itself",
			})
			return true
		})
	}
	return vs
}

// isHeadingScanCall reports whether call tests a line against a `## `
// heading or compiles a regexp anchored on one.
func isHeadingScanCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, isIdent := sel.X.(*ast.Ident)
	if !isIdent {
		return false
	}
	for _, arg := range call.Args {
		lit := stringLiteral(arg)
		if lit == "" {
			continue
		}
		if prefixTests[sel.Sel.Name] && (pkg.Name == "strings" || pkg.Name == "bytes") && strings.HasPrefix(lit, "## ") {
			return true
		}
		if pkg.Name == "regexp" && strings.HasPrefix(lit, "^##") && !strings.HasPrefix(lit, "^###") {
			return true
		}
	}
	return false
}

// stringLiteral unwraps arg to its string value, seeing through the
// []byte(...) conversion a bytes.HasPrefix call wraps its literal in.
func stringLiteral(arg ast.Expr) string {
	if conv, ok := arg.(*ast.CallExpr); ok && len(conv.Args) == 1 {
		arg = conv.Args[0]
	}
	lit, ok := arg.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	if strings.HasPrefix(lit.Value, "`") {
		return strings.Trim(lit.Value, "`")
	}
	v, err := strconv.Unquote(lit.Value)
	if err != nil { //coverage:ignore unreachable: the literal reached here came from parser.ParseFile, which rejects a malformed quoted string before this policy sees it
		return ""
	}
	return v
}
