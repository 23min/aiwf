package policies

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec/branch"
)

// TestM0158_AC1_BranchPackageExistsWithRulesAndAntiRules pins
// M-0158/AC-1: the `internal/workflows/spec/branch/` package
// exists and exports `Rules()` / `AntiRules()` returning
// `[]spec.Rule` / `[]spec.AntiRule`. These two accessors are the
// public entry points consumer drift tests union with the parent
// `spec.Rules()` / `spec.AntiRules()`.
//
// Structural pin: the compile-time use in this test file proves
// both accessors exist with the documented signature. A
// regression that renamed or unexported either accessor would
// fail compilation, surfacing the drift at the build step.
func TestM0158_AC1_BranchPackageExistsWithRulesAndAntiRules(t *testing.T) {
	t.Parallel()

	rules := branch.Rules()
	antirules := branch.AntiRules()

	// Cycle 1 scaffolds the accessors with empty returns. Cycles 2
	// and 3 populate them. The non-nil sentinel pins the contract:
	// the accessors return slices (possibly empty), never nil
	// pointers — consumers can range without nil-check.
	if rules == nil {
		t.Error("branch.Rules() returned nil; want non-nil (possibly empty) slice")
	}
	if antirules == nil {
		t.Error("branch.AntiRules() returned nil; want non-nil (possibly empty) slice")
	}
}

// TestM0158_AC7_RulesAndAntiRulesDeterministicallyOrdered pins
// M-0158/AC-7: both `branch.Rules()` and `branch.AntiRules()`
// return their entries sorted by cell id. Determinism matters
// for renderer / diff consumers (stable output across runs) and
// for the meta-tests that build per-id lookup maps.
//
// Cycle 1 scaffolds an empty catalog (trivially sorted). The
// test is structural: once Cycles 2 + 3 populate the catalogs,
// the same assertion catches an unsorted regression.
func TestM0158_AC7_RulesAndAntiRulesDeterministicallyOrdered(t *testing.T) {
	t.Parallel()

	rules := branch.Rules()
	ruleIDs := make([]string, len(rules))
	for i, r := range rules {
		ruleIDs[i] = r.ID
	}
	if !sort.StringsAreSorted(ruleIDs) {
		t.Errorf("branch.Rules() not sorted by ID; got %v", ruleIDs)
	}

	antirules := branch.AntiRules()
	antiIDs := make([]string, len(antirules))
	for i, ar := range antirules {
		antiIDs[i] = ar.ID
	}
	if !sort.StringsAreSorted(antiIDs) {
		t.Errorf("branch.AntiRules() not sorted by ID; got %v", antiIDs)
	}
}

// TestM0158_AC8_PackageDocCitesADR0011AndConvention pins
// M-0158/AC-8: the branch package's doc comment explains the
// layer-4 carve-out, cites ADR-0011 explicitly, and names the
// cell-id-to-test-name convention.
//
// Structural assertion: parse the package's doc.go (or spec.go)
// via go/ast, extract the package-level doc, and assert the
// load-bearing tokens are present. Heading-scoped per CLAUDE.md
// §"Substring assertions are not structural assertions" — the
// markers must live INSIDE the package doc, not in a function
// comment.
func TestM0158_AC8_PackageDocCitesADR0011AndConvention(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	path := filepath.Join(root, "internal", "workflows", "spec", "branch", "spec.go")

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if file.Doc == nil {
		t.Fatal("branch package's spec.go has no package-level doc comment")
	}
	doc := file.Doc.Text()

	wantMarkers := []struct {
		name   string
		marker string
	}{
		{"layer-4 carve-out", "layer 4"},
		{"ADR-0011 citation", "ADR-0011"},
		{"cell-id convention (branch-cell-N)", "branch-cell-"},
		{"test-naming convention", "Test-naming"},
	}
	for _, w := range wantMarkers {
		if !strings.Contains(doc, w.marker) {
			t.Errorf("package doc must name %s (substring %q)", w.name, w.marker)
		}
	}
}
