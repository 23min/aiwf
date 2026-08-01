package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// projectionFindingsExemptVerbs is the explicit, reviewed allowlist of
// exported internal/verb/*.go entry points (functions returning
// (*Result, error)) that legitimately never reach projectionFindings —
// the validate-then-write gate every other mutating verb runs before
// writing. Each Reason names the concrete, source-grounded reason
// projectionFindings cannot or need not fire — not just "doesn't call
// it." A new entry requires a reason of the same shape, reviewed the
// same way any other code change is; see internal/verb/verb.go's
// package doc for the four reason categories this list draws from.
var projectionFindingsExemptVerbs = []struct {
	File   string
	Func   string
	Reason string
}{
	{"setarea.go", "SetArea", "area-mistag/unknown/overlap need a touchedByEntity map built by scanning commit history, unreachable from an in-memory projection; gated by the pre-push hook's full aiwf check instead"},
	{"setpriority.go", "SetPriority", "shares SetArea's rationale: priority has no check.Run rule computable from an in-memory projection"},
	{"renamearea.go", "RenameArea", "shares SetArea's rationale: area-membership rules are git-history-dependent"},
	{"authorize.go", "Authorize", "records a scope event via an empty-diff commit (Plan.AllowEmpty); no entity-content mutation exists to project"},
	{"acknowledgeillegal.go", "AcknowledgeIllegal", "sovereign empty-diff act (Plan.AllowEmpty); no entity-content mutation to project"},
	{"acknowledgemistag.go", "AcknowledgeMistag", "sovereign empty-diff act (Plan.AllowEmpty); no entity-content mutation to project"},
	{"auditonly.go", "PromoteAuditOnly", "audit-only recovery mode (G24): refuses unless the entity is already at the target state, so the commit is empty-diff; nothing to project"},
	{"auditonly.go", "PromoteACPhaseAuditOnly", "same audit-only rationale as PromoteAuditOnly"},
	{"auditonly.go", "CancelAuditOnly", "same audit-only rationale as PromoteAuditOnly"},
	{"archive.go", "Archive", "purely structural multi-entity sweep (file moves by status, no field-level content change); same shape as Rewidth, validation deferred to the pre-push hook"},
	{"rewidth.go", "Rewidth", "purely structural multi-entity sweep; its own doc comment states check.Run on a tree mid-rename would be spurious noise, deferred to the pre-push hook"},
	{"contractbind.go", "ContractBind", "contract subsystem verb: writes aiwf.yaml's contracts: block, not an entity file; runs contractCheckForBinding, a narrower scoped gate, by design"},
	{"contractbind.go", "ContractUnbind", "contract subsystem verb: writes aiwf.yaml; a referential-integrity check is enough to remove a binding, no config-correspondence gate needed"},
	{"contractrecipe.go", "RecipeInstall", "contract subsystem verb: writes aiwf.yaml; idempotency/--force checks only, by design"},
	{"contractrecipe.go", "RecipeRemove", "contract subsystem verb: writes aiwf.yaml; a referential-integrity scan only, by design"},
}

// PolicyVerbsProjectionFindingsPresence asserts that every exported
// internal/verb/*.go function returning (*Result, error) — a verb
// entry point — calls projectionFindings, directly or via a
// same-package helper it calls into, unless it appears on the
// reviewed allowlist above. This is the mirror image of
// PolicyVerbsValidateThenWrite: that policy bans forbidden writer
// calls from every exported verb; this one requires the validation
// call to be present, so a verb that skips it silently (rather than
// for one of the allowlist's documented reasons) fails CI instead of
// surfacing only at the next verb-layer audit.
//
// Reachability is a same-package call-graph walk over bare-identifier
// call expressions, not a type-checked analysis. A verb entry point
// that only calls projectionFindings through an unexported helper
// (e.g. EditBody -> editBodyExplicit) is still recognized as compliant.
// PolicyVerbsValidateThenWrite answers its own question with a
// substring scan; these two no longer share a mechanism.
func PolicyVerbsProjectionFindingsPresence(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}

	type entryPoint struct {
		name string
		file string
		line int
	}

	fset := token.NewFileSet()
	graph := callGraph{}
	var entries []entryPoint

	for _, f := range files {
		if !strings.HasPrefix(f.Path, "internal/verb/") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv != nil {
				continue
			}
			graph[fn.Name.Name] = calledIdents(fn)
			if isCapitalized(fn.Name.Name) && returnsResultAndError(fn.Type) {
				entries = append(entries, entryPoint{
					name: fn.Name.Name,
					file: f.Path,
					line: fset.Position(fn.Pos()).Line,
				})
			}
		}
	}

	exempt := map[string]bool{}
	for _, e := range projectionFindingsExemptVerbs {
		exempt[e.Func] = true
	}

	var out []Violation
	for _, e := range entries {
		if exempt[e.name] {
			continue
		}
		if !reachesCall(e.name, "projectionFindings", graph, map[string]bool{}) {
			out = append(out, Violation{
				Policy: "projection-findings-presence",
				File:   e.file,
				Line:   e.line,
				Detail: e.name + " never calls projectionFindings, directly or via a same-package helper, and is not on the reviewed allowlist in projection_findings_presence.go — call it, or add a source-grounded allowlist entry",
			})
		}
	}
	return out, nil
}

// returnsResultAndError reports whether t's result list is exactly
// (*Result, error) — the verb entry-point signature.
func returnsResultAndError(t *ast.FuncType) bool {
	if t.Results == nil || len(t.Results.List) != 2 {
		return false
	}
	star, ok := t.Results.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	ident, ok := star.X.(*ast.Ident)
	if !ok || ident.Name != "Result" {
		return false
	}
	errIdent, ok := t.Results.List[1].Type.(*ast.Ident)
	return ok && errIdent.Name == "error"
}

// callGraph maps a same-package function name to the set of unqualified
// function names its body calls.
type callGraph map[string]map[string]bool

// calledIdents returns the unqualified function names fn calls.
//
// Only a bare identifier in call position counts, which is what confines
// the result to same-package functions: a qualified call (gitops.Rename)
// and a method call (e.Paths) reach code this walk does not model, and a
// name appearing in a comment or a string is not a call at all. Matching
// on source text instead would answer a different question — a body
// containing applyDirRename would read as calling Rename, so a verb that
// consults nothing would read as reaching whatever Rename reaches.
func calledIdents(fn *ast.FuncDecl) map[string]bool {
	out := map[string]bool{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if ident, ok := call.Fun.(*ast.Ident); ok {
			out[ident.Name] = true
		}
		return true
	})
	return out
}

// reachesCall walks the same-package call graph starting at the function
// named name, returning true if target is called anywhere in the
// reachable set. visited guards against infinite recursion on a call
// cycle.
//
// An edge is a name in call position, which is weaker than a call that
// runs. A call inside a branch no input selects is an edge; so is a call
// to a local variable that shadows the package function of that name.
// And a function outside graph — declared in another package, or reached
// through a value rather than a name — ends the walk, so a real call can
// also be missed.
//
// So the walk answers "is there a call-shaped edge to this name", and a
// caller may read neither presence nor absence as proof about what
// executes. What it is good for is the question both consumers ask: does
// this function's source connect to a named seam at all, or has it been
// written as if that seam did not exist.
func reachesCall(name, target string, graph callGraph, visited map[string]bool) bool {
	if visited[name] {
		return false
	}
	visited[name] = true
	calls, ok := graph[name]
	if !ok {
		return false
	}
	if calls[target] {
		return true
	}
	for callee := range calls {
		if reachesCall(callee, target, graph, visited) {
			return true
		}
	}
	return false
}
