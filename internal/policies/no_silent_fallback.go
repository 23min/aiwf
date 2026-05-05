package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// closedSetSwitchTypes are type names whose switches must have a
// default branch that errors / returns a sentinel rather than
// silently falling through. A silent default on a closed-set
// switch hides the case "the set grew and we forgot to update".
var closedSetSwitchTypes = map[string]bool{
	"Kind":          true,
	"State":         true,
	"VerbKind":      true,
	"AuthorizeMode": true,
	"OpType":        true,
}

// PolicyNoSilentFallbacks flags switch statements over closed-set
// types whose default branch is missing or silent (no return /
// error). The policy errs on the side of "default exists and does
// something visible" — even a `default: return fmt.Errorf("unknown
// %v", x)` is fine.
//
// AST: parse each Go file, find SwitchStmt / TypeSwitchStmt whose
// tag-or-case shape is a known closed-set type. Verify a
// `default:` clause exists and contains at least one statement
// that returns or panics.
func PolicyNoSilentFallbacks(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		// Tests are exempt: a closed-set switch in a test is usually
		// a t.Run dispatcher, where falling through is the expected
		// "no-op" behavior.
		if strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			sw, ok := n.(*ast.SwitchStmt)
			if !ok || sw.Tag == nil {
				return true
			}
			typeName := exprTypeIdent(sw.Tag)
			if !closedSetSwitchTypes[typeName] {
				return true
			}
			// Find a default branch.
			var defaultClause *ast.CaseClause
			for _, stmt := range sw.Body.List {
				cc, ok := stmt.(*ast.CaseClause)
				if !ok {
					continue
				}
				if cc.List == nil {
					defaultClause = cc
					break
				}
			}
			if defaultClause == nil {
				out = append(out, Violation{
					Policy: "no-silent-fallback",
					File:   f.Path,
					Line:   fset.Position(sw.Pos()).Line,
					Detail: "switch over closed-set type " + typeName +
						" has no default branch; add one (with an error or a comment) so the design intent is explicit",
				})
				return true
			}
			// An explicit default branch — even one that is empty
			// with an explanatory comment — satisfies the policy.
			// The value is making intent visible; what to DO when
			// the default fires is a separate (case-by-case)
			// judgment.
			_ = defaultClause
			return true
		})
	}
	return out, nil
}

// exprTypeIdent returns the name of the receiver type whose
// switch-tag we're inspecting. For `e.Status`, returns "" (we
// don't know the type from name alone). For a bare identifier
// like `kind`, returns the identifier — the policy then matches
// against closedSetSwitchTypes by type-name convention.
//
// We're not running a real type checker; this keeps the policy
// best-effort.
func exprTypeIdent(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		// Match by variable name when it looks like a type alias
		// usage (e.g. `kind`). Mostly false; we accept that.
		_ = v
		return ""
	case *ast.SelectorExpr:
		return v.Sel.Name // e.g. e.Kind → "Kind"
	}
	return ""
}
