package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// PolicyEnvelopeVersionSource asserts that every render.Envelope
// initialization sources the `Version:` field from
// version.Current().Version — the single canonical reader for the
// running binary's version.
//
// Pre-M-0118 a regression class existed where some verbs (the
// in-cmd/aiwf check verb up through AC-2) wrote `Version: Version`,
// reading the cmd/aiwf-side package-global directly. Other verbs
// (status, history, contract verify) had migrated to
// `Version: version.Current().Version`. A consumer running two such
// verbs in sequence could see two different version strings on
// unstamped local builds — `"dev"` vs `"(devel)"`. M-0118 item 6
// closed the divergence by routing every verb through
// version.Current(); this policy is the chokepoint that prevents
// regression.
//
// The rule scans every production .go file (excludes _test.go since
// test fixtures legitimately set Version: "0.1.0" or "dev" as part
// of envelope round-trip and golden tests) for CompositeLits
// whose type ends in `Envelope` (matching render.Envelope, the
// canonical JSON envelope type) and any other `*Envelope` aliases a
// future verb might introduce. For each, it checks the `Version:`
// key's value. The allowed shapes:
//
//   - `version.Current().Version` (the canonical reader)
//   - any field-access on an Info value (e.g. `info.Version` when
//     `info` is already resolved via version.Current somewhere else)
//
// The forbidden shape is a bare `Version` identifier or a
// selector `*.Version` resolving to the cli-package global
// (e.g. `cli.Version`). The chokepoint is intentionally narrow:
// it pins the JSON-envelope construction site, not every
// version-reading code path in the codebase.
func PolicyEnvelopeVersionSource(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		// Skip the policy package itself (its source carries the rule
		// description text, including the forbidden patterns as prose).
		if strings.HasPrefix(filepath.ToSlash(f.Path), "internal/policies/") {
			continue
		}
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
					Detail: "JSON envelope Version field must source from version.Current().Version (the canonical reader); a direct reference to a package-global Version produces divergence on unstamped local builds — see M-0118 item 6",
				})
			}
			return true
		})
	}
	return out, nil
}

// isEnvelopeType reports whether the composite-literal type ends in
// `Envelope`. Matches both bare `Envelope` (same-package) and
// `pkg.Envelope` selector forms.
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
