package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// cliHelperFuncNames is the closed set of helper function names that
// were lifted from cmd/aiwf/main.go into internal/cli/cliutil/ as part
// of M-0114/AC-2. Re-declaring any of them in the cmd/aiwf package
// would silently shadow the canonical cliutil exports and is the
// regression this policy guards against.
//
// The unexported walkUpFor helper is included even though it's only
// referenced by ResolveRoot — duplicating it in cmd/aiwf would
// re-create the same staleness the cliutil lift was supposed to end.
var cliHelperFuncNames = map[string]bool{
	"resolveRoot":              true,
	"walkUpFor":                true,
	"registerFormatCompletion": true,
	"allKindNames":             true,
	"statusesForID":            true,
	"completeEntityIDs":        true,
	"completeEntityIDFlag":     true,
	"completeEntityIDArg":      true,
}

// PolicyCLIHelperLocations asserts that none of the helper functions
// lifted to cliutil during M-0114 are declared (as top-level function
// declarations) inside cmd/aiwf/. After the lift, every cmd/aiwf
// caller routes through cliutil.RegisterFormatCompletion / AllKindNames /
// StatusesForID / CompleteEntityIDs / CompleteEntityIDFlag /
// CompleteEntityIDArg / ResolveRoot.
//
// Test files inside cmd/aiwf are exempt: a fixture or helper test
// might legitimately define a local function whose name collides
// with one of these. WalkGoFiles is invoked with excludeTests=true,
// which also keeps the assertion narrow to production code.
//
// Methods (functions with a receiver) are out of scope — the policy
// only matches top-level FuncDecls without a receiver.
func PolicyCLIHelperLocations(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
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
			if !ok || fn.Recv != nil {
				continue
			}
			if !cliHelperFuncNames[fn.Name.Name] {
				continue
			}
			out = append(out, Violation{
				Policy: "cli-helper-locations",
				File:   f.Path,
				Line:   fset.Position(fn.Pos()).Line,
				Detail: "function " + fn.Name.Name + " was lifted to internal/cli/cliutil/ in M-0114; re-declaring it in cmd/aiwf shadows the canonical export — route the caller through cliutil instead",
			})
		}
	}
	return out, nil
}
