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

	var out []Violation
	for _, sc := range emittedFindingCodeSites(files) {
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

// findingCodeSite is one finding code observed at a Finding{}/pseudo-
// finding composite-literal construction site, with its optional subcode,
// the severity it emits, and its source location.
type findingCodeSite struct {
	Code    string
	Subcode string
	File    string
	Line    int
	// Severity is what the site's `Severity:` field emits, as far as
	// static reading can tell. Empty when the literal carries no
	// Severity field at all.
	Severity findingSeverity
}

// findingSeverity is the severity a Finding{} construction site emits,
// read off the source rather than by running the rule.
type findingSeverity string

const (
	findingSeverityError   findingSeverity = "error"
	findingSeverityWarning findingSeverity = "warning"
	// findingSeverityVaries marks a site whose `Severity:` is not a
	// severity literal — a local assigned on a branch, or a call. The
	// value it takes is decided at run time by config or entity state,
	// which a static read cannot evaluate.
	findingSeverityVaries findingSeverity = "varies"
)

// resolveSeverityExpr classifies a `Severity:` field expression. A bare
// or package-qualified SeverityError / SeverityWarning is determinate;
// anything else varies, since resolving it would mean evaluating the
// condition that picks between the arms.
func resolveSeverityExpr(expr ast.Expr) findingSeverity {
	name := ""
	switch v := expr.(type) {
	case *ast.Ident:
		name = v.Name
	case *ast.SelectorExpr:
		name = v.Sel.Name
	default:
		return findingSeverityVaries
	}
	switch name {
	case "SeverityError":
		return findingSeverityError
	case "SeverityWarning":
		return findingSeverityWarning
	}
	return findingSeverityVaries
}

// emittedFindingCodeSites walks every non-test .go file's composite
// literals for a Code (and optional Subcode) field and returns one site
// per emission, resolving bare string literals, same-package Code*
// constants, and typed codespkg.Code{ID:…} descriptor references —
// including `CodeXxx.ID` and cross-package `pkg.CodeXxx.ID` selectors.
//
// It is the single source of truth for "what codes the check layer can
// emit", shared by PolicyFindingCodesHaveHints and
// PolicyFindingCodesDocumentedInSkill so the hint-completeness and
// skill-documentation chokepoints cannot drift from each other.
func emittedFindingCodeSites(files []FileEntry) []findingCodeSite {
	codeConsts := loadCheckCodeConstants(files)
	fset := token.NewFileSet()
	var out []findingCodeSite
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
			var severity findingSeverity
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
				case "Severity":
					severity = resolveSeverityExpr(kv.Value)
				}
			}
			if code == "" {
				return true
			}
			out = append(out, findingCodeSite{
				Code:     code,
				Subcode:  subcode,
				File:     f.Path,
				Line:     fset.Position(cl.Pos()).Line,
				Severity: severity,
			})
			return true
		})
	}
	return out
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

// loadCheckCodeConstants reads every const and var declaration in the
// check package and returns name → string-code value for two shapes:
// string-constant codes (`const CodeFoo = "foo"`) and typed descriptor
// codes (`var CodeFoo = codespkg.Code{ID: "foo"}`). Used to resolve
// `Code: CodeFoo` and `Code: CodeFoo.ID` references in Finding{}
// literals.
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
			if !ok || (gen.Tok != token.CONST && gen.Tok != token.VAR) {
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
					switch val := vs.Values[i].(type) {
					case *ast.BasicLit:
						// String-constant code form: take the quoted literal directly.
						if val.Kind != token.STRING {
							continue
						}
						if unq, err := strconv.Unquote(val.Value); err == nil {
							out[name.Name] = unq
						}
					case *ast.CompositeLit:
						// Typed-descriptor form (a codespkg.Code value): pull its ID field literal.
						if id, ok := compositeLitStringField(val, "ID"); ok {
							out[name.Name] = id
						}
					}
				}
			}
		}
	}
	return out
}

// compositeLitStringField returns the string-literal value of the named
// field in a composite literal (e.g. the ID of a codespkg.Code{ID:…}
// descriptor), and whether such a field was found.
func compositeLitStringField(cl *ast.CompositeLit, field string) (string, bool) {
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != field {
			continue
		}
		lit, ok := kv.Value.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		if s, err := strconv.Unquote(lit.Value); err == nil {
			return s, true
		}
	}
	return "", false
}

// resolveStringExpr extracts a string value from an AST expression
// when possible. Handles bare string literals, identifiers that resolve
// via codeConsts (`CodeFoo`), `.ID` selectors on a descriptor — both
// same-package (`CodeFoo.ID`) and cross-package (`pkg.CodeFoo.ID`) —
// and a cross-package reference to a string-constant code
// (`pkg.CodeFoo`). Returns "" otherwise — callers treat unresolved
// values as opaque and skip the policy check (we can't fairly judge a
// code we can't read).
//
// Names resolve through codeConsts, so the package qualifier carries no
// information and is dropped: a selector naming nothing the check
// package declares misses the map and reads as unresolved, exactly as
// an unknown bare identifier does. codeConsts indexes every string
// const and var in the check package, not only the Code* ones, so a
// `Code: pkg.SeverityError` would resolve to "error" rather than to
// nothing — a fabricated code that the documented-in-skill and hint
// policies both reject on sight, which is why the map is left broad
// rather than filtered down to a naming convention that would instead
// drop a future code someone names differently.
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
	case *ast.SelectorExpr:
		if v.Sel.Name != "ID" {
			// `pkg.CodeFoo` — a cross-package reference to a
			// string-constant code, which carries no `.ID` to select.
			return codeConsts[v.Sel.Name]
		}
		switch x := v.X.(type) {
		case *ast.Ident:
			return codeConsts[x.Name]
		case *ast.SelectorExpr:
			return codeConsts[x.Sel.Name]
		}
		return ""
	}
	return ""
}
