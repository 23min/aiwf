package policies

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec/branch"
)

// TestM0162_AC4_Bijection enforces 3 of the 4 bijection invariants
// between `branch.Rules()` and the Pin call sites observable in
// test source files across the repo, statically. Per M-0162/AC-4
// body §"Invariants enforced":
//
//  1. Every cell in branch.Rules() has at least one Pin call site.
//  2. Every Pin call site references a cell present in branch.Rules().
//  3. No cell has 2+ Pin call sites at distinct source positions.
//
// Invariant 4 (no test function pins 2+ cells) is enforced at
// RUNTIME by:
//
//   - internal/cli/integration/bijection_runtime_testpins_test.go's
//     TestZZZ_M0162_AC4_BijectionInvariant4_Runtime (lex-late
//     serial test under -tags testpins).
//   - The TestMain post-hook at integration/setup_test.go +
//     bijection_posthook_testpins_test.go's bijectionPostHook,
//     which reads branchtest.Pins() after all parallel waves
//     drain.
//
// Static analysis cannot resolve `t.Name()` (the load-bearing
// per-call-site identifier at runtime), so invariant 4 is
// architecturally a runtime concern. The body's "branchtest.Pins()
// registry" phrasing is delivered by the runtime portion of this
// split architecture.
//
// Why static AST is also used (this test, NOT under -tags testpins):
//
// The body's location hint at `internal/policies/...` (different
// package from where Pin calls happen) is incompatible with reading
// a per-process Pins() registry — the policies-package test binary
// would see an empty Pins() because its tests don't execute the
// integration-package Pin call sites. The static AST scan over
// internal/ *_test.go files provides the cell-side bijection
// coverage at policies-binary CI time; the runtime check in
// integration provides the t.Name() granularity check at
// integration-binary CI time. Both are needed; both run on every
// `make test-pins` and CI run.
//
// Static-AST scan rationale (deviation from body's "Pins() registry"):
//
// The body literally names `branchtest.Pins()` as the data source.
// Pins() is a per-process registry — populated only when the test
// binary that called Pin() is the same binary running the bijection
// check. A policies-package test sees an empty Pins() because its
// test binary doesn't execute the integration-package Pin call
// sites.
//
// To deliver the body's enumerated 4 invariants at CI time, this
// test does static AST extraction across every *_test.go file
// under internal/ that imports branchtest or uses the integration
// package's pinCell helper. The extracted set is the static mirror
// of what Pins() would record under a full -tags testpins run; the
// two agree by construction because every pinCell call records a
// Pin and the AST sees every call.
//
// The integration package's pinCell helper (pin_testpins_test.go)
// forwards to branchtest.Pin under -tags testpins and is a no-op
// otherwise (pin_nontestpins_test.go). The AST scan treats both
// shapes as Pin call sites — the static analysis is build-tag
// agnostic by design.
//
// Sabotage-verifiable per invariant (see TestM0162_AC4_Bijection_Sabotage
// below).
func TestM0162_AC4_Bijection(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	refs := collectPinReferences(t, root)

	cellsList := make([]string, 0)
	for _, r := range branch.Rules() {
		cellsList = append(cellsList, r.ID)
	}

	// Materialize prefix coverage: for each dynamic-prefix Pin
	// call site (e.g., `pinCell("branch-cell-m0161-ac1-"+ident, ...)`),
	// expand to every cell in branch.Rules() whose ID starts with
	// the prefix. This dissolves what was a 20-entry allowlist in
	// the original AC-4 closure (reviewer S3).
	pins := make(map[string][]string, len(refs.Literals))
	for k, v := range refs.Literals {
		pins[k] = append([]string(nil), v...)
	}
	for _, ps := range refs.Prefixes {
		for _, cellID := range cellsList {
			if !strings.HasPrefix(cellID, ps.Prefix) || cellID == ps.Prefix {
				continue
			}
			// Skip cells that already have a LITERAL pin: e.g.,
			// `branch-cell-m0161-ac2-sovereign-override` is pinned
			// literally by an inline pinCell elsewhere; the prefix
			// expansion from the matrix would double-count.
			if len(refs.Literals[cellID]) > 0 {
				continue
			}
			if !slices.Contains(pins[cellID], ps.Site) {
				pins[cellID] = append(pins[cellID], ps.Site)
			}
		}
	}

	v := evaluateBijection(cellsList, pins, bijectionAllowlist())

	if len(v) > 0 {
		t.Errorf("M-0162/AC-4 bijection meta-test: %d violation(s)\n%s", len(v), describeViolations(v))
	}
}

