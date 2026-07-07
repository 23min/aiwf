package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// PolicyLoggingChokepoint asserts that production code does not call a
// bare stdio print function. ADR-0017 AC-3's discipline: every
// operator-facing text output routes through the cliutil text-output
// wrapper set (Errorf/Errorln/Printf/Println/Print) or OutputFormat's
// own text-mode branch, so a stray fmt.Println / fmt.Fprintln(os.Stderr,
// …) can't reintroduce an unaccounted-for output path. This is the
// independent AST-walking backstop for the .golangci.yml forbidigo
// rule — the discipline holds even if the linter rule is ever disabled.
//
// Two shapes fire:
//   - A bare fmt.Println/fmt.Print/fmt.Printf call — always writes to
//     stdout, with no legitimate writer-parameterized variant.
//   - An fmt.Fprintln/fmt.Fprintf call whose first argument is the
//     literal os.Stdout or os.Stderr identifier — forbidigo cannot
//     express this (it matches only the callee, never the argument
//     list), so this policy is the sole enforcement for this shape.
//
// Scope is every non-test Go file the repo walk returns (internal/ and
// cmd/; the policies package itself and test files are excluded by
// WalkGoFiles). Legitimate exceptions — the cliutil text-output wrapper
// set's own implementation, and OutputFormat's text-mode branch — are
// allowlisted by file path with a one-line rationale.
//
// Known blind spots, consistent with the repo's other AST policies
// (e.g. atomic_write_chokepoint.go): an aliased fmt import (`f "fmt"`),
// method-value indirection (`p := fmt.Println`), and a writer argument
// that holds os.Stdout/os.Stderr indirectly through a variable are not
// matched.
func PolicyLoggingChokepoint(root string) ([]Violation, error) {
	// File-path allowlist. Key is the repo-relative forward-slash
	// path; value is the rationale (kept here so the exemption and
	// its justification travel together).
	allow := map[string]string{
		"internal/cli/cliutil/outputformat.go": "OutputFormat's own text-mode branch — the sanctioned envelope writer",
		"internal/cli/cliutil/textio.go":       "the cliutil text-output wrapper set's own implementation",
	}
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err //coverage:ignore WalkGoFiles errors only on a filesystem walk failure; not reachable with a valid tree root.
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		if _, ok := allow[f.Path]; ok {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "fmt" {
				return true
			}
			switch sel.Sel.Name {
			case "Println", "Print", "Printf":
				// Always a violation — no destination argument to check.
			case "Fprintln", "Fprintf":
				if len(call.Args) < 1 || !isOSStdioWriter(call.Args[0]) {
					return true
				}
			default:
				return true
			}
			out = append(out, Violation{
				Policy: "logging-chokepoint",
				File:   f.Path,
				Line:   fset.Position(call.Pos()).Line,
				Detail: "fmt." + sel.Sel.Name + " is a bare stdio print; route through cliutil's text-output " +
					"wrapper set (ADR-0017 AC-3) or allowlist the file with a rationale",
			})
			return true
		})
	}
	return out, nil
}

// isOSStdioWriter reports whether e is the literal os.Stdout or
// os.Stderr selector expression.
func isOSStdioWriter(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "os" {
		return false
	}
	return sel.Sel.Name == "Stdout" || sel.Sel.Name == "Stderr"
}
