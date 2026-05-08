package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"strings"
)

// hintTableEntryPattern matches a quoted key in the hint table
// (e.g. `"refs-resolve/wrong-kind":`). We accept a simple shape:
// a "..."-quoted string followed by optional whitespace and a
// colon at the start of a line (allowing leading tabs/spaces).
var hintTableEntryPattern = regexp.MustCompile(`(?m)^\s*"([a-z][a-z0-9_-]*(?:/[a-z0-9_-]+)?)"\s*:`)

// PolicyFindingCodesHaveHints asserts that every Code: literal
// used in a Finding{} composite-literal across the codebase has a
// matching entry in internal/check/hint.go's hintTable. A
// finding without a hint produces a "what now?" gap for the user —
// the renderer expects every code to resolve.
//
// Subcodes are honored: `Code: "refs-resolve" Subcode: "unresolved"`
// matches `refs-resolve/unresolved` in the hint table; bare codes
// match the unsuffixed key.
//
// Codes whose value comes from a constant (`Code: CodeProvenance...`)
// are resolved by reading the constant's literal value from the
// declaring file. The walk is bounded to the check package's
// constants; the codebase happens to declare every finding-code
// constant there.
func PolicyFindingCodesHaveHints(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}

	hintKeys := loadHintTableKeys(files)
	codeConsts := loadCheckCodeConstants(files)

	var out []Violation
	fset := token.NewFileSet()
	type seenCode struct {
		Code    string
		Subcode string
		File    string
		Line    int
	}
	var seen []seenCode

	for _, f := range files {
		if !strings.HasSuffix(f.Path, ".go") || strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			// Look at fields named Code / Subcode in the composite
			// literal — works for any struct that has those fields,
			// including check.Finding and any pseudo-finding type.
			var code, subcode string
			for _, elt := range cl.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				ident, ok := kv.Key.(*ast.Ident)
				if !ok {
					continue
				}
				switch ident.Name {
				case "Code":
					code = resolveStringExpr(kv.Value, codeConsts)
				case "Subcode":
					subcode = resolveStringExpr(kv.Value, codeConsts)
				}
			}
			if code == "" {
				return true
			}
			seen = append(seen, seenCode{
				Code:    code,
				Subcode: subcode,
				File:    f.Path,
				Line:    fset.Position(cl.Pos()).Line,
			})
			return true
		})
	}

	for _, sc := range seen {
		key := sc.Code
		if sc.Subcode != "" {
			key = sc.Code + "/" + sc.Subcode
		}
		if _, ok := hintKeys[key]; ok {
			continue
		}
		// Subcode-less fallback: hint table sometimes carries a bare
		// key for a code with multiple subcodes. The runtime HintFor
		// falls back to the bare code when the subcoded entry is
		// missing — mirror that behavior to avoid false positives.
		if sc.Subcode != "" {
			if _, ok := hintKeys[sc.Code]; ok {
				continue
			}
		}
		out = append(out, Violation{
			Policy: "finding-codes-have-hints",
			File:   sc.File,
			Line:   sc.Line,
			Detail: "Finding{Code: " + strconv.Quote(sc.Code) +
				func() string {
					if sc.Subcode != "" {
						return ", Subcode: " + strconv.Quote(sc.Subcode)
					}
					return ""
				}() +
				"} has no entry in internal/check/hint.go hintTable",
		})
	}
	return out, nil
}

// loadHintTableKeys reads internal/check/hint.go and returns
// the set of keys defined in the hintTable map literal. Falls back
// to an empty map if the file isn't found.
func loadHintTableKeys(files []FileEntry) map[string]struct{} {
	keys := map[string]struct{}{}
	for _, f := range files {
		if f.Path != "internal/check/hint.go" {
			continue
		}
		matches := hintTableEntryPattern.FindAllSubmatch(f.Contents, -1)
		for _, m := range matches {
			if len(m) < 2 {
				continue
			}
			keys[string(m[1])] = struct{}{}
		}
		break
	}
	return keys
}

// loadCheckCodeConstants reads every const declaration in the
// check package and returns name → string-literal value for the
// ones whose value is a quoted string. Used to resolve
// `Code: CodeFoo` references in Finding{} literals.
func loadCheckCodeConstants(files []FileEntry) map[string]string {
	out := map[string]string{}
	fset := token.NewFileSet()
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "internal/check/") {
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
					if i >= len(vs.Values) {
						continue
					}
					lit, ok := vs.Values[i].(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					unq, err := strconv.Unquote(lit.Value)
					if err != nil {
						continue
					}
					out[name.Name] = unq
				}
			}
		}
	}
	return out
}

// resolveStringExpr extracts a string value from an AST expression
// when possible. Handles bare string literals and identifiers that
// resolve via codeConsts. Returns "" otherwise — callers treat
// unresolved values as opaque and skip the policy check (we can't
// fairly judge a code we can't read).
func resolveStringExpr(expr ast.Expr, codeConsts map[string]string) string {
	switch v := expr.(type) {
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return ""
		}
		s, err := strconv.Unquote(v.Value)
		if err != nil {
			return ""
		}
		return s
	case *ast.Ident:
		return codeConsts[v.Name]
	}
	return ""
}
