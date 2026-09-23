package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/workflows/spec"
)

// Every illegal transition must reach the negative driver exactly once.
func TestM0125_AC4_IllegalCellsAllCovered(t *testing.T) {
	t.Parallel()
	want := map[string]int{}
	for _, rule := range spec.Rules() {
		if rule.Outcome == spec.OutcomeIllegal {
			want[cellKey(rule)]++
		}
	}
	got := map[string]int{}
	for _, tc := range enumerateIllegalCases(t) {
		got[cellKey(tc.rule)]++
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("negative-driver coverage (-want +got):\n%s", diff)
	}
}

func TestM0125_AC4_IllegalSubtestNamesUnique(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for _, tc := range enumerateIllegalCases(t) {
		if seen[tc.name] {
			t.Errorf("duplicate negative subtest name %q", tc.name)
		}
		seen[tc.name] = true
	}
}

func TestM0125_AC4_NoTestingSkipInNegativeDrivers(t *testing.T) {
	t.Parallel()

	driverFiles := []string{
		"m0125_negative_driver_test.go",
	}

	fset := token.NewFileSet()
	for _, file := range driverFiles {
		astFile, err := parser.ParseFile(fset, file, nil, parser.AllErrors)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		ast.Inspect(astFile, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			method := sel.Sel.Name
			if method != "Skip" && method != "Skipf" && method != "SkipNow" {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != "t" {
				return true
			}
			pos := fset.Position(call.Pos())
			t.Errorf("%s:%d: t.%s bypasses a negative driver's assertion", file, pos.Line, method)
			return true
		})
	}
}
