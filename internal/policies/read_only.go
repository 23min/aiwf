package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// readOnlyVerb describes a read-only verb's expected location. As
// verbs migrate from cmd/aiwf/ to internal/cli/<pkg>/ (M-0115+),
// the policy needs to find the verb's body in either location.
type readOnlyVerb struct {
	// FuncName is the FuncDecl name the policy looks for. For verbs
	// still living in cmd/aiwf/, this is `run<Verb>Cmd` (legacy form).
	// For verbs migrated to internal/cli/<pkg>/, this is `Run` (the
	// canonical NewCmd/Run pair per the M-0115 subpackage pattern).
	FuncName string
	// FilePrefix is the path-prefix the policy walks to find the
	// FuncDecl. For legacy verbs: "cmd/aiwf/". For migrated verbs:
	// "internal/cli/<verb>/" (the per-verb subpackage directory).
	FilePrefix string
}

// readOnlyVerbs lists every verb the kernel treats as read-only.
// The kernel principle: reads are pure functions, mutations go
// through `verb.Apply`. A direct gitops.Commit / gitops.Mv /
// gitops.Add or os.WriteFile call from one of these is a regression.
//
// Render has two run functions — runRenderSiteCmd (read-only) and
// runRenderRoadmapCmd (writes only with --write, policed by the
// dedicated RenderRoadmap policy). Only the site path is listed.
var readOnlyVerbs = []readOnlyVerb{
	{FuncName: "runCheckCmd", FilePrefix: "cmd/aiwf/"},
	{FuncName: "Run", FilePrefix: "internal/cli/history/"},
	{FuncName: "runShowCmd", FilePrefix: "cmd/aiwf/"},
	{FuncName: "runDoctorCmd", FilePrefix: "cmd/aiwf/"},
	{FuncName: "runStatusCmd", FilePrefix: "cmd/aiwf/"},
	{FuncName: "Run", FilePrefix: "internal/cli/whoami/"},
	{FuncName: "Run", FilePrefix: "internal/cli/schema/"},
	{FuncName: "runRenderSiteCmd", FilePrefix: "cmd/aiwf/"},
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
// entry points contain no direct call to a known mutating function.
// Transitive mutations (a helper they call that calls gitops.Add)
// are not detected — a real call-graph analysis would be needed;
// this catches the direct case which is almost always how the
// regression starts.
//
// Each entry in readOnlyVerbs has a FilePrefix — the policy walks
// .go files under that prefix looking for a FuncDecl with the
// expected name. As verbs migrate from cmd/aiwf/run<Verb>Cmd to
// internal/cli/<verb>/Run (M-0115+), the entry's FilePrefix
// switches from "cmd/aiwf/" to "internal/cli/<verb>/".
func PolicyReadOnlyVerbsDoNotMutate(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	seen := map[int]bool{}
	for _, f := range files {
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		fileRel := filepath.ToSlash(f.Path)
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			for i, rv := range readOnlyVerbs {
				if fn.Name.Name != rv.FuncName {
					continue
				}
				if !strings.HasPrefix(fileRel, rv.FilePrefix) {
					continue
				}
				seen[i] = true
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
	}
	for i, rv := range readOnlyVerbs {
		if !seen[i] {
			out = append(out, Violation{
				Policy: "read-only-verbs-do-not-mutate",
				File:   rv.FilePrefix,
				Detail: "policy expects " + rv.FuncName + " under " + rv.FilePrefix +
					" but no FuncDecl with that name was found — update the policy or restore the verb",
			})
		}
	}
	return out, nil
}
