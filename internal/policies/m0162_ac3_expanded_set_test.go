package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec/branch"
)

// TestM0162_AC3_CellPresence pins M-0162/AC-3's cell-presence
// claim: every cell ID referenced by an E2E Scenario CellID
// literal or an inline pinCell() call in internal/cli/integration/
// is present in branch.Rules().
//
// The test parses every *_test.go file under internal/cli/integration/
// for two patterns:
//
//  1. `CellID: "branch-cell-..."` in a Scenario struct literal.
//  2. `pinCell("branch-cell-...", ...)` standalone call.
//
// For every literal-string cell ID extracted, the test asserts the
// ID resolves to an entry in branch.Rules(). Dynamic-string forms
// (`pinCell("branch-cell-"+name, ...)`) are skipped — those are
// enumerated explicitly in rules_m0162_ac3.go's generator script
// and rely on the test below (TestM0162_AC3_DynamicCellsPresence)
// for coverage.
//
// Sabotage-verifiable:
//   - Remove a cell from branch.Rules() that an E2E references →
//     this test names the missing cell ID + the source file.
//   - Add a CellID to an E2E that has no matching cell in Rules()
//     → same failure mode.
func TestM0162_AC3_CellPresence(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	integrationDir := filepath.Join(root, "internal", "cli", "integration")

	// Collect cell IDs referenced by E2E tests.
	refs := collectE2ECellRefs(t, integrationDir)
	if len(refs) == 0 {
		t.Fatal("M-0162/AC-3: no cell IDs found in E2E tests — has the seam been undone?")
	}

	// Build the set of cells present in branch.Rules().
	present := make(map[string]bool)
	for _, r := range branch.Rules() {
		present[r.ID] = true
	}

	for cellID, sources := range refs {
		if !present[cellID] {
			t.Errorf("M-0162/AC-3: cell %q referenced by E2E test(s) %v is ABSENT from branch.Rules()", cellID, sources)
		}
	}
}

// TestM0162_AC3_DynamicCellsPresence pins the dynamic-string
// pinCell call sites (`pinCell("branch-cell-..."+name, ...)`) by
// asserting the enumerated AC-1 trunk-name shapes and AC-2 rung-
// pair matrix rows have cells in branch.Rules(). These are the
// matrix-row cells that can't be statically extracted from the
// source — the cell IDs are constructed at runtime from the matrix
// rows themselves. Adding a new trunk shape or rung-pair without
// adding the corresponding cell to ac3ExpandedCells() fires this
// test.
func TestM0162_AC3_DynamicCellsPresence(t *testing.T) {
	t.Parallel()

	present := make(map[string]bool)
	for _, r := range branch.Rules() {
		present[r.ID] = true
	}

	// AC-1: 4 trunk-name shapes (TestAuthorize_AC1_NonMainTrunkNames_Accept).
	ac1Shapes := []string{"main", "github-classic-master", "operator-chosen-dev", "operator-chosen-trunk"}
	for _, s := range ac1Shapes {
		id := "branch-cell-m0161-ac1-" + s
		if !present[id] {
			t.Errorf("M-0162/AC-3: AC-1 trunk-shape cell %q ABSENT from branch.Rules()", id)
		}
	}

	// AC-2: 16 rung-pair cells (TestAuthorize_AC2_RungPair_Matrix).
	rungs := []string{"trunk", "epic", "milestone", "patch"}
	for _, c := range rungs {
		for _, ta := range rungs {
			id := "branch-cell-m0161-ac2-" + c + "_to_" + ta
			if !present[id] {
				t.Errorf("M-0162/AC-3: AC-2 rung-pair cell %q ABSENT from branch.Rules()", id)
			}
		}
	}
}

// TestM0162_AC3_NoEmptySuffixCells pins the M-0162/AC-3 reviewer
// S11 finding (dead-cell bug): a regen-script pre-fix accidentally
// emitted two cells with bare-prefix IDs (`branch-cell-m0161-ac1-`
// and `branch-cell-m0161-ac2-`) because the inline-pinCell regex
// matched the literal prefix in `pinCell("branch-cell-m0161-ac1-"
// +tc.name, ...)` without recognizing the concatenation. The cells
// existed but no Pin call site ever referenced them (the dynamic
// concatenation always produced ...-main, ...-trunk_to_epic, etc.),
// violating AC-4's bijection invariant #1 ("every cell has at least
// one Pin") silently.
//
// This guard scans branch.Rules() for any cell ID matching the
// "<prefix>-" shape with an empty suffix and fires loudly. The
// regen script at scripts/m0162-build-ac3-cells.py was fixed to
// skip prefix-only literals (those followed by `+`), but the guard
// remains as a structural anchor against a future regression.
//
// Sabotage-verifiable: edit rules_m0162_ac3.go to re-add an entry
// with `ID: "branch-cell-m0161-ac1-"` and this test fires naming
// the offending cell.
func TestM0162_AC3_NoEmptySuffixCells(t *testing.T) {
	t.Parallel()

	emptySuffix := regexp.MustCompile(`^branch-cell-[a-z0-9-]+-$`)
	for _, r := range branch.Rules() {
		if emptySuffix.MatchString(r.ID) {
			t.Errorf("M-0162/AC-3 (reviewer S11): cell %q has empty suffix; likely from a regen-script extraction of a `\"prefix-\"+var` shape that should have been skipped. See scripts/m0162-build-ac3-cells.py Pass 2 for the guard.", r.ID)
		}
	}
}

// collectE2ECellRefs walks every *_test.go in dir, parses
// the Go AST, and extracts:
//   - CellID: "branch-cell-..." literal in struct literals.
//   - pinCell("branch-cell-...", ...) call arguments (literal only).
//
// Returns map[cellID][]sourceFiles for diagnostic context.
func collectE2ECellRefs(t *testing.T, dir string) map[string][]string {
	t.Helper()
	refs := make(map[string]map[string]bool)

	matches, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatalf("glob %s: %v", dir, err)
	}

	cellIDRe := regexp.MustCompile(`^branch-cell-`)
	for _, path := range matches {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		base := filepath.Base(path)
		ast.Inspect(f, func(n ast.Node) bool {
			// CellID: "branch-cell-..." in struct literal.
			if kv, ok := n.(*ast.KeyValueExpr); ok {
				if id, ok := kv.Key.(*ast.Ident); ok && id.Name == "CellID" {
					if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
						val := strings.Trim(lit.Value, `"`)
						if cellIDRe.MatchString(val) {
							addRef(refs, val, base)
						}
					}
				}
			}
			// pinCell("branch-cell-...", ...) call.
			if call, ok := n.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "pinCell" && len(call.Args) > 0 {
					if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						val := strings.Trim(lit.Value, `"`)
						if cellIDRe.MatchString(val) {
							addRef(refs, val, base)
						}
					}
				}
			}
			return true
		})
	}

	// Flatten to map[id][]sourcesSorted
	out := make(map[string][]string, len(refs))
	for id, sources := range refs {
		list := make([]string, 0, len(sources))
		for s := range sources {
			list = append(list, s)
		}
		out[id] = list
	}
	return out
}

func addRef(refs map[string]map[string]bool, id, source string) {
	if refs[id] == nil {
		refs[id] = make(map[string]bool)
	}
	refs[id][source] = true
}