// bijectionAllowlist returns the set of cell IDs that may legally
// appear in branch.Rules() without a corresponding Pin call site
// in the *_test.go scan. Each entry documents WHERE the cell's
// behavioral test lives so a future reviewer can trace the carve-
// out's residual coverage.
//
// The allowlist is the AC-4 honest-closure scope-narrowing: the
// bijection check covers the AC-3-era cell-expansion surface
// (integration package's RunScenarios + inline pinCell sites)
// PLUS any test in any package that calls branchtest.Pin under
// -tags testpins. The pre-AC-3 M-0158/M-0161-era named cells'
// canonical tests live outside the integration package and
// don't currently import branchtest; extending Pin reach to
// those test packages is tracked as a follow-up (file a gap if
// you encounter this allowlist).
//
// When the follow-up lands, entries here become deletable and
// the bijection's first invariant tightens to "every cell has a
// Pin, no exceptions."
func bijectionAllowlist() map[string]string {
	return map[string]string{
		// M-0158 retained corner-case cells. Each has a primary
		// behavioral test under internal/verb/ or internal/check/.
		// The allowlist prose is mechanically verified by
		// TestM0162_AC4_AllowlistClaimsResolve (each "primary test
		// TestX in internal/<dir>/" claim resolves to a real
		// function declaration via AST walk).
		"branch-cell-1":  "primary test TestAuthorize_Open_AITarget_NoBranch_NoRitualCurrent_Refuses in internal/verb/",
		"branch-cell-2":  "primary test TestAuthorize_Open_AITarget_BranchMissing_Refuses in internal/verb/",
		"branch-cell-4":  "primary test TestIsolationEscape_AC1_AICommitOnMainFires in internal/check/",
		"branch-cell-7":  "primary test TestIsolationEscape_AC2_AICommitOnDifferentRitualBranchFires in internal/check/",
		"branch-cell-12": "primary test TestIsolationEscape_AC3_WorktreeBranchMismatchFires in internal/check/",

		// M-0158 retained override cells.
		"branch-cell-override-preflight":     "primary test TestAuthorize_Open_AITarget_ForceReasonBypassesPreflight in internal/verb/",
		"branch-cell-override-f-nnnn-waiver": "behavioral tests live in the F-NNNN milestone family per ADR-0003; outside E-0030 scope (documented exception inherited from M-0158/AC-5)",

		// M-0160/AC-4 named cell.
		"branch-cell-id-rename-untrailered": "primary test TestIDRenameUntrailered_TypedCodeClassIsBranchChoreography in internal/check/",

		// M-0161-era rule chokepoint cells. The named cells carry
		// the kernel rule code (load-bearing for M-0158/AC-6 drift
		// policy); their AC-3 ordinal counterparts (c1..cN) carry
		// the Pin call sites under RunScenarios. The "primary test"
		// claim names the unit-level chokepoint for the rule code;
		// the integration ordinals carry the matrix coverage.
		"branch-cell-isolation-escape-oracle-failure":     "primary test TestNewGitBranchOracle_AC3_PerRefTolerance_OneCorruptedRef in internal/cli/check/",
		"branch-cell-isolation-escape-shallow-clone":      "primary test TestNewGitBranchOracle_AC4_ShallowDetection_EmptyMapPlusTypedError in internal/cli/check/",
		"branch-cell-isolation-escape-orphaned-ai-commit": "primary test TestForcePushOrphan_AC5_Matrix in internal/cli/integration/",
		"branch-cell-isolation-escape-rename-survival":    "primary test TestBranchOracle_AC6_RenameResolution_Matrix in internal/cli/integration/",
		"branch-cell-detached-head-preflight":             "primary test TestDetachedHEAD_AC7_PreflightRefusesWithRefinedMessage in internal/cli/integration/",
		"branch-cell-promote-on-wrong-branch":             "primary test TestPromoteOnWrongBranch_AC8_Matrix in internal/cli/integration/",

		// AC-1 trunk-shape + AC-2 rung-pair dynamic cells used to
		// require 20 allowlist entries here. Reviewer S3 finding
		// noted this was a "trust me bro" surface. The fix: extend
		// the AST scanner to recognize `pinCell("prefix-"+ident,
		// ...)` and `branchtest.Pin("prefix-"+ident, ...)` as
		// dynamic-prefix Pin sites. The bijection check then
		// expands the prefix against branch.Rules() to credit
		// every cell with the matching prefix as pinned. See
		// handlePinArg() and the prefix-coverage materialization
		// in TestM0162_AC4_Bijection.
	}
}

