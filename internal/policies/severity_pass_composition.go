package policies

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strings"
)

// Anchors this policy hangs on. Every finding is phrased against one of
// these names, so each is asserted to exist before anything is read into
// its absence.
const (
	severityPkg      = "internal/severity"
	severitySeamFunc = "Apply"
	severityLoadFunc = "Load"
	severityFromFunc = "From"
	checkPkg         = "internal/check"
	checkRunFunc     = "Run"
	checkFindingType = "Finding"
)

// PolicySeverityPassComposition asserts that the aiwf.yaml severity
// escalations are composed in exactly one place, and that every surface
// reading findings goes through it.
//
// Two claims, in the two directions a shared seam can be defeated:
//
//   - Every severity pass internal/check exports is called by
//     severity.Apply. A pass that exists but is not wired reaches no
//     caller at all, which is the shape a newly-added knob takes on the
//     day it lands.
//   - Every production function calling check.Run also calls
//     severity.Apply. A call site composing its own subset — or none —
//     reports findings at severities `aiwf check` does not agree with.
//     On a read surface that under-reports; at the verb-time projection
//     guard, it means a verb reports success and commits a state the
//     pre-push hook then refuses.
//
// Four bounds a reader should hold:
//
//   - Both claims match on the selector spelling (`check.Run`,
//     `severity.Apply`). A call through an import alias, or through a
//     same-package helper this scan does not follow, reads as absent —
//     the aliasing blind spot the os.* write policies share. A finding
//     is fixed by calling the seam in the same function, which is where
//     it belongs regardless.
//   - Same-function granularity is deliberate. A call site that applies
//     the policy in a caller two frames up satisfies no check here, and
//     should not: the findings a function returns are the ones it must
//     have escalated.
//   - A surface that assembles findings without calling check.Run is
//     outside the scan entirely — `aiwf check --shape-only` composes
//     check.TreeDiscipline alone and is reached by no claim here. It
//     escalates nothing, and no pass covers a tree-discipline code, so
//     the omission is inert rather than latent; a future surface of the
//     same shape would need its own answer.
//   - It proves a call happened with the consumer's own policy, never
//     that the call had an effect. severity.Apply(nil, severity.Load(…),
//     t) satisfies every check here while escalating nothing, because
//     the findings argument is an expression this scan does not
//     evaluate. What closes that is a behavioral test per surface,
//     asserting an escalated finding reaches the surface's own output;
//     each wired surface carries one, except `aiwf doctor`, which reads
//     only ids-unique — a code no pass touches — and so has no
//     observable effect to assert. Likewise a pass whose rule a surface
//     never runs is inert there, which is what makes one uniform seam
//     safe; a pass that escalates the wrong code set is review's to
//     catch.
func PolicySeverityPassComposition(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil { //coverage:ignore defensive: WalkGoFiles errors only when the scan root is unreadable, which every other policy in this package would fail on first
		return nil, err
	}

	fset := token.NewFileSet()
	var (
		exportedPasses []string        // check.Apply* taking []Finding
		seamCalls      map[string]bool // selector names called inside severity.Apply
		seamFile       string
		seamFound      bool
	)
	var runSites []Violation

	for _, f := range files {
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil { //coverage:ignore defensive: the tree under scan compiles, so a parse failure needs a file edited mid-run
			continue
		}
		dir := path.Dir(f.Path)
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv != nil {
				continue
			}
			if dir == checkPkg && isSeverityPass(fn) {
				exportedPasses = append(exportedPasses, fn.Name.Name)
			}
			if dir == severityPkg && fn.Name.Name == severitySeamFunc {
				seamFound = true
				seamFile = f.Path
				seamCalls = selectorCalls(fn, "check")
			}
			if dir == severityPkg {
				// The seam's own package is where the passes are called
				// from; it never reads a tree of its own.
				continue
			}
			if !callsSelector(fn, "check", checkRunFunc) {
				continue
			}
			if !callsSelector(fn, "severity", severitySeamFunc) {
				runSites = append(runSites, Violation{
					Policy: "severity-pass-composition",
					File:   f.Path,
					Line:   fset.Position(fn.Pos()).Line,
					Detail: fmt.Sprintf("%s calls check.%s but never severity.%s — the findings it returns carry each rule's config-agnostic default severity, so this surface disagrees with `aiwf check` about the same finding on the same tree; apply the shared severity policy before the findings leave this function",
						fn.Name.Name, checkRunFunc, severitySeamFunc),
				})
				continue
			}
			if line, ok := unsourcedPolicyArg(fset, fn); !ok {
				runSites = append(runSites, Violation{
					Policy: "severity-pass-composition",
					File:   f.Path,
					Line:   line,
					Detail: fmt.Sprintf("%s calls severity.%s with a policy that does not come from severity.%s or severity.%s — a literal escalates nothing, so the call satisfies every structural check here while leaving the surface exactly as divergent as before; pass the consumer's own policy",
						fn.Name.Name, severitySeamFunc, severityLoadFunc, severityFromFunc),
				})
			}
		}
	}

	out := severityAnchorViolations(exportedPasses, seamFound)
	if !seamFound {
		// Every unwired-pass finding below is phrased against a seam that
		// was not found; reporting them too would blame each pass for the
		// anchor's absence.
		return append(out, runSites...), nil
	}
	sort.Strings(exportedPasses)
	for _, pass := range exportedPasses {
		if seamCalls[pass] {
			continue
		}
		out = append(out, Violation{
			Policy: "severity-pass-composition",
			File:   seamFile,
			Detail: fmt.Sprintf("check.%s is an exported severity pass that severity.%s never calls — a pass the seam does not compose reaches none of the surfaces routed through it, however many call sites its author remembered to edit; call it from the seam, or unexport it if it is not a severity pass",
				pass, severitySeamFunc),
		})
	}
	return append(out, runSites...), nil
}

