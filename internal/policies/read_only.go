package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// readOnlyVerbs lists the cmd/aiwf entry-point functions for verbs
// that must never mutate disk state. The kernel principle: reads
// are pure functions, mutations go through `verb.Apply`. A direct
// gitops.Commit / gitops.Mv / gitops.Add or os.WriteFile call from
// one of these is a regression — the verb is now writing without
// a Plan.
//
// Names ending in `Cmd` are the body-function shape introduced by
// E-14's Cobra migration: each Cobra command's RunE delegates to a
// `run<Verb>Cmd(...)` helper that holds the verb's actual logic.
// The legacy `runX(args []string) int` shape remains for verbs not
// yet migrated.
var readOnlyVerbs = map[string]bool{
	"runCheckCmd":      true,
	"runHistoryCmd":    true,
	"runShow":          true,
	"runDoctorCmd":     true,
	"runStatus":        true,
	"runWhoami":        true,
	"runSchemaCmd":     true,
	"runRenderSiteCmd": true, // html render path; runRenderRoadmapCmd writes only with --write and is policed via the RenderRoadmap rule
}

// forbiddenMutations is the set of function/method calls a
// read-only verb's body must not contain.
var forbiddenMutations = []string{
	"gitops.Commit",
	"gitops.CommitAllowEmpty",
	"gitops.Mv",
	"gitops.Add",
	"gitops.Restore",
	"verb.Apply",
	"os.Create",
	"os.WriteFile",
	"os.Remove",
	"os.RemoveAll",
}

// PolicyReadOnlyVerbsDoNotMutate asserts that the read-only-verb
// entry points contain no direct call to a known mutating
// function. Transitive mutations (a helper they call that calls
// gitops.Add) are not detected by this policy — a real call-graph
// analysis would be needed; this catches the direct case which is
// almost always how the regression starts.
//
// Note on render: post-E-14 the verb splits into two functions —
// runRenderSiteCmd (html path, no writes) and runRenderRoadmapCmd
// (markdown path, writes only with --write). Only the site path is
// listed above; the roadmap path is governed by the dedicated
// RenderRoadmap policy below.
func PolicyReadOnlyVerbsDoNotMutate(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	seen := map[string]bool{}
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "cmd/aiwf/") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !readOnlyVerbs[fn.Name.Name] {
				continue
			}
			seen[fn.Name.Name] = true
			if fn.Body == nil {
				continue
			}
			start := fset.Position(fn.Body.Lbrace).Offset
			end := fset.Position(fn.Body.Rbrace).Offset
			if start < 0 || end <= start || end > len(f.Contents) {
				continue
			}
			body := string(f.Contents[start:end])
			for _, mut := range forbiddenMutations {
				if strings.Contains(body, mut) {
					out = append(out, Violation{
						Policy: "read-only-verbs-do-not-mutate",
						File:   f.Path,
						Line:   fset.Position(fn.Pos()).Line,
						Detail: fn.Name.Name + " calls " + mut +
							" — read-only verbs must not write disk state directly",
					})
				}
			}
		}
	}
	for name := range readOnlyVerbs {
		if !seen[name] {
			out = append(out, Violation{
				Policy: "read-only-verbs-do-not-mutate",
				File:   "cmd/aiwf/",
				Detail: "policy lists " + name +
					" but no FuncDecl with that name was found — update the policy or restore the verb",
			})
		}
	}
	return out, nil
}
