package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// commitConstructionSeamFile is the one file allowed to call both
// gitops.CommitTree and gitops.ReconcilePaths directly: the exported
// entry point (gitops.CommitVerbChange, M-0186/AC-5) that bundles the
// full verb-commit sequence — commit-tree, the post-commit hook,
// reconciliation — so a future consumer reuses it instead of composing
// the two primitives itself a second time.
const commitConstructionSeamFile = "internal/gitops/verbcommit.go"

// commitConstructionSeamFunc is the exported entry point's name.
const commitConstructionSeamFunc = "CommitVerbChange"

// commitConstructionSeamCaller is the one file/function allowed to call
// the seam itself — verb.Apply, its sole current consumer.
const (
	commitConstructionSeamCallerFile = "internal/verb/apply.go"
	commitConstructionSeamCallerFunc = "Apply"
)

// commitConstructionViolation is the single Violation construction site
// for this policy — every firing path funnels through it so one firing
// fixture covers all three checks below (G-0259/firing-fixture-presence).
func commitConstructionViolation(file string, line int, detail string) Violation {
	return Violation{
		Policy: "commit-construction-single-seam",
		File:   file,
		Line:   line,
		Detail: detail,
	}
}

// PolicyCommitConstructionSingleSeam asserts M-0186/AC-5: the
// commit-construction core is reachable through exactly one exported
// seam (gitops.CommitVerbChange), and that seam has exactly one caller
// today (verb.Apply). Three checks:
//
//  1. gitops.CommitVerbChange is declared in commitConstructionSeamFile.
//  2. Nothing outside that file calls gitops.CommitTree or
//     gitops.ReconcilePaths directly — a second file composing the two
//     primitives would be a duplicate, ad hoc commit-construction path.
//  3. Nothing outside verb.Apply calls gitops.CommitVerbChange — a
//     second caller today would contradict the AC's "sole caller"
//     claim; a real second consumer arriving later updates this policy
//     alongside adding itself.
func PolicyCommitConstructionSingleSeam(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	seamFuncFound := false

	for _, f := range files {
		if !strings.HasPrefix(f.Path, "internal/verb/") && !strings.HasPrefix(f.Path, "internal/gitops/") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if f.Path == commitConstructionSeamFile && fn.Name.Name == commitConstructionSeamFunc {
				seamFuncFound = true
			}
			if fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok || pkg.Name != "gitops" {
					return true
				}
				switch sel.Sel.Name {
				case "CommitTree", "ReconcilePaths":
					if f.Path != commitConstructionSeamFile {
						out = append(out, commitConstructionViolation(f.Path, fset.Position(call.Pos()).Line,
							"calls gitops."+sel.Sel.Name+" directly outside "+commitConstructionSeamFile+
								" — route through gitops."+commitConstructionSeamFunc+" so commit-construction stays one reusable seam"))
					}
				case commitConstructionSeamFunc:
					if f.Path != commitConstructionSeamCallerFile || fn.Name.Name != commitConstructionSeamCallerFunc {
						out = append(out, commitConstructionViolation(f.Path, fset.Position(call.Pos()).Line,
							"calls gitops."+commitConstructionSeamFunc+" from "+fn.Name.Name+
								" — verb."+commitConstructionSeamCallerFunc+" is the seam's sole caller today; route through it"))
					}
				}
				return true
			})
		}
	}

	if !seamFuncFound {
		out = append(out, commitConstructionViolation(commitConstructionSeamFile, 0,
			"gitops."+commitConstructionSeamFunc+" not found — the commit-construction core must expose this exported entry point (M-0186/AC-5)"))
	}

	return out, nil
}