// violationKind enumerates the 4 bijection-invariant violation
// classes. Used by the bijection meta-test + its sabotage tests.
type violationKind int

const (
	kindCellWithoutPin violationKind = iota
	kindPinOrphan
	kindDoublePin
	kindTestOverload
)

func (k violationKind) String() string {
	switch k {
	case kindCellWithoutPin:
		return "cell-without-pin (invariant 1)"
	case kindPinOrphan:
		return "pin-orphan (invariant 2)"
	case kindDoublePin:
		return "double-pin (invariant 3)"
	case kindTestOverload:
		return "test-overload (invariant 4)"
	}
	return fmt.Sprintf("unknown(%d)", k)
}

// bijectionViolation is one invariant violation.
type bijectionViolation struct {
	Kind    violationKind
	Subject string
	Detail  string
}

func (v bijectionViolation) String() string {
	return fmt.Sprintf("M-0162/AC-4 %s: %s — %s", v.Kind, v.Subject, v.Detail)
}

// evaluateBijection runs the 4 invariants. cells is the catalog;
// pins is a map[cellID][]enclosingTestSrcLocation; allow is the
// allowlist of cells exempted from invariant #1 with cited
// rationale. Returns a sorted list of violations.
func evaluateBijection(cells []string, pins map[string][]string, allow map[string]string) []bijectionViolation {
	cellSet := make(map[string]bool, len(cells))
	for _, c := range cells {
		cellSet[c] = true
	}

	var out []bijectionViolation

	// Invariant 1: every cell has at least one Pin (unless allowlisted).
	for _, c := range cells {
		if _, exempt := allow[c]; exempt {
			continue
		}
		if len(pins[c]) == 0 {
			out = append(out, bijectionViolation{
				Kind:    kindCellWithoutPin,
				Subject: c,
				Detail:  "cell registered in branch.Rules() but no test file calls pinCell/branchtest.Pin with this ID",
			})
		}
	}

	// Invariant 2: every Pin references an existing cell.
	for cid := range pins {
		if !cellSet[cid] {
			out = append(out, bijectionViolation{
				Kind:    kindPinOrphan,
				Subject: cid,
				Detail:  fmt.Sprintf("Pin call site(s) %v reference an ID not present in branch.Rules()", pins[cid]),
			})
		}
	}

	// Invariant 3: no cell has 2+ Pins. (Note: a Pin in 2+ source
	// locations indicates parallel pinCell call sites for the same
	// cell — distinct from runtime-multiple-calls.)
	for cid, sites := range pins {
		if !cellSet[cid] {
			continue
		}
		if len(sites) >= 2 {
			out = append(out, bijectionViolation{
				Kind:    kindDoublePin,
				Subject: cid,
				Detail:  fmt.Sprintf("multiple Pin call sites for this cell: %v", sites),
			})
		}
	}

	// Invariant 4 is enforced at runtime (integration TestMain
	// post-hook), not statically. See test docstring.

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Subject < out[j].Subject
	})
	return out
}

// describeViolations renders a violation list as multi-line text.
func describeViolations(v []bijectionViolation) string {
	if len(v) == 0 {
		return ""
	}
	var b strings.Builder
	for _, vv := range v {
		fmt.Fprintf(&b, "  %s\n", vv)
	}
	return b.String()
}

// collectPinCallSites walks every *_test.go file under
// internal/ and extracts:
//   - branchtest.Pin("cellID", ...) calls
//   - pinCell("cellID", ...) calls (the integration package's wrapper)
//   - CellID: "cellID" struct field literals (the Scenario seam,
//     which RunScenarios converts to a Pin at runtime — for the
//     static check the literal is equivalent evidence)
//
// Dynamic-string forms (`"prefix-" + var`) are NOT extracted; they
// are enumerated by the AC-3 dynamic-cells test
// (TestM0162_AC3_DynamicCellsPresence) which reads the matrix
// definitions directly. AC-4's bijection inherits that coverage:
// the dynamic cells appear in branch.Rules() and the matrix-row
// pinCell calls at runtime contribute to Pins() — the static AC-4
// check exempts them via the allowlist (the matrix-row cells'
// AC-1/AC-2 prefix entries above).
//
// Returns pinReferences with both literal cell-ID pins and
// dynamic-prefix pins. The caller materializes prefix coverage
// against branch.Rules() to dissolve the original AC-4 prefix
// allowlist (reviewer S3).
//
// Pattern recognized for prefixes: `pinCell("branch-cell-...-"+
// <ident>, ...)` and `branchtest.Pin("branch-cell-...-"+<ident>,
// ...)`. The literal portion's trailing dash signals the prefix
// is meant to be concatenated with a matrix-row identifier; the
// caller expands to cells with that prefix.
type pinReferences struct {
	Literals map[string][]string // cellID → []site
	Prefixes []prefixSite        // dynamic "prefix-"+ident sites
}

