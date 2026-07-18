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
// Reachability is a same-package call-graph walk over raw call
// text, not a type-checked analysis — the same shallow, deliberate
// scope PolicyVerbsValidateThenWrite already uses for its own
// substring scan. A verb entry point that only calls projectionFindings
// through an unexported helper (e.g. EditBody -> editBodyExplicit)
// is still recognized as compliant.
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
	bodies := map[string]string{}
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
			start := fset.Position(fn.Body.Lbrace).Offset
			end := fset.Position(fn.Body.Rbrace).Offset
			if start < 0 || end <= start || end > len(f.Contents) {
				continue
			}
			body := string(f.Contents[start:end])
			bodies[fn.Name.Name] = body
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
		if !reachesProjectionFindings(e.name, bodies, map[string]bool{}) {
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

// reachesProjectionFindings walks the same-package call graph
// starting at the function named name, returning true if
// projectionFindings is called anywhere in the reachable set. visited
// guards against infinite recursion on a call cycle.
func reachesProjectionFindings(name string, bodies map[string]string, visited map[string]bool) bool {
	if visited[name] {
		return false
	}
	visited[name] = true
	body, ok := bodies[name]
	if !ok {
		return false
	}
	if strings.Contains(body, "projectionFindings(") {
		return true
	}
	for callee := range bodies {
		if callee == name || visited[callee] {
			continue
		}
		if strings.Contains(body, callee+"(") && reachesProjectionFindings(callee, bodies, visited) {
			return true
		}
	}
	return false
}
