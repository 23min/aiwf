package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec"
)

// TestM0125_AC4_IllegalCellsAllCovered asserts that every Illegal
// cell in spec.Rules() is enumerated by the union of AC-2's
// enumerateVerbTimeIllegalCases and AC-3's enumerateCheckTimeIllegalCases.
//
// This is M-0125's negative-coverage commitment from the milestone
// body ("Every illegal cell in Rules() has at least one negative
// test") rendered as a mechanical drift gate. The chokepoint catches:
//
//  1. A new Illegal cell with a RejectionLayer outside
//     {VerbTime, CheckTime} (zero value, future enum addition) —
//     neither enumerator picks it up.
//  2. An enumerator refactor that accidentally filters cells out —
//     the explicit set-membership check surfaces the drop here
//     rather than as silent missing coverage in CI.
//  3. A cell whose Outcome was retyped from Legal to Illegal but
//     missed the corresponding RejectionLayer assignment.
//
// Identity is cellKey (Kind, FromState, Verb, preconditions-sig) —
// the same disambiguator M-0124's positive coverage meta-test uses.
// The structural invariants on Illegal cells (RejectionLayer non-zero,
// ExpectedErrorCode non-empty) are enforced separately by M-0123's
// AC-2 drift policies; AC-4 does not duplicate them.
func TestM0125_AC4_IllegalCellsAllCovered(t *testing.T) {
	t.Parallel()

	enumerated := enumeratedIllegalCellKeys(t)
	for _, rule := range spec.Rules() {
		if rule.Outcome != spec.OutcomeIllegal {
			continue
		}
		key := cellKey(rule)
		if _, ok := enumerated[key]; !ok {
			t.Errorf("Illegal cell missing from negative-driver enumeration union: %s\n  rule: kind=%s from=%s verb=%s rejection_layer=%v preconditions=%+v",
				key, rule.Kind, rule.FromState, rule.Verb, rule.RejectionLayer, rule.Preconditions)
		}
	}
}

// TestM0125_AC4_NoExtraIllegalEnumerations is the converse of the
// all-covered check: every enumerated case (in either enumerator)
// corresponds to a real Illegal cell in spec.Rules(). Catches the
// case where one of the enumerators accidentally includes non-Illegal
// cells (e.g. an Outcome filter inversion) or fabricates cases not
// grounded in the spec.
func TestM0125_AC4_NoExtraIllegalEnumerations(t *testing.T) {
	t.Parallel()

	illegalKeys := map[string]bool{}
	for _, rule := range spec.Rules() {
		if rule.Outcome == spec.OutcomeIllegal {
			illegalKeys[cellKey(rule)] = true
		}
	}
	for _, c := range enumerateVerbTimeIllegalCases(t) {
		key := cellKey(c.rule)
		if !illegalKeys[key] {
			t.Errorf("verb-time enumeration includes case not grounded in an Illegal spec cell: %s (case name %q)", key, c.name)
		}
	}
	for _, c := range enumerateCheckTimeIllegalCases(t) {
		key := cellKey(c.rule)
		if !illegalKeys[key] {
			t.Errorf("check-time enumeration includes case not grounded in an Illegal spec cell: %s (case name %q)", key, c.name)
		}
	}
}

// TestM0125_AC4_IllegalSubtestNamesUnique asserts illegalCaseName
// produces unique names across the union of verb-time and check-time
// enumerations. t.Run on a duplicate name silently shadows the second
// subtest under most test runners — the result is reported but the
// disambiguation is lost. illegalCaseName composes
// (Kind, FromState, Verb) with preconditionSignature; this test
// confirms the signature is sufficient.
func TestM0125_AC4_IllegalSubtestNamesUnique(t *testing.T) {
	t.Parallel()

	seen := map[string]int{}
	for _, c := range enumerateVerbTimeIllegalCases(t) {
		seen[c.name]++
	}
	for _, c := range enumerateCheckTimeIllegalCases(t) {
		seen[c.name]++
	}
	for name, count := range seen {
		if count > 1 {
			t.Errorf("illegalCaseName collision: name %q assigned to %d distinct cases — case-name disambiguation insufficient", name, count)
		}
	}
}

