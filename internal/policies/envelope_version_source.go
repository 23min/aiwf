package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// PolicyEnvelopeVersionSource asserts that every keyed `Version:` in a
// composite literal written as `Envelope` or `pkg.Envelope` takes its
// value from a function call's `.Version` — in practice
// version.Current().Version, the single canonical reader for the
// running binary's version.
//
// A verb that sets its envelope's Version from another source can
// report a different string from a verb that reads
// version.Current(), so a consumer running the two in sequence can
// see two versions for one binary.
//
// The rule scans every production .go file outside internal/policies
// (_test.go is excluded: test fixtures set literal versions) for
// composite literals whose type is named `Envelope`, bare or
// package-qualified; a literal whose type is left out (an element
// written inside `[]Envelope{...}`) is not checked. It accepts only a
// `.Version` selector on a function call, without checking that the
// call is version.Current(), and rejects every other value,
// `info.Version` included. The chokepoint is intentionally narrow: it pins
// the envelope construction site, not every version-reading code
// path in the codebase.
func PolicyEnvelopeVersionSource(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			comp, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if !isEnvelopeType(comp.Type) {
				return true
			}
			for _, elt := range comp.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok || key.Name != "Version" {
					continue
				}
				if isAllowedVersionSource(kv.Value) {
					continue
				}
				out = append(out, Violation{
					Policy: "envelope-version-source",
					File:   f.Path,
					Line:   fset.Position(kv.Pos()).Line,
					Detail: "JSON envelope Version must be a function call's .Version, in practice version.Current().Version; a source other than version.Current() can report a different version string from the verbs that use it",
				})
			}
			return true
		})
	}
	return out, nil
}

// isEnvelopeType reports whether the composite-literal type is named
// `Envelope`, either bare (same-package) or as a `pkg.Envelope`
// selector.
func isEnvelopeType(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name == "Envelope"
	case *ast.SelectorExpr:
		return t.Sel.Name == "Envelope"
	}
	return false
}

// isAllowedVersionSource reports whether the value passed to a
// JSON envelope's Version field comes from an allowed source. The
// canonical pattern is `version.Current().Version` — a SelectorExpr
// whose RHS is "Version" applied to a CallExpr whose selector is
// "Current". Any other shape that resolves to a function call's
// `.Version` field is also accepted (covers `someResolver().Version`
// patterns a future verb might introduce). Bare Idents and bare
// SelectorExprs (which would be package-global access) are not.
func isAllowedVersionSource(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		// A non-selector value (literal string, bare ident, etc) is
		// not allowed. Strings are forbidden because the policy's
		// whole point is to route through the canonical reader.
		return false
	}
	if sel.Sel.Name != "Version" {
		return false
	}
	// X must be a CallExpr (function/method call returning a value
	// whose Version field we read). This excludes bare package-global
	// access like `cli.Version` (where X is *ast.Ident).
	if _, isCall := sel.X.(*ast.CallExpr); isCall {
		return true
	}
	return false
}