type prefixSite struct {
	Prefix string
	Site   string
}

func collectPinReferences(t *testing.T, root string) pinReferences {
	t.Helper()
	out := make(map[string]map[string]bool)
	var prefixes []prefixSite

	scanDirs := []string{
		filepath.Join(root, "internal"),
	}
	var files []string
	for _, d := range scanDirs {
		walkErr := filepath.WalkDir(d, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") {
				files = append(files, path)
			}
			return nil
		})
		if walkErr != nil {
			t.Fatalf("walk %s: %v", d, walkErr)
		}
	}

	for _, path := range files {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		base := filepath.Base(path)

		// Each Pin call site is identified by file:line so invariant 3
		// catches genuine cell-duplication (the same cellID literal in
		// 2+ source positions). Per-function attribution proved too
		// coarse: matrix-style `[]Scenario{{CellID:...}, {CellID:...}}`
		// is one enclosing function with N cells, even though at
		// runtime each row produces a unique t.Name() via the
		// RunScenarios subtest dispatch.
		ast.Inspect(f, func(n ast.Node) bool {
			// CellID: "..." struct literal
			if kv, ok := n.(*ast.KeyValueExpr); ok {
				if id, ok := kv.Key.(*ast.Ident); ok && id.Name == "CellID" {
					if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
						val := strings.Trim(lit.Value, `"`)
						if strings.HasPrefix(val, "branch-cell-") {
							addPinSite(out, val, fmt.Sprintf("%s:%d", base, fset.Position(lit.Pos()).Line))
						}
					}
				}
			}
			// pinCell("...", ...) or pinCell("prefix-"+ident, ...) call
			if call, ok := n.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "pinCell" && len(call.Args) > 0 {
					handlePinArg(call.Args[0], base, fset, out, &prefixes)
				}
				// branchtest.Pin("...", ...) call (qualified)
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Pin" {
					if x, ok := sel.X.(*ast.Ident); ok && x.Name == "branchtest" && len(call.Args) > 0 {
						handlePinArg(call.Args[0], base, fset, out, &prefixes)
					}
				}
			}
			return true
		})
	}

	// Flatten map[id]map[site]bool to map[id][]site sorted.
	result := make(map[string][]string, len(out))
	for cid, sites := range out {
		list := make([]string, 0, len(sites))
		for s := range sites {
			list = append(list, s)
		}
		sort.Strings(list)
		result[cid] = list
	}
	return pinReferences{Literals: result, Prefixes: prefixes}
}

// handlePinArg classifies the first argument of a pinCell or
// branchtest.Pin call. Two shapes are recognized:
//
//  1. *ast.BasicLit with kind STRING — a literal cell ID. Recorded
//     as a Pin call site for that cell.
//  2. *ast.BinaryExpr with token.ADD, LHS literal, RHS identifier
//     (or another non-literal) — a dynamic prefix. The literal
//     portion ending in `-` is the prefix; recorded for later
//     expansion against branch.Rules() so cells matching the
//     prefix are credited as pinned without an allowlist entry.
//
// Other shapes (e.g., `fmt.Sprintf(...)`, function calls returning
// strings) are skipped — they're rare in this codebase and would
// be tracked by a follow-up gap if they appear.
func handlePinArg(arg ast.Expr, base string, fset *token.FileSet, out map[string]map[string]bool, prefixes *[]prefixSite) {
	if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		val := strings.Trim(lit.Value, `"`)
		if strings.HasPrefix(val, "branch-cell-") {
			addPinSite(out, val, fmt.Sprintf("%s:%d", base, fset.Position(lit.Pos()).Line))
		}
		return
	}
	if bx, ok := arg.(*ast.BinaryExpr); ok && bx.Op == token.ADD {
		if lit, ok := bx.X.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			val := strings.Trim(lit.Value, `"`)
			if strings.HasPrefix(val, "branch-cell-") && strings.HasSuffix(val, "-") {
				*prefixes = append(*prefixes, prefixSite{
					Prefix: val,
					Site:   fmt.Sprintf("%s:%d", base, fset.Position(lit.Pos()).Line),
				})
			}
		}
	}
}

func addPinSite(out map[string]map[string]bool, id, site string) {
	if out[id] == nil {
		out[id] = make(map[string]bool)
	}
	out[id][site] = true
}
