package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// PolicyNoRetryLoopsOnGitErrors flags for-loops in production code
// that contain a `git` invocation in the body. Retries on git
// failures hide environmental problems (lock contention, signing
// failures, network errors) and can race against the holder of a
// lock; the kernel pattern is "diagnose, surface, let the operator
// decide" — see step 5c's lock-contention diagnostic.
//
// Detection: AST-walk every for-statement; flag bodies that contain
// `exec.Command(... "git" ...)` or `exec.CommandContext(... "git"
// ...)` calls. The verb package's Apply path is exempt because it
// runs git steps in sequence, not in retry — the policy lists
// known-OK functions to skip.
func PolicyNoRetryLoopsOnGitErrors(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	exemptFunctions := map[string]bool{
		// Apply runs git steps once each; not a retry.
		"Apply": true,
	}
	var out []Violation
	fset := token.NewFileSet()
	for _, f := range files {
		// Tests legitimately retry git in fixture setup.
		if strings.HasSuffix(f.Path, "_test.go") {
			continue
		}
		astFile, perr := parser.ParseFile(fset, f.AbsPath, f.Contents, parser.AllErrors)
		if perr != nil {
			continue
		}
		// Track the enclosing function for each for-statement so we
		// can apply the exemption list. Inspect with a stack.
		var stack []*ast.FuncDecl
		ast.Inspect(astFile, func(n ast.Node) bool {
			if n == nil {
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
				return true
			}
			if fn, ok := n.(*ast.FuncDecl); ok {
				stack = append(stack, fn)
				return true
			}
			loop, ok := n.(*ast.ForStmt)
			if !ok {
				return true
			}
			if loop.Body == nil {
				return true
			}
			start := fset.Position(loop.Body.Lbrace).Offset
			end := fset.Position(loop.Body.Rbrace).Offset
			if start < 0 || end <= start || end > len(f.Contents) {
				return true
			}
			body := string(f.Contents[start:end])
			if !strings.Contains(body, `"git"`) && !strings.Contains(body, `exec.Command`) {
				return true
			}
			// Apply exemption if the enclosing function is on the
			// list.
			if len(stack) > 0 {
				enclosing := stack[len(stack)-1]
				if exemptFunctions[enclosing.Name.Name] {
					return true
				}
			}
			out = append(out, Violation{
				Policy: "no-retry-loops-on-git-errors",
				File:   f.Path,
				Line:   fset.Position(loop.Pos()).Line,
				Detail: "for-loop body invokes git; the kernel does not silently retry git failures (lock contention, signing). Diagnose and surface instead",
			})
			return true
		})
	}
	return out, nil
}
