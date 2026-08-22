package policies

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// PolicyViolationPolicyIDLiteral requires every Violation's `Policy:`
// field to be set from a string literal.
//
// This is what makes PolicyFiringFixturePresence's inventory complete.
// That gate finds a policy's firing sites by matching the source line
// `Policy: "<kebab-id>"`, so a policy that sets the field from a
// constant or a variable is absent from the inventory: nothing then
// proves it can fire, and a refactor that turns it into a no-op leaves
// CI green — the failure G-0259 exists to catch, one level up.
//
// The rule is a ban rather than a mandate, and that is the point: it
// costs once, at the moment someone writes a non-literal, instead of
// charging every policy for a proof that it is visible. It also retires
// the alternative, which would be teaching the inventory scan to resolve
// constants — a parsed-AST evaluator carrying its own blind spots.
//
// Scope is the production sources under internal/policies, matching the
// inventory scan it protects. A test file is out of scope: its
// Violations are fixtures, not firing sites.
// The directory is read directly rather than through WalkGoFiles, which
// skips internal/policies by construction — the same reason
// constructionSites reads it directly.
func PolicyViolationPolicyIDLiteral(root string) ([]Violation, error) {
	dir := filepath.Join(root, "internal", "policies")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, perr := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if perr != nil {
			return nil, fmt.Errorf("parsing %s: %w", name, perr) //coverage:ignore a production source that does not parse fails the build long before any policy runs.
		}
		out = append(out, nonLiteralPolicyIDs(fset, file, "internal/policies/"+name)...)
	}
	return out, nil
}

// nonLiteralPolicyIDs is the pure core: it walks one parsed file for
// composite literals of type Violation and reports each `Policy:` field
// whose value is not a string literal.
func nonLiteralPolicyIDs(fset *token.FileSet, file *ast.File, relPath string) []Violation {
	var out []Violation
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok || !isViolationType(lit.Type) {
			return true
		}
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "Policy" {
				continue
			}
			if isStringLit(kv.Value) {
				continue
			}
			out = append(out, Violation{
				Policy: "violation-policy-id-literal",
				File:   relPath,
				Detail: fmt.Sprintf("line %d: the Violation's Policy field is not a string literal, so the firing-fixture inventory cannot see this policy and nothing proves it can fire; spell the id inline (`Policy: \"my-policy-id\"`).",
					fset.Position(kv.Value.Pos()).Line),
			})
		}
		return true
	})
	return out
}

// isViolationType reports whether a composite literal's type names
// Violation, either bare (in-package) or qualified (policies.Violation).
func isViolationType(t ast.Expr) bool {
	switch v := t.(type) {
	case *ast.Ident:
		return v.Name == "Violation"
	case *ast.SelectorExpr:
		return v.Sel.Name == "Violation"
	}
	return false
}

// isStringLit reports whether e is a plain string literal. A concatenation
// is not one: its operands can be constants, which is the case this ban
// exists to refuse.
func isStringLit(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.STRING && !strings.HasPrefix(lit.Value, "`")
}
