package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// PolicyCaptureStdoutSingleton asserts that `func captureStdout`
// (or its exported sibling `func CaptureStdout`) is defined in
// exactly one place: internal/cli/cliutil/testutil/capture.go.
//
// Pre-M-0118 the helper was duplicated across cmd/aiwf/helpers_test.go
// and internal/cli/initcmd/helpers_test.go because _test.go files can't
// cross package boundaries. M-0118/AC-7 lifted the canonical
// implementation to a shared testutil package; this policy is the
// drift chokepoint that prevents the duplication from creeping back.
//
// The rule scans every .go file under the repo for FuncDecls named
// "captureStdout" or "CaptureStdout". Exactly one declaration is
// expected, and it must be in internal/cli/cliutil/testutil/. Any
// duplicate or misplaced declaration fires a violation pointing at
// the offending file:line.
func PolicyCaptureStdoutSingleton(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, false)
	if err != nil {
		return nil, err
	}
	const wantPrefix = "internal/cli/cliutil/testutil/"
	var out []Violation
	canonicalSeen := false
	fset := token.NewFileSet()
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
			name := fn.Name.Name
			if name != "captureStdout" && name != "CaptureStdout" {
				continue
			}
			if strings.HasPrefix(fileRel, wantPrefix) {
				canonicalSeen = true
				continue
			}
			out = append(out, Violation{
				Policy: "capture-stdout-singleton",
				File:   f.Path,
				Line:   fset.Position(fn.Pos()).Line,
				Detail: name + " is defined outside the canonical testutil location; M-0118/AC-7 lifted it to " +
					wantPrefix + "capture.go. Call testutil.CaptureStdout instead of redefining the helper per-package.",
			})
		}
	}
	if !canonicalSeen {
		out = append(out, Violation{
			Policy: "capture-stdout-singleton",
			File:   wantPrefix,
			Detail: "expected canonical CaptureStdout under " + wantPrefix +
				" but no FuncDecl with that name was found — restore the helper or update the policy",
		})
	}
	return out, nil
}
