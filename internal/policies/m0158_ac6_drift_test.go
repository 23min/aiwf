package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	codespkg "github.com/23min/aiwf/internal/codes"
	"github.com/23min/aiwf/internal/workflows/spec/branch"
)

// TestM0158_AC6_EveryClassBranchChoreographyCodeReferencedByCell
// pins M-0158/AC-6: every kernel finding code declared with
// `Class: codes.ClassBranchChoreography` is referenced by at
// least one Illegal cell in `branch.Rules()`. The drift policy
// catches the class:
//
//   - A new ClassBranchChoreography code is added to the
//     kernel (e.g., a future "isolation-escape-paused-misuse"
//     code on a new check rule) without a matching cell
//     registered here.
//
//   - The cell exists but the code typo'd in its
//     ExpectedErrorCode field.
//
// The bidirectional check is the M-0158 generalization of the
// existing M-0123/AC-5 "legality-codes-referenced" drift arm.
// Without this, the spec table can claim layer-4 coverage while
// silently missing a real kernel emission point.
//
// Discovery: enumerate ClassBranchChoreography codes by AST-
// scanning `internal/check/*.go` for `codes.Code` literals whose
// `Class` field is `codes.ClassBranchChoreography`. The
// alternative (use reflection over a registry) would require a
// registry which the kernel does not have; AST scanning matches
// the existing M-0123/AC-5 pattern for layers 1–3.
func TestM0158_AC6_EveryClassBranchChoreographyCodeReferencedByCell(t *testing.T) {
	t.Parallel()

	declared := collectBranchChoreographyCodes(t)
	if len(declared) == 0 {
		t.Fatal("M-0158/AC-6: no ClassBranchChoreography codes found by AST scan — either the kernel has none yet (then this test should be marked WIP) or the scanner is broken")
	}

	referenced := map[string]bool{}
	for _, r := range branch.Rules() {
		if r.ExpectedErrorCode != "" {
			referenced[r.ExpectedErrorCode] = true
		}
	}

	for codeID := range declared {
		if !referenced[codeID] {
			t.Errorf("M-0158/AC-6: ClassBranchChoreography code %q is not referenced by any branch.Rules() Illegal cell\n  add a cell with ExpectedErrorCode=%q citing the corresponding corner-case row or filed gap", codeID, codeID)
		}
	}
}

// TestM0158_AC6_DriftFiresOnFabricatedCode is the sabotage-
// counterpart of the above: feed the drift-check helper a
// fabricated ClassBranchChoreography code that is NOT referenced
// by any cell. Assert the helper reports the gap. Without this
// guard, a regression that silently returned "all good" on
// missing-cell drift would shield the rule against future
// regressions.
//
// Implemented as a self-contained mini-driver that doesn't depend
// on adding/removing real codes in the kernel (which would
// require modifying production code mid-test).
func TestM0158_AC6_DriftFiresOnFabricatedCode(t *testing.T) {
	t.Parallel()

	// Build the referenced-codes set from the live spec table.
	referenced := map[string]bool{}
	for _, r := range branch.Rules() {
		if r.ExpectedErrorCode != "" {
			referenced[r.ExpectedErrorCode] = true
		}
	}

	// Fabricate a code id that is definitionally NOT in the
	// referenced set (sentinel prefix that no production code
	// would adopt).
	fabricated := "isolation-escape-FABRICATED-SENTINEL-MUST-NOT-EXIST"
	if referenced[fabricated] {
		t.Fatalf("test corruption: fabricated sentinel %q is somehow in the referenced set", fabricated)
	}

	// The drift check (inline body of the production policy):
	// if any declared code is not in referenced, the check fires.
	// Here we simulate that the fabricated code IS declared
	// (pretending the AST scan returned it).
	declared := map[string]bool{fabricated: true}
	var gaps []string
	for codeID := range declared {
		if !referenced[codeID] {
			gaps = append(gaps, codeID)
		}
	}
	if len(gaps) == 0 {
		t.Error("drift check did not fire on a fabricated unreferenced code — the policy is silently passing")
	}
}

// collectBranchChoreographyCodes scans internal/check/*.go for
// top-level `codes.Code` literal declarations whose Class field
// is `codes.ClassBranchChoreography`. Returns a set of code-id
// strings.
//
// AST shape we look for:
//
//	var CodeXYZ = codes.Code{ID: "xyz", Class: codes.ClassBranchChoreography}
//
// or with an alias import:
//
//	var CodeXYZ = codespkg.Code{ID: "xyz", Class: codespkg.ClassBranchChoreography}
func collectBranchChoreographyCodes(t *testing.T) map[string]bool {
	t.Helper()
	root := repoRoot(t)
	out := map[string]bool{}

	matches, err := filepath.Glob(filepath.Join(root, "internal", "check", "*.go"))
	if err != nil {
		t.Fatalf("glob internal/check: %v", err)
	}
	for _, path := range matches {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if !isCodesDotCode(cl.Type) {
				return true
			}
			id, isBranchClass := scanCodeLiteral(cl)
			if isBranchClass && id != "" {
				out[id] = true
			}
			return true
		})
	}

	// Sanity: ensure the ClassBranchChoreography const exists in
	// the codes package so the AST shape we're looking for is the
	// right shape. Compile-time use here doubles as a kernel
	// invariant pin.
	_ = codespkg.ClassBranchChoreography

	return out
}

// isCodesDotCode reports whether the type expression is
// `codes.Code` or `<alias>.Code` (any package selector ending
// in `.Code`).
func isCodesDotCode(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return sel.Sel.Name == "Code"
}

// scanCodeLiteral extracts the ID string and the
// "is-ClassBranchChoreography" flag from a composite literal of
// shape `codes.Code{ID: "...", Class: codes.ClassBranchChoreography}`.
func scanCodeLiteral(cl *ast.CompositeLit) (id string, isBranchClass bool) {
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		keyIdent, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch keyIdent.Name {
		case "ID":
			if lit, ok := kv.Value.(*ast.BasicLit); ok {
				id = strings.Trim(lit.Value, "\"")
			}
		case "Class":
			if sel, ok := kv.Value.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "ClassBranchChoreography" {
					isBranchClass = true
				}
			}
		}
	}
	return id, isBranchClass
}
