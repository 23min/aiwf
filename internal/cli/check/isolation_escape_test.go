package check

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

// TestRunProvenanceCheck_AC13_IsolationEscapeWired pins M-0106/AC-13's
// CLI-integration half: the RunProvenanceCheck function in this
// package contains a literal call to `check.RunIsolationEscape`.
// The function call is the wire-up that hooks the new kernel rule
// into the pre-push pipeline; without it the rule is dead code
// regardless of how complete its algorithm is.
//
// The assertion is AST-level (not a substring match on the source)
// per CLAUDE.md §"Substring assertions are not structural
// assertions". A regression that comments out the call,
// accidentally reorders the function so the call lives in dead
// code, or renames it fires this test.
//
// The test is deliberately strict on identifier shape: it matches
// any call expression whose `Fun` is a `*ast.SelectorExpr` with
// X.Name == "check" AND Sel.Name == "RunIsolationEscape". A regression
// that swaps the package alias breaks the package alias test, not
// this test; if a downstream rename is intended, the test fails
// loudly and the author updates it deliberately.
func TestRunProvenanceCheck_AC13_IsolationEscapeWired(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	path, err := filepath.Abs("provenance.go")
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parser.ParseFile(%s): %v", path, err)
	}

	var runProvenanceCheck *ast.FuncDecl
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fd.Name.Name == "RunProvenanceCheck" {
			runProvenanceCheck = fd
			break
		}
	}
	if runProvenanceCheck == nil {
		t.Fatal("RunProvenanceCheck function declaration not found in provenance.go")
	}

	var found bool
	ast.Inspect(runProvenanceCheck, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		x, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if x.Name == "check" && sel.Sel.Name == "RunIsolationEscape" {
			found = true
			return false
		}
		return true
	})

	if !found {
		t.Error("RunProvenanceCheck must contain a call to check.RunIsolationEscape — the wire-up that hooks M-0106's isolation-escape rule into the pre-push pipeline (AC-13)")
	}
}