// TestM0125_AC4_EnumerationsMutuallyExclusive asserts no Illegal cell
// appears in both the verb-time and check-time enumerators. The split
// is by RejectionLayer (VerbTime XOR CheckTime); a cell appearing in
// both means either a spec inconsistency (a cell with multiple
// RejectionLayers, which the type system disallows) or an enumerator
// filter bug that lets non-matching cells through.
//
// Mutual exclusion is the Illegal-side analogue of M-0124's "every
// case has a target" assertion: it pins the partition the enumerators
// rely on, so a future refactor can't merge them silently.
func TestM0125_AC4_EnumerationsMutuallyExclusive(t *testing.T) {
	t.Parallel()

	verbTime := map[string]spec.Rule{}
	for _, c := range enumerateVerbTimeIllegalCases(t) {
		verbTime[cellKey(c.rule)] = c.rule
	}
	for _, c := range enumerateCheckTimeIllegalCases(t) {
		key := cellKey(c.rule)
		if prior, dup := verbTime[key]; dup {
			t.Errorf("Illegal cell %q is in BOTH enumerations:\n  verb-time copy: %+v\n  check-time copy: %+v\nthe RejectionLayer split should partition the Illegal set",
				key, prior, c.rule)
		}
	}
}

// enumeratedIllegalCellKeys returns the union of cell keys from the
// verb-time and check-time enumerators. Used by the all-covered
// assertion. Parallels M-0124's enumeratedCellKeys helper.
func enumeratedIllegalCellKeys(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	verbCases := enumerateVerbTimeIllegalCases(t)
	for i := range verbCases {
		out[cellKey(verbCases[i].rule)] = true
	}
	checkCases := enumerateCheckTimeIllegalCases(t)
	for i := range checkCases {
		out[cellKey(checkCases[i].rule)] = true
	}
	return out
}

// TestM0125_AC4_NoTestingSkipInNegativeDrivers asserts that the
// per-cell loops in M-0125's verb-time and check-time negative
// drivers do not call any of t.Skip / t.Skipf / t.SkipNow. Impl-gap
// cells must dispatch through a staleness-asserting helper
// (runImplGapStalenessVerbTime for AC-2, or the isImplGap branch
// inside runNegativeCheckTimeCell for AC-3); a t.Skip would silently
// bypass the staleness assertion and let the divergence tracking
// degrade to one-way (skip never un-skips itself).
//
// This is the assertion-strength tooth for AC-4: AC-4's
// IllegalCellsAllCovered catches enumeration drift (every cell has
// a subtest somewhere), but doesn't catch "subtest is just a skip
// with no assertion." This test catches that regression mode by
// walking the AST of the two driver files and flagging any t.Skip*
// call.
//
// Codifies the M-0125 retrofit lesson: the original drafts used
// t.Skipf for impl-gap cells, which meant ac2KnownImplGaps and
// ac3KnownImplGaps entries could become stale (kernel learns to
// reject) with no test-side signal. The retrofit replaced t.Skipf
// with staleness helpers — this policy prevents the regression.
//
// Scope: only the two driver files. Other M-0125 test files (AC-1
// preconditions test, the AC-4 meta-tests themselves) don't have
// per-cell impl-gap dispatch and aren't subject to the rule.
//
// testutil.SkipIfShortOrUnsupported(t) calls are unaffected — that's
// a `testutil.` selector, not `t.`, so the AST check filters past
// it. References to "t.Skipf" inside comments are also unaffected
// (the AST sees them as comments, not function calls).
func TestM0125_AC4_NoTestingSkipInNegativeDrivers(t *testing.T) {
	t.Parallel()

	driverFiles := []string{
		"m0125_negative_driver_test.go",
		"m0125_negative_checktime_driver_test.go",
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
			t.Errorf("%s:%d: t.%s in M-0125 negative driver — impl-gap cells must use a staleness-asserting helper (runImplGapStalenessVerbTime for AC-2, or the isImplGap branch inside runNegativeCheckTimeCell for AC-3), not t.Skip. The retrofit pattern makes divergence tracking two-way; t.Skipf reverts to one-way (skip never un-skips itself when a tracked impl-gap closes).",
				file, pos.Line, method)
			return true
		})
	}
}
