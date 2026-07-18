package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// PolicyCrossBranchScanConsolidation asserts that the cross-branch
// collision scan (trunk.DetectCollisions) is composed in exactly one
// place — trunk.ScanCrossBranch — and never invoked directly by a
// consumer. E-0060 shipped the cross-branch read path with the
// LocalRefHits+RemoteRefHits union and the DetectCollisions blob-stat
// pass copied eagerly across three call sites (cliutil.LoadTreeWithTrunk,
// list.crossBranchListRows, show.buildCrossBranchShowView); each
// recomputed the same O(entities×refs) scan and discarded nearly all of
// it (G-0418). Consolidating the composition behind a single trunk
// helper keeps the "hits scanned equal the hits handed to
// DetectCollisions" coupling in one place, and is the seam the lazy
// absent-id scan hangs off.
//
// A cross-package call to trunk.DetectCollisions is the regression this
// forbids: the sanctioned caller is trunk.ScanCrossBranch, which lives
// in package trunk and invokes DetectCollisions unqualified — so a
// `trunk.DetectCollisions(...)` selector only ever appears in a
// re-duplicated consumer. Scope is every non-test Go file the repo walk
// returns; a new direct call surfaces here with a finding pointing at
// the offending line.
//
// Known blind spots, consistent with the repo's other AST policies: an
// aliased trunk import (`t "…/trunk"`) or a method-value indirection
// (`f := trunk.DetectCollisions`) is not matched.
func PolicyCrossBranchScanConsolidation(root string) ([]Violation, error) {
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
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "trunk" || sel.Sel.Name != "DetectCollisions" {
				return true
			}
			out = append(out, Violation{
				Policy: "cross-branch-scan-consolidation",
				File:   f.Path,
				Line:   fset.Position(call.Pos()).Line,
				Detail: "trunk.DetectCollisions called directly; route the cross-branch " +
					"collision scan through trunk.ScanCrossBranch so the union+collision " +
					"composition lives in one place (G-0418)",
			})
			return true
		})
	}
	return out, nil
}
