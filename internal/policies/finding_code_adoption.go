package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
)

// PolicyFindingCodeAdoption asserts that finding-code string
// constants declared in internal/check/*.go are used at both emit
// sites (struct-literal `Code:` keyed-field-value) and comparison
// sites (`Code == "..."` / `switch Code`) rather than their literal
// values. Concretely: `Finding{Code: "acs-shape", ...}` violates the
// rule; `Finding{Code: check.CodeACsShape, ...}` satisfies it.
//
// Why this exists (G-0129): without a mechanical chokepoint for
// finding codes specifically, future contributors can re-introduce
// emit-vs-test drift past CI by adding a new bare-string emit site or
// renaming a typed code without updating the literal-string mirror in
// a test. The provenance.go family was already typed; this extends the
// closure to the remaining ~25 codes across acs.go, archive_rules.go,
// check.go, entity_body.go, entity_id_narrow_width.go,
// epic_active_drafts.go, fsm_history_consistent.go,
// fsm_history_walker.go, and tree_discipline.go.
//
// Scope:
//   - All .go files (production AND tests), per the gap's point 4 —
//     the emit-vs-test drift is one of the main wins; excluding tests
//     would leave the closure asymmetric.
//   - Outside internal/check/ itself (the package owns the constant
//     definitions).
//   - Outside internal/policies/ (this file's docstring and the
//     synthetic-input test fixtures carry code literals as content,
//     not as comparison-site drift).
//   - Three detection sites: keyed-field-value `Code: "..."` in
//     struct literals (emit chokepoint), `==`/`!=` BinaryExpr against
//     a literal code (comparison drift), switch/case literal clauses
//     (switch drift).
//
// Allowlist: `//enums:ignore <reason>` line-suffix comments on the
// violating line suppress the finding. Same shape as the Status
// policy in enum_literal_adoption.go.
func PolicyFindingCodeAdoption(root string) ([]Violation, error) {
	consts, err := enumerateCheckFindingCodeConstants(root)
	if err != nil {
		return nil, err
	}
	const excludeTests = false
	files, err := WalkGoFiles(root, excludeTests)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		fileRel := filepath.ToSlash(f.Path)
		// Skip the check package itself (declares the constants).
		if strings.HasPrefix(fileRel, "internal/check/") {
			continue
		}
		// Skip the policies package: own fixtures + this file's
		// docstring carry code literals as content.
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
			case *ast.KeyValueExpr:
				// Emit-site chokepoint: `Code: "..."` in a struct
				// literal. The key must be the identifier `Code`
				// (case-sensitive — matches the Finding struct's
				// field name) and the value must be a string literal.
				ident, ok := node.Key.(*ast.Ident)
				if !ok || ident.Name != "Code" {
					return true
				}
				if v := stringLiteralValue(node.Value); v != "" {
					reportIfFindingCode(&out, fset, f.Path, node.Value.Pos(), v, consts, ignored)
				}
			case *ast.BinaryExpr:
				if node.Op != token.EQL && node.Op != token.NEQ {
					return true
				}
				if v := stringLiteralValue(node.X); v != "" {
					reportIfFindingCode(&out, fset, f.Path, node.X.Pos(), v, consts, ignored)
				}
				if v := stringLiteralValue(node.Y); v != "" {
					reportIfFindingCode(&out, fset, f.Path, node.Y.Pos(), v, consts, ignored)
				}
			case *ast.CaseClause:
				for _, expr := range node.List {
					if v := stringLiteralValue(expr); v != "" {
						reportIfFindingCode(&out, fset, f.Path, expr.Pos(), v, consts, ignored)
					}
				}
			}
			return true
		})
	}
	return out, nil
}

// enumerateCheckFindingCodeConstants reads every .go file under
// internal/check/ (excluding _test.go) and returns a map from string-
// literal value (e.g., "acs-shape") to constant identifier (e.g.,
// "CodeACsShape"). Only top-level constants whose name begins with
// "Code" are considered.
//
// Done at policy-run time so adding a new finding-code constant
// auto-extends the rule with no second source of truth.
func enumerateCheckFindingCodeConstants(root string) (map[string]string, error) {
	out := map[string]string{}
	const excludeTests = true
	files, err := WalkGoFiles(root, excludeTests)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	for _, f := range files {
		fileRel := filepath.ToSlash(f.Path)
		if !strings.HasPrefix(fileRel, "internal/check/") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		for _, decl := range astFile.Decls {
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
					if !strings.HasPrefix(name.Name, "Code") {
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
	}
	return out, nil
}

// reportIfFindingCode appends a Violation to out when value is a known
// finding-code literal AND the source line at pos is not on the
// ignored list.
func reportIfFindingCode(
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
		Policy: "finding-code-adoption",
		File:   relPath,
		Line:   line,
		Detail: "string literal " + strconv.Quote(value) +
			" used as finding code; use check." + name + " instead. " +
			"Suppress with `//enums:ignore <reason>` if the literal is intentional.",
	})
}
