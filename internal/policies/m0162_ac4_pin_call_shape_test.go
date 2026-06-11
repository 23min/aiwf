package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestM0162_AC4_PinCallShapeRecognized mechanically closes the
// reviewer R3-T4 finding: the AST scanner at handlePinArg
// recognizes only two argument shapes for pinCell / branchtest.Pin
// calls:
//
//   - *ast.BasicLit with kind STRING
//   - *ast.BinaryExpr with token.ADD, LHS literal
//
// Any other shape (fmt.Sprintf, const reference, ident+"-suffix",
// multi-binary concat, function call returning a string) is
// SILENTLY skipped by the scanner. If a future contributor writes
// such a pattern, the bijection invariants 1+2 would surface the
// resulting orphan cell at CI time — but the silent miss in the
// scanner itself is invisible.
//
// This test walks every *_test.go file under internal/ and asserts
// that every pinCell(...) and branchtest.Pin(...) call's first
// argument matches one of the recognized shapes. If a new pattern
// appears, this test fires loudly naming the file:line and the
// AST type, prompting the contributor to either:
//
//   - Extend handlePinArg to recognize the new shape, OR
//   - Rewrite the call to use a recognized shape.
//
// Sabotage-verifiable: add `pinCell(fmt.Sprintf("branch-cell-%d",
// i), t.Name())` to any test file and this test fires.
func TestM0162_AC4_PinCallShapeRecognized(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	internalDir := filepath.Join(root, "internal")

	// Framework-seam files: these contain pinCell calls that
	// FORWARD dynamic values from other call sites (RunScenarios's
	// sc.CellID and pinCell helper's parameter pass-through). The
	// actual cell ID literal lives at the call sites these forward
	// from (Scenario struct field literals, inline pinCell calls in
	// test bodies). Exempting these files prevents false positives
	// on the framework's own forwarding chain.
	frameworkSeamFiles := map[string]bool{
		"branch_scenarios_helpers_test.go": true, // RunScenarios → pinCell(sc.CellID, t.Name())
		"pin_testpins_test.go":             true, // pinCell helper → branchtest.Pin(cellID, testName)
	}

	var unknownShapes []string
	walkErr := filepath.WalkDir(internalDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if frameworkSeamFiles[filepath.Base(path)] {
			return nil
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if perr != nil {
			return nil
		}
		base := filepath.Base(path)
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			if !isPinCall(call) {
				return true
			}
			arg := call.Args[0]
			if isRecognizedPinShape(arg) {
				return true
			}
			pos := fset.Position(arg.Pos())
			unknownShapes = append(unknownShapes,
				base+":"+strconv.Itoa(pos.Line)+" — argument of type "+astTypeName(arg))
			return true
		})
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk %s: %v", internalDir, walkErr)
	}

	if len(unknownShapes) > 0 {
		t.Errorf("M-0162/AC-4 (reviewer R3-T4): %d pinCell/branchtest.Pin call(s) with argument shapes the AST scanner does not recognize\n%s\n  Recognized shapes: BasicLit STRING; BinaryExpr ADD with LHS BasicLit.\n  Fix: extend handlePinArg() in m0162_ac4_bijection_test.go to recognize the new shape, OR rewrite the call site to use a recognized shape.",
			len(unknownShapes), "  "+strings.Join(unknownShapes, "\n  "))
	}
}

// isPinCall reports whether call is `pinCell(...)` or
// `branchtest.Pin(...)`.
func isPinCall(call *ast.CallExpr) bool {
	if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "pinCell" {
		return true
	}
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Pin" {
		if x, ok := sel.X.(*ast.Ident); ok && x.Name == "branchtest" {
			return true
		}
	}
	return false
}

// isRecognizedPinShape mirrors handlePinArg's accepted shapes.
// Update both when the scanner is extended.
func isRecognizedPinShape(arg ast.Expr) bool {
	// Shape 1: literal string.
	if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		return true
	}
	// Shape 2: BinaryExpr ADD with LHS literal string.
	if bx, ok := arg.(*ast.BinaryExpr); ok && bx.Op == token.ADD {
		if lit, ok := bx.X.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			return true
		}
	}
	return false
}

// astTypeName returns a human-readable name for an AST node type.
// Used in error reporting.
func astTypeName(n ast.Node) string {
	switch v := n.(type) {
	case *ast.BasicLit:
		return "BasicLit(" + v.Kind.String() + ")"
	case *ast.BinaryExpr:
		return "BinaryExpr(" + v.Op.String() + ")"
	case *ast.CallExpr:
		return "CallExpr"
	case *ast.Ident:
		return "Ident(" + v.Name + ")"
	case *ast.SelectorExpr:
		return "SelectorExpr"
	default:
		return "unknown"
	}
}
