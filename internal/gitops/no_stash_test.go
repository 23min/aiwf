package gitops

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// forbiddenStashSymbols pins M-0186/AC-3: the git-stash-based verb
// commit isolation (G-0275/G-0276) is retired in favor of
// gitops.CommitTree + gitops.ReconcilePaths, which never touch the
// live index at all. These four symbols must never reappear in this
// package — a structural (AST-based) check, not a compiling reference,
// since referencing a removed symbol would simply fail to build rather
// than prove absence.
var forbiddenStashSymbols = map[string]bool{
	"StashStaged": true,
	"StashPop":    true,
	"StashTopRef": true,
	"StashDrop":   true,
}

// TestStashSymbolsDoNotExist walks every non-test .go file in this
// package's directory and fails if any top-level declaration
// (function, type, var, const) is named one of the retired Stash*
// symbols. Test files are excluded deliberately: a *_test.go file
// mentioning "Stash" in a doc comment (like this one) must not
// self-trigger the check.
func TestStashSymbolsDoNotExist(t *testing.T) {
	t.Parallel()
	dir := "."
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || hasTestSuffix(name) {
			continue
		}
		path := filepath.Join(dir, name)
		astFile, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		for _, decl := range astFile.Decls {
			for _, name := range declaredNames(decl) {
				if forbiddenStashSymbols[name] {
					t.Errorf("%s declares %s — the git-stash isolation primitive was retired in M-0186/AC-3 and must not reappear", path, name)
				}
			}
		}
	}
}

// hasTestSuffix reports whether a filename ends in "_test.go".
func hasTestSuffix(name string) bool {
	const suffix = "_test.go"
	return len(name) >= len(suffix) && name[len(name)-len(suffix):] == suffix
}

// declaredNames returns every top-level identifier a declaration
// introduces: the function name for a FuncDecl, or every spec's
// name(s) for a GenDecl (var/const/type).
func declaredNames(decl ast.Decl) []string {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		return []string{d.Name.Name}
	case *ast.GenDecl:
		var names []string
		for _, spec := range d.Specs {
			switch s := spec.(type) {
			case *ast.ValueSpec:
				for _, n := range s.Names {
					names = append(names, n.Name)
				}
			case *ast.TypeSpec:
				names = append(names, s.Name.Name)
			}
		}
		return names
	}
	return nil
}
