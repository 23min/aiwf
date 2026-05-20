package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
)

// PolicyEnumLiteralAdoption asserts that closed-set status string
// constants declared in internal/entity/entity.go are used at
// comparison sites rather than their literal values. Concretely:
// `ac.Status == "open"` violates the rule; `ac.Status == entity.StatusOpen`
// satisfies it.
//
// Why this exists (G-0126): without a mechanical chokepoint, future
// contributors can re-introduce literal-vs-constant drift past CI.
// Review caught one such drift through the M-0118-era refactor and
// missed three others; the kernel rule "framework correctness must
// not depend on LLM behavior" forbids relying on reviewer recall for
// a check this mechanical.
//
// Scope:
//   - Production .go files only (test fixtures legitimately string-
//     literal-compare status values for parser-tolerance assertions).
//   - Outside internal/entity/ itself (the package owns the constant
//     definitions; comparing a constant against its own literal is a
//     non-violation by construction).
//   - Comparison sites only: `==` / `!=` BinaryExpr and switch/case
//     literal clauses. Raw assignments (`s := "open"`) are not
//     flagged — they're often test data or YAML decoding scaffolding.
//
// Allowlist: `//enums:ignore <reason>` line-suffix comments on the
// violating line suppress the finding. Matches `//coverage:ignore`
// convention. The reason is required prose so the suppression carries
// audit context.
func PolicyEnumLiteralAdoption(root string) ([]Violation, error) {
	consts, err := enumerateEntityStatusConstants(root)
	if err != nil {
		return nil, err
	}
	const excludeTests = true
	files, err := WalkGoFiles(root, excludeTests)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		fileRel := filepath.ToSlash(f.Path)
		// Skip the entity package itself.
		if strings.HasPrefix(fileRel, "internal/entity/") {
			continue
		}
		// Skip the policies package: this file's docstring and
		// the synthetic-input test fixtures both carry status
		// literals as content, not as comparison-site drift.
		if strings.HasPrefix(fileRel, "internal/policies/") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.ParseComments)
		if perr != nil {
			continue
		}
		ignored := collectIgnoredLines(astFile, fset)
		ast.Inspect(astFile, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.BinaryExpr:
				if node.Op != token.EQL && node.Op != token.NEQ {
					return true
				}
				if v := stringLiteralValue(node.X); v != "" {
					reportIfStatus(&out, fset, f.Path, node.X.Pos(), v, consts, ignored)
				}
				if v := stringLiteralValue(node.Y); v != "" {
					reportIfStatus(&out, fset, f.Path, node.Y.Pos(), v, consts, ignored)
				}
			case *ast.CaseClause:
				for _, expr := range node.List {
					if v := stringLiteralValue(expr); v != "" {
						reportIfStatus(&out, fset, f.Path, expr.Pos(), v, consts, ignored)
					}
				}
			}
			return true
		})
	}
	return out, nil
}

// enumerateEntityStatusConstants reads internal/entity/entity.go and
// returns a map from string-literal value to constant identifier
// name. Only top-level constants whose name begins with "Status" are
// considered — the seed denylist per the M-0119 spec; expansion to
// `Kind*`, `Phase*`, etc. is a deliberate future-gap call.
//
// Done at policy-run time so adding a new status auto-extends the
// rule with no second source of truth.
func enumerateEntityStatusConstants(root string) (map[string]string, error) {
	path := filepath.Join(root, "internal", "entity", "entity.go")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if !strings.HasPrefix(name.Name, "Status") {
					continue
				}
				if i >= len(vs.Values) {
					continue
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				v, err := strconv.Unquote(lit.Value)
				if err != nil {
					continue
				}
				out[v] = name.Name
			}
		}
	}
	return out, nil
}

// stringLiteralValue returns the unquoted string value of expr when
// expr is an *ast.BasicLit of token.STRING kind; otherwise "".
func stringLiteralValue(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	v, err := strconv.Unquote(lit.Value)
	if err != nil {
		return ""
	}
	return v
}

// reportIfStatus appends a Violation to out when value is a known
// status literal AND the source line at pos is not on the ignored
// list.
func reportIfStatus(
	out *[]Violation,
	fset *token.FileSet,
	relPath string,
	pos token.Pos,
	value string,
	consts map[string]string,
	ignored map[int]bool,
) {
	name, ok := consts[value]
	if !ok {
		return
	}
	line := fset.Position(pos).Line
	if ignored[line] {
		return
	}
	*out = append(*out, Violation{
		Policy: "enum-literal-adoption",
		File:   relPath,
		Line:   line,
		Detail: "string literal " + strconv.Quote(value) +
			" used at comparison site; use entity." + name + " instead. " +
			"Suppress with `//enums:ignore <reason>` if the literal is intentional.",
	})
}

// collectIgnoredLines returns the set of line numbers carrying a
// `//enums:ignore <reason>` comment. The comment is line-suffix
// style (same as `//coverage:ignore`) — the line it suppresses is
// the line the comment is on (a Go end-of-line comment binds to the
// preceding source code line via Position).
func collectIgnoredLines(f *ast.File, fset *token.FileSet) map[int]bool {
	out := map[int]bool{}
	for _, group := range f.Comments {
		for _, c := range group.List {
			text := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
			if strings.HasPrefix(text, "enums:ignore") {
				out[fset.Position(c.Pos()).Line] = true
			}
		}
	}
	return out
}