// severityAnchorViolations reports an anchor this policy hangs on that
// the scan no longer finds. Without them every other finding here is
// either vacuous or blames the wrong file.
func severityAnchorViolations(passes []string, seamFound bool) []Violation {
	var out []Violation
	if len(passes) == 0 {
		out = append(out, Violation{
			Policy: "severity-pass-composition",
			File:   checkPkg + "/",
			Detail: fmt.Sprintf("no exported Apply* function taking []%s was found under %s — with no passes to compose, this policy would report green while checking nothing; repoint it at the passes' new location",
				checkFindingType, checkPkg),
		})
	}
	if !seamFound {
		out = append(out, Violation{
			Policy: "severity-pass-composition",
			File:   severityPkg + "/severity.go",
			Detail: fmt.Sprintf("func %s is not declared under %s — every finding here is phrased against that seam, so this policy cannot answer anything until the anchor is repointed",
				severitySeamFunc, severityPkg),
		})
	}
	return out
}

// unsourcedPolicyArg reports whether every severity.Apply call in fn is
// handed a policy that came from the consumer's configuration, and the
// line of the first that was not.
//
// The presence check above cannot tell a wired call site from one that
// passes severity.Policy{}: both call the seam, and the literal
// escalates nothing. The accepted forms are a direct severity.Load /
// severity.From call, or a local assigned from one — the shape a
// function needs when it applies one policy to two trees.
func unsourcedPolicyArg(fset *token.FileSet, fn *ast.FuncDecl) (line int, ok bool) {
	sourced := policySourcedLocals(fn)
	line, ok = 0, true
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if !ok {
			return false
		}
		sel, isSel := selectorOf(n)
		if !isSel || sel.X.(*ast.Ident).Name != "severity" || sel.Sel.Name != severitySeamFunc {
			return true
		}
		args := n.(*ast.CallExpr).Args
		if len(args) < 2 || !policySourced(args[1], sourced) {
			line, ok = fset.Position(n.Pos()).Line, false
			return false
		}
		return true
	})
	return line, ok
}

// policySourcedLocals returns the local names fn assigns from
// severity.Load or severity.From.
func policySourcedLocals(fn *ast.FuncDecl) map[string]bool {
	out := map[string]bool{}
	record := func(lhs, rhs []ast.Expr) {
		for i, r := range rhs {
			if i >= len(lhs) || !policyConstructor(r) {
				continue
			}
			if ident, isIdent := lhs[i].(*ast.Ident); isIdent {
				out[ident.Name] = true
			}
		}
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			record(s.Lhs, s.Rhs)
		case *ast.ValueSpec:
			for _, name := range s.Names {
				record([]ast.Expr{name}, s.Values)
			}
		}
		return true
	})
	return out
}

// policySourced reports whether arg is a policy the consumer declared.
func policySourced(arg ast.Expr, sourced map[string]bool) bool {
	if ident, isIdent := arg.(*ast.Ident); isIdent {
		return sourced[ident.Name]
	}
	return policyConstructor(arg)
}

// policyConstructor reports whether e is a severity.Load / severity.From
// call.
func policyConstructor(e ast.Expr) bool {
	sel, ok := selectorOf(e)
	if !ok || sel.X.(*ast.Ident).Name != "severity" {
		return false
	}
	return sel.Sel.Name == severityLoadFunc || sel.Sel.Name == severityFromFunc
}

// isSeverityPass reports whether fn is an exported severity pass: a
// package-level Apply* whose first parameter is a []Finding to mutate.
// The shape is what separates a pass from any other exported Apply.
func isSeverityPass(fn *ast.FuncDecl) bool {
	if !fn.Name.IsExported() || !strings.HasPrefix(fn.Name.Name, "Apply") {
		return false
	}
	if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return false
	}
	slice, ok := fn.Type.Params.List[0].Type.(*ast.ArrayType)
	if !ok || slice.Len != nil {
		return false
	}
	ident, ok := slice.Elt.(*ast.Ident)
	return ok && ident.Name == checkFindingType
}

// selectorCalls returns the set of selector names fn calls on pkg, e.g.
// {"ApplyTDDStrict": true} for a body calling check.ApplyTDDStrict.
func selectorCalls(fn *ast.FuncDecl, pkg string) map[string]bool {
	out := map[string]bool{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if sel, ok := selectorOf(n); ok && sel.X.(*ast.Ident).Name == pkg {
			out[sel.Sel.Name] = true
		}
		return true
	})
	return out
}

// callsSelector reports whether fn's body calls pkg.name.
func callsSelector(fn *ast.FuncDecl, pkg, name string) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found {
			return false
		}
		sel, ok := selectorOf(n)
		if ok && sel.X.(*ast.Ident).Name == pkg && sel.Sel.Name == name {
			found = true
		}
		return !found
	})
	return found
}

// selectorOf narrows a node to a call of the form <ident>.<name>(...).
func selectorOf(n ast.Node) (*ast.SelectorExpr, bool) {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, false
	}
	if _, ok := sel.X.(*ast.Ident); !ok {
		return nil, false
	}
	return sel, true
}
