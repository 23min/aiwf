package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestSetContracts_HasOneCaller holds every write of aiwf.yaml's contracts
// block to setContracts in internal/verb/contractbind.go, the one place
// binding ids are written at canonical width. A writer that called
// Doc.SetContracts directly would write a binding id at whatever width it
// was handed, and no output test would see it unless someone thought to
// drive that writer.
func TestSetContracts_HasOneCaller(t *testing.T) {
	t.Parallel()
	files, err := WalkGoFiles(repoRoot(t), true)
	if err != nil {
		t.Fatal(err)
	}
	const home, homeFunc = "internal/verb/contractbind.go", "setContracts"
	fset := token.NewFileSet()
	var sawHome bool
	for _, f := range files {
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, 0)
		if perr != nil {
			continue
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "SetContracts" {
					return true
				}
				if f.Path == home && fn.Name.Name == homeFunc {
					sawHome = true
					return true
				}
				t.Errorf("%s:%d: %s calls SetContracts directly; route the write through %s in %s so binding ids are written canonical",
					f.Path, fset.Position(call.Pos()).Line, fn.Name.Name, homeFunc, home)
				return true
			})
		}
	}
	if !sawHome {
		t.Errorf("no SetContracts call found in %s's %s; the chokepoint this test guards has moved", home, homeFunc)
	}
}
