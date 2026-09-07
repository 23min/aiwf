package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// sectionScanPackages are the packages whose `##` scans decide whether a
// verb refuses or a rule reports. They are what this ban covers.
//
// They are not the only code reading an entity body for headings:
// internal/roadmap's extractSection reads an epic file to render its
// `## Goal`, and disagrees with the parser on a heading spelled with two
// spaces. That decides render output rather than a refusal, so it is out
// of scope here — but the module does not have one section scanner, it
// has two, and the second is unbanned.
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
// other than which sections a body has. scanACBodies uses the heading as
// a terminator, bounding where one AC's body stops, so the prescribed
// remedy — ask the parser which sections there are — does not serve it.
//
// One entry, and it earns its place: deleting it makes the policy fire.
// An entry that changes no verdict is worse than absent, because it
// still excuses the function it names if that function is ever rewritten
// into a real scan.
var sectionScanExempt = map[string]bool{
	"scanACBodies": true,
}

// prefixTests match a line against the start of a heading; searchTests
// look for one anywhere in a body. A hand-rolled heading scan is written
// from one family or the other.
var (
	prefixTests = map[string]bool{"HasPrefix": true, "TrimPrefix": true, "CutPrefix": true}
	searchTests = map[string]bool{
		"Contains": true, "Index": true, "LastIndex": true, "Count": true,
		"Split": true, "SplitSeq": true, "SplitN": true, "SplitAfter": true, "Cut": true,
		"Equal": true, "EqualFold": true,
	}
)

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
			vs = append(vs, headingScansIn(f, fset, rel, stringConsts(f))...)
		}
	}
	return vs, nil
}

// stringConsts maps a file's package-level string constants and
// variables to their values, so a scan built from an extracted literal
// is classified by what the literal says rather than by its name.
func stringConsts(f *ast.File) map[string]string {
	out := map[string]string{}
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || (gen.Tok != token.CONST && gen.Tok != token.VAR) {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok { //coverage:ignore unreachable: go/ast produces only ValueSpec under a const or var GenDecl, and gen.Tok is filtered to those above
				continue
			}
			for i, name := range vs.Names {
				if i < len(vs.Values) {
					if lit := stringLiteral(vs.Values[i], nil); lit != "" {
						out[name.Name] = lit
					}
				}
			}
		}
	}
	return out
}

// headingScansIn reports every heading scan in one parsed file.
func headingScansIn(f *ast.File, fset *token.FileSet, rel string, consts map[string]string) []Violation {
	var vs []Violation
	for _, decl := range f.Decls {
		fn, isFunc := decl.(*ast.FuncDecl)
		if isFunc && sectionScanExempt[fn.Name.Name] {
			continue
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !isHeadingScanCall(call, consts) {
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

// isHeadingScanCall reports whether call tests a line against a `##`
// heading or compiles a regexp anchored on one.
func isHeadingScanCall(call *ast.CallExpr, consts map[string]string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, isIdent := sel.X.(*ast.Ident)
	if !isIdent {
		return false
	}
	strFunc := pkg.Name == "strings" || pkg.Name == "bytes"
	for _, arg := range call.Args {
		lit := stringLiteral(arg, consts)
		if lit == "" {
			continue
		}
		// A bare `##` counts: a scan more tolerant than the parser is the
		// drift this exists to catch, not a lesser case of it.
		if strFunc && (prefixTests[sel.Sel.Name] || searchTests[sel.Sel.Name]) && isHeadingLiteral(lit, prefixTests[sel.Sel.Name]) {
			return true
		}
		if pkg.Name == "regexp" && isHeadingPattern(lit) {
			return true
		}
	}
	return false
}

// isHeadingLiteral reports whether a string a call tests a line against
// names a `##` section heading. A prefix test must open with it; a search
// need only contain it.
//
// `###` is a different question — which acceptance criteria a milestone
// body carries, not which sections — and it is exempt in whichever of the
// two spellings an author reaches for, here and in isHeadingPattern.
func isHeadingLiteral(lit string, prefix bool) bool {
	if prefix {
		return strings.HasPrefix(lit, "##") && !strings.HasPrefix(lit, "###")
	}
	return strings.Contains(lit, "##") && !strings.Contains(lit, "###")
}

// isHeadingPattern reports whether a regexp source is anchored on a `##`
// heading, after stripping what stands between `^` and the hashes: inline
// flag groups, and leading-whitespace tolerance. Both are how a scan gets
// written that is looser than the parser, which is the drift this catches
// rather than a lesser case of it.
func isHeadingPattern(pattern string) bool {
	anchored := patternPreamble.ReplaceAllString(pattern, "^")
	return strings.HasPrefix(anchored, "^##") && !strings.HasPrefix(anchored, "^###")
}

// patternPreamble matches a regexp source's opening inline flag groups
// and its caret together with any leading-whitespace tolerance after it,
// replacing the lot with a bare caret.
var patternPreamble = regexp.MustCompile(`^(?:\(\?[^)]*\))*\^(?:\\s\*|\\s\+|[ \t]\*|[ \t]\+|\[ \\t\]\*)?`)

// stringLiteral unwraps arg to its string value, seeing through the
// []byte(...) conversion a bytes.HasPrefix call wraps its literal in and
// through a name bound to a string constant in the same file.
func stringLiteral(arg ast.Expr, consts map[string]string) string {
	if conv, ok := arg.(*ast.CallExpr); ok && len(conv.Args) == 1 {
		arg = conv.Args[0]
	}
	if ident, ok := arg.(*ast.Ident); ok {
		return consts[ident.Name]
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
