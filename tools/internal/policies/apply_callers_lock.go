package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// PolicyApplyCallersAcquireLock asserts that every cmd dispatcher
// (a `run*` entry point) that calls `verb.Apply` directly also
// calls `acquireRepoLock`. Apply is the only path that writes to
// disk; without the repo-lock, two concurrent verb invocations
// could corrupt each other's state.
//
// Scope: only `runX` functions in tools/cmd/aiwf/. Internal
// helpers (decorateAndFinish, finishVerb) are exempt — the policy
// trusts that the run-dispatcher above them takes the lock.
func PolicyApplyCallersAcquireLock(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "tools/cmd/aiwf/") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			// Only check dispatcher entry points (runX). Helpers like
			// decorateAndFinish / finishVerb run inside a dispatcher
			// that already took the lock; this policy trusts that
			// invariant rather than requiring the helper to re-take
			// the lock.
			if !strings.HasPrefix(fn.Name.Name, "run") {
				continue
			}
			start := fset.Position(fn.Body.Lbrace).Offset
			end := fset.Position(fn.Body.Rbrace).Offset
			if start < 0 || end <= start || end > len(f.Contents) {
				continue
			}
			body := string(f.Contents[start:end])
			callsApply := strings.Contains(body, "verb.Apply(") ||
				strings.Contains(body, "finishVerb") ||
				strings.Contains(body, "decorateAndFinish")
			if !callsApply {
				continue
			}
			hasLock := strings.Contains(body, "acquireRepoLock")
			if !hasLock {
				out = append(out, Violation{
					Policy: "apply-callers-acquire-lock",
					File:   f.Path,
					Line:   fset.Position(fn.Pos()).Line,
					Detail: fn.Name.Name +
						" calls verb.Apply (or finishVerb / decorateAndFinish) without acquireRepoLock; concurrent invocations could corrupt repo state",
				})
			}
		}
	}
	return out, nil
}
