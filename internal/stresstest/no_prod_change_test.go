package stresstest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"testing"
	"unicode"

	"github.com/google/go-cmp/cmp"
)

// The stress probes must work without dedicated production hooks. This guard
// pins the reviewed exported surface of the production files they observe;
// unrelated API additions require review and an explicit baseline update.
// LockKillScenario uses repolock's public lock API. MidWriteKillScenario observes
// AtomicWriteFile's documented temporary filenames without importing pathutil.
func TestNoNewExportsInRepolockOrPathutil(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "repolock (unix) — the only file AC-1's probe touches; the harness never runs on windows",
			path: "../repolock/repolock_unix.go",
			want: []string{"Acquire", "ErrBusy", "Lock", "Lock.Release"},
		},
		{
			name: "pathutil.go — AC-2 never imports internal/pathutil at all, so its exports are irrelevant, not merely unchanged",
			path: "../pathutil/pathutil.go",
			want: []string{"ErrNotAbsolute", "Inside", "Resolve"},
		},
		{
			name: "atomic.go — AC-2 observes AtomicWriteFile's temp-file side effect from outside; it never calls the function",
			path: "../pathutil/atomic.go",
			want: []string{"AtomicWriteFile", "StageAtomicWriteFile"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := exportedTopLevelNames(t, tc.path)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("exported surface of %s mismatch (-want +got):\n%s", tc.path, diff)
			}
		})
	}
}

// exportedTopLevelNames parses the .go file at path and returns the
// sorted set of its exported top-level declarations: free functions
// and methods (reported as "Recv.Method"), and const/var/type names.
func exportedTopLevelNames(t *testing.T, path string) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}

	var names []string
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if !isExportedName(d.Name.Name) {
				continue
			}
			if d.Recv == nil || len(d.Recv.List) == 0 {
				names = append(names, d.Name.Name)
				continue
			}
			names = append(names, recvTypeName(d.Recv.List[0].Type)+"."+d.Name.Name)
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					for _, n := range s.Names {
						if isExportedName(n.Name) {
							names = append(names, n.Name)
						}
					}
				case *ast.TypeSpec:
					if isExportedName(s.Name.Name) {
						names = append(names, s.Name.Name)
					}
				}
			}
		}
	}
	sort.Strings(names)
	return names
}

// recvTypeName returns a method receiver's bare type name, unwrapping
// a pointer receiver (*Lock -> "Lock").
func recvTypeName(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return "" //coverage:ignore defensive: every receiver in this repo's own source is either T or *T; a generic or qualified receiver type doesn't occur
}

// isExportedName reports whether name starts with an uppercase
// letter — go/ast's own IsExported check, reimplemented locally to
// avoid pulling in the full go/doc dependency for one predicate.
func isExportedName(name string) bool {
	if name == "" { //coverage:ignore defensive: go/parser never yields an empty identifier name for a syntactically valid declaration
		return false
	}
	return unicode.IsUpper(rune(name[0]))
}
