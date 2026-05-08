package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// PolicyFilepathJoinSegmentBySegment flags `filepath.Join(...)`
// calls whose argument-after-the-first contains an embedded path
// separator. The Windows-portability rule:
//
//	filepath.Join(root, "work/epics/foo")  // ❌ literal "/" embedded
//	filepath.Join(root, "work", "epics", "foo")  // ✅ segments
//
// `filepath.Join` cleans separators so the embedded-slash form
// happens to work today on every platform — but the form
// communicates a Unix-only assumption that future readers may
// copy into code paths where Join doesn't normalize (e.g.
// `os.WriteFile("work/epics/" + name)`). The policy enforces the
// segment-by-segment shape so the codebase carries one
// platform-portable convention.
//
// Special case: the first argument is exempt. The first arg is
// usually a `root` variable (computed) or an absolute path string;
// embedded separators there reflect intent (the caller is naming
// a real path) and gocritic's rule narrows to the same scope.
//
// Both forward and backward slashes count as path separators.
// Backslash usually appears only via `\\` escape sequences in Go
// strings; gocritic catches `\\` literals just as well.
func PolicyFilepathJoinSegmentBySegment(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, false) // include tests; portability matters in tests too
	if err != nil {
		return nil, err
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		// The policies package itself uses filepath.Join in its
		// scanners; it's already segment-by-segment everywhere
		// that matters. Skipping avoids self-diagnosis noise.
		if strings.HasPrefix(f.Path, "internal/policies/") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "filepath" || sel.Sel.Name != "Join" {
				return true
			}
			// Inspect args[1:] (skip the first; gocritic does too).
			for i := 1; i < len(call.Args); i++ {
				lit, ok := call.Args[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				val, err := strconv.Unquote(lit.Value)
				if err != nil {
					continue
				}
				if !strings.ContainsAny(val, "/\\") {
					continue
				}
				out = append(out, Violation{
					Policy: "filepath-join-segment-by-segment",
					File:   f.Path,
					Line:   fset.Position(call.Pos()).Line,
					Detail: "filepath.Join arg " + strconv.Quote(val) +
						" embeds a path separator; pass each segment as a separate argument so the call communicates platform-portable intent",
				})
			}
			return true
		})
	}
	return out, nil
}
