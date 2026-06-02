package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec/branch"
)

// TestM0158_AC5_EveryBranchCellHasMatchingTest pins M-0158/AC-5:
// every cell in `branch.Rules()` has at least one matching test
// in the kernel's test set. "Matching" uses a multi-token
// heuristic against test-function names — the test-naming
// convention documented in `internal/workflows/spec/branch/spec.go`.
//
// Conventions accepted as a match:
//
//  1. Test name contains the cell's id (`branch-cell-1`,
//     `branch-cell-override-cherry-pick`, etc.). This is the
//     strongest convention — a paste of the id into the test
//     docstring or function name.
//  2. The cell's natural-key tokens. For corner-case cells N, a
//     curated keyword set per cell number maps to the
//     milestone-test naming the cell already carries in its
//     Code comment (e.g., `AICommitOnMainFires` for
//     branch-cell-4, `ForceReasonBypasses` for
//     branch-cell-override-preflight).
//
// Scope: the test set scanned is `internal/verb/`, `internal/check/`,
// `internal/cli/`, and `internal/policies/`. The scan reads test
// names from `go/ast` parses; it does not run the tests.
//
// The convention is documented in branch/spec.go's package doc;
// the milestones M-0102..M-0106 named their tests against
// behavioral fixtures, and this meta-test maps those names back
// to cells via the keyword set below. Adding a new branch cell
// without a paired test, OR renaming a test in a way that
// breaks the keyword match, fires this check.
func TestM0158_AC5_EveryBranchCellHasMatchingTest(t *testing.T) {
	t.Parallel()

	testNames := collectKernelTestNames(t)

	// Per-cell match-token set. The first list is "id-direct" —
	// if any test name contains the cell id, that's a match. The
	// second list is per-cell-id curated keywords — the test set
	// is matched if it contains ANY of the cell's keywords.
	keywords := map[string][]string{
		// "AICommitOnMain" intentionally NOT listed: it substring-
		// matches cell-4's TestIsolationEscape_AC1_AICommitOnMainFires,
		// which would let one test name satisfy two cells. Per the
		// M-0159 pre-fix patch round addressing the cross-cell
		// match-bleed flagged by the confidence-audit workflow. The
		// two remaining keywords (cell-1 specific) are the canonical
		// authorize-preflight test names from authorize_cmd_test.go.
		"branch-cell-1":                      {"NoBranch_NoRitualCurrent", "OnNonRitualBranch_NoBranch"},
		"branch-cell-2":                      {"BranchMissing_Refuses"},
		"branch-cell-3":                      {"ImplicitFromCurrent_AcceptsAndEmitsTrailer", "ImplicitRitualBranch_AcceptsAndRecords"},
		"branch-cell-4":                      {"AICommitOnMainFires", "FiresOnViolatingCommit"},
		"branch-cell-5":                      {"AICommitOnBoundBranchSilent", "SilentOnBoundBranchCommit"},
		"branch-cell-6":                      {"PausedScopeSilent", "AICommitOnBoundBranchPaused"},
		"branch-cell-7":                      {"AICommitOnDifferentRitualBranchFires", "DifferentRitualBranch"},
		"branch-cell-8":                      {"CherryPickReAuthorSilent"},
		"branch-cell-9":                      {"HumanMergeFirstParentSilent"},
		"branch-cell-10":                     {"ForceAmendedCommitSilent"},
		"branch-cell-11":                     {"NoScopeOpenedSilent"},
		"branch-cell-12":                     {"WorktreeBranchMismatchFires", "AC3_Worktree"},
		"branch-cell-override-preflight":     {"ForceReasonBypassesPreflight"},
		"branch-cell-override-cherry-pick":   {"CherryPickReAuthorSilent"},
		"branch-cell-override-force-amend":   {"ForceAmendedCommitSilent"},
		"branch-cell-override-f-nnnn-waiver": {}, // documented exception below
	}

	for _, cell := range branch.Rules() {
		// The f-nnnn-waiver cell's behavioral tests live in the
		// F-NNNN milestone family (outside E-0030 scope). Document
		// the exception in the cell's comment + here, then allow
		// no-match without failing.
		if cell.ID == "branch-cell-override-f-nnnn-waiver" {
			continue
		}

		idDirect := hasNameContaining(testNames, cell.ID)
		var keywordMatch bool
		for _, kw := range keywords[cell.ID] {
			if hasNameContaining(testNames, kw) {
				keywordMatch = true
				break
			}
		}
		if !idDirect && !keywordMatch {
			tokens := keywords[cell.ID]
			t.Errorf("M-0158/AC-5: no test found matching cell %q\n  expected at least one test name containing one of: %v\n  (or the literal id %q)", cell.ID, tokens, cell.ID)
		}
	}
}

// hasNameContaining reports whether any of the test names
// contains the substring. Case-sensitive (Go test names are).
func hasNameContaining(names []string, substr string) bool {
	for _, n := range names {
		if strings.Contains(n, substr) {
			return true
		}
	}
	return false
}

// collectKernelTestNames walks the test files under internal/verb/,
// internal/check/, internal/cli/check/, internal/cli/authorize/,
// internal/cli/integration/, and internal/policies/ and returns
// every `func TestX(t *testing.T)` name. Built via go/ast, so it
// captures test names declaratively without running anything.
func collectKernelTestNames(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)

	scanDirs := []string{
		filepath.Join(root, "internal", "verb"),
		filepath.Join(root, "internal", "check"),
		filepath.Join(root, "internal", "cli", "check"),
		filepath.Join(root, "internal", "cli", "authorize"),
		filepath.Join(root, "internal", "cli", "integration"),
		filepath.Join(root, "internal", "policies"),
	}

	var out []string
	for _, dir := range scanDirs {
		matches, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
		if err != nil {
			t.Fatalf("glob %s: %v", dir, err)
		}
		for _, path := range matches {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			for _, decl := range file.Decls {
				fd, ok := decl.(*ast.FuncDecl)
				if !ok || fd.Recv != nil {
					continue
				}
				name := fd.Name.Name
				if !strings.HasPrefix(name, "Test") {
					continue
				}
				out = append(out, name)
			}
		}
	}
	return out
}
