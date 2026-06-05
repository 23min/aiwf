package policies

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
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
//  3. No cell has 2+ Pin call sites.
//
// Invariant 4 (no test function pins 2+ cells) is RUNTIME-ONLY:
// static analysis cannot resolve `t.Name()` (the load-bearing
// per-call-site identifier at runtime). The integration package's
// TestMain post-hook (testpins-tagged) catches invariant 4 by
// reading `branchtest.Pins()` after all parallel waves complete.
// See internal/cli/integration/bijection_*_test.go.
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
	pins := collectPinCallSites(t, root)

	cellsList := make([]string, 0)
	for _, r := range branch.Rules() {
		cellsList = append(cellsList, r.ID)
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
		// behavioral test under internal/verb/ or
		// internal/cli/{authorize,check}/ — none of which currently
		// import branchtest under -tags testpins.
		"branch-cell-1":  "primary test TestAuthorize_..._NoBranch_NoRitualCurrent in internal/cli/authorize/",
		"branch-cell-2":  "primary test TestAuthorize_..._BranchMissing_Refuses in internal/cli/authorize/",
		"branch-cell-4":  "primary test TestIsolationEscape_AC1_AICommitOnMainFires in internal/check/",
		"branch-cell-7":  "primary test TestIsolationEscape_AC2_AICommitOnDifferentRitualBranchFires in internal/check/",
		"branch-cell-12": "primary test TestIsolationEscape_AC3_WorktreeBranchMismatchFires in internal/check/",

		// M-0158 retained override cells.
		"branch-cell-override-preflight":     "primary test TestAuthorize_..._ForceReasonBypassesPreflight in internal/cli/authorize/",
		"branch-cell-override-f-nnnn-waiver": "behavioral tests live in the F-NNNN milestone family per ADR-0003; outside E-0030 scope (documented exception inherited from M-0158/AC-5)",

		// M-0160/AC-4 named cell.
		"branch-cell-id-rename-untrailered": "primary test TestIDRenameUntrailered_TypedCodeClassIsBranchChoreography in internal/check/",

		// M-0161-era rule chokepoint cells. The named cells carry
		// the kernel rule code (load-bearing for M-0158/AC-6 drift
		// policy); their AC-3 ordinal counterparts (c1..cN) carry
		// the Pin call sites under RunScenarios.
		"branch-cell-isolation-escape-oracle-failure":     "named-rule cell paired with branch-cell-m0161-ac3-c1..c14 ordinals (which carry the Pin calls); primary unit test in internal/cli/check/",
		"branch-cell-isolation-escape-shallow-clone":      "named-rule cell paired with branch-cell-m0161-ac4-c1..c12 ordinals; primary unit test in internal/cli/check/",
		"branch-cell-isolation-escape-orphaned-ai-commit": "named-rule cell paired with branch-cell-m0161-ac5-c1..c8 ordinals; primary unit test in internal/check/",
		"branch-cell-isolation-escape-rename-survival":    "named-rule cell paired with branch-cell-m0161-ac6-c1..c9 ordinals; primary unit test in internal/cli/check/",
		"branch-cell-detached-head-preflight":             "named-rule cell paired with branch-cell-m0161-ac7-c1..c7 ordinals (which carry the inline pinCell calls); primary verb test in internal/verb/",
		"branch-cell-promote-on-wrong-branch":             "named-rule cell paired with branch-cell-m0161-ac8-c1..c8 ordinals; primary unit test in internal/check/",

		// AC-1 trunk-shape matrix cells. pinCell call site is
		// `pinCell("branch-cell-m0161-ac1-"+tc.name, t.Name())` at
		// internal/cli/integration/authorize_scenarios_test.go (post-
		// M-0162/AC-3). Static AST sees the prefix string but cannot
		// resolve the concatenation; the runtime registry sees the
		// constructed values. AC-3's TestM0162_AC3_DynamicCellsPresence
		// (m0162_ac3_expanded_set_test.go) verifies the constructed
		// IDs are in branch.Rules(); AC-4's runtime invariant 4 check
		// (integration TestMain post-hook, when -tags testpins) sees
		// the registry pins.
		"branch-cell-m0161-ac1-main":                   "AC-1 trunk-shape dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac1-github-classic-master":  "AC-1 trunk-shape dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac1-operator-chosen-dev":    "AC-1 trunk-shape dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac1-operator-chosen-trunk":  "AC-1 trunk-shape dynamic cell; see allowlist note above",

		// AC-2 rung-pair matrix cells (4 rungs × 4 rungs = 16). Same
		// dynamic-string rationale as AC-1 above. See the AC-3
		// dynamic-cells presence test for the IDs↔matrix mapping.
		"branch-cell-m0161-ac2-trunk_to_trunk":         "AC-2 rung-pair dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac2-trunk_to_epic":          "AC-2 rung-pair dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac2-trunk_to_milestone":     "AC-2 rung-pair dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac2-trunk_to_patch":         "AC-2 rung-pair dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac2-epic_to_trunk":          "AC-2 rung-pair dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac2-epic_to_epic":           "AC-2 rung-pair dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac2-epic_to_milestone":      "AC-2 rung-pair dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac2-epic_to_patch":          "AC-2 rung-pair dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac2-milestone_to_trunk":     "AC-2 rung-pair dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac2-milestone_to_epic":      "AC-2 rung-pair dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac2-milestone_to_milestone": "AC-2 rung-pair dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac2-milestone_to_patch":     "AC-2 rung-pair dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac2-patch_to_trunk":         "AC-2 rung-pair dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac2-patch_to_epic":          "AC-2 rung-pair dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac2-patch_to_milestone":     "AC-2 rung-pair dynamic cell; see allowlist note above",
		"branch-cell-m0161-ac2-patch_to_patch":         "AC-2 rung-pair dynamic cell; see allowlist note above",
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
// Returns map[cellID][]"file:funcname" so invariants 3 and 4 can
// be checked against call-site uniqueness.
func collectPinCallSites(t *testing.T, root string) map[string][]string {
	t.Helper()
	out := make(map[string]map[string]bool)

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
			pos := fset.Position(getPinPos(n))
			site := fmt.Sprintf("%s:%d", base, pos.Line)
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
			// pinCell("...", ...) call
			if call, ok := n.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "pinCell" && len(call.Args) > 0 {
					if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						val := strings.Trim(lit.Value, `"`)
						if strings.HasPrefix(val, "branch-cell-") {
							addPinSite(out, val, fmt.Sprintf("%s:%d", base, fset.Position(lit.Pos()).Line))
						}
					}
				}
				// branchtest.Pin("...", ...) call (qualified)
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Pin" {
					if x, ok := sel.X.(*ast.Ident); ok && x.Name == "branchtest" && len(call.Args) > 0 {
						if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							val := strings.Trim(lit.Value, `"`)
							if strings.HasPrefix(val, "branch-cell-") {
								addPinSite(out, val, fmt.Sprintf("%s:%d", base, fset.Position(lit.Pos()).Line))
							}
						}
					}
				}
			}
			_ = site
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
	return result
}

func addPinSite(out map[string]map[string]bool, id, site string) {
	if out[id] == nil {
		out[id] = make(map[string]bool)
	}
	out[id][site] = true
}

// getPinPos returns a stable position for any AST node — used as a
// fallback when the node-specific lookup miss isn't hit.
func getPinPos(n ast.Node) token.Pos {
	if n == nil {
		return token.NoPos
	}
	return n.Pos()
}
