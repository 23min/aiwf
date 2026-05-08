package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// PolicyVerbsValidateThenWrite asserts that exported verb functions
// in internal/verb/ do not directly call file-writing
// primitives. The kernel pattern is "validate-then-write": a verb
// returns a *Plan; verb.Apply (in apply.go) is the only writer.
//
// Forbidden direct calls inside an exported verb function body:
// gitops.Mv, gitops.Add, gitops.Restore, gitops.Commit,
// gitops.CommitAllowEmpty, os.WriteFile, os.Create, os.Remove. A
// regression — say, a new verb that "just creates the file inline"
// — surfaces here.
//
// Apply itself (the writer) is exempt by name. Helpers like
// auditOnlyTrailers and helpers prefixed with lowercase are
// internal: they're not exported verbs, so the rule doesn't apply
// directly. The policy targets capitalized func names (the verb
// API).
func PolicyVerbsValidateThenWrite(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	mutators := []string{
		"gitops.Mv(",
		"gitops.Add(",
		"gitops.Restore(",
		"gitops.Commit(",
		"gitops.CommitAllowEmpty(",
		"os.WriteFile(",
		"os.Create(",
		"os.Remove(",
		"os.RemoveAll(",
	}
	var out []Violation
	fset := token.NewFileSet()
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
			if !ok || fn.Body == nil {
				continue
			}
			// Exclude Apply itself — it is the writer.
			if fn.Name.Name == "Apply" || fn.Name.Name == "rollback" {
				continue
			}
			// Only check exported verb functions (Capitalized).
			if !isCapitalized(fn.Name.Name) {
				continue
			}
			start := fset.Position(fn.Body.Lbrace).Offset
			end := fset.Position(fn.Body.Rbrace).Offset
			if start < 0 || end <= start || end > len(f.Contents) {
				continue
			}
			body := string(f.Contents[start:end])
			for _, mut := range mutators {
				if strings.Contains(body, mut) {
					out = append(out, Violation{
						Policy: "verbs-validate-then-write",
						File:   f.Path,
						Line:   fset.Position(fn.Pos()).Line,
						Detail: fn.Name.Name + " calls " + strings.TrimSuffix(mut, "(") +
							" directly; verbs must build a *Plan and let verb.Apply write",
					})
				}
			}
		}
	}
	return out, nil
}

func isCapitalized(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	return c >= 'A' && c <= 'Z'
}
