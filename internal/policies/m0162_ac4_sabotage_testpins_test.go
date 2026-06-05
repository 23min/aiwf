//go:build testpins

package policies

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec/branch/branchtest"
)

// TestM0162_AC4_Bijection_Sabotage exercises the 4 invariants
// against synthetic fixture data, asserting each violation kind
// is reported correctly. Also pins the 3 meta-cells so the
// catalog's own enforcement chokepoints satisfy invariant 1
// (every cell has at least one Pin).
//
// File is build-tagged testpins because pinning via branchtest.Pin
// only matters under that tag — without testpins, the registry is
// a no-op and the meta-cells would be allowlisted anyway. The
// static AC-4 check reads branchtest.Pin call-site LITERALS
// regardless of build-tag (AST-level extraction), so the meta-
// cell pins satisfy invariant 1 even when the registry itself
// is inactive.
//
// Sabotage discrimination per AC-4 body §"Mechanical assertions"
// item 4. Each subtest constructs a synthetic snapshot of cells +
// pins, runs evaluateBijection, and asserts the right violation
// kind is reported.
func TestM0162_AC4_Bijection_Sabotage(t *testing.T) {
	t.Parallel()
	branchtest.Pin("branch-cell-meta-bijection-enforced", t.Name())

	t.Run("invariant_1_cell_without_pin_fires", func(t *testing.T) {
		t.Parallel()
		branchtest.Pin("branch-cell-meta-cell-orphan-detected", t.Name())

		cells := []string{"X-cell", "Y-cell-unpinned"}
		pins := map[string][]string{"X-cell": {"line:1"}}
		v := evaluateBijection(cells, pins, nil)
		if !containsViolationKind(v, kindCellWithoutPin, "Y-cell-unpinned") {
			t.Errorf("invariant-1 sabotage: expected cell-without-pin violation naming %q\n  got: %+v", "Y-cell-unpinned", v)
		}
	})

	t.Run("invariant_2_orphan_pin_fires", func(t *testing.T) {
		t.Parallel()
		branchtest.Pin("branch-cell-meta-pin-orphan-detected", t.Name())

		cells := []string{"X-cell"}
		pins := map[string][]string{
			"X-cell":          {"line:1"},
			"Z-cell-nonexist": {"line:2"},
		}
		v := evaluateBijection(cells, pins, nil)
		if !containsViolationKind(v, kindPinOrphan, "Z-cell-nonexist") {
			t.Errorf("invariant-2 sabotage: expected orphan-pin violation naming %q\n  got: %+v", "Z-cell-nonexist", v)
		}
	})

	t.Run("invariant_3_double_pin_fires", func(t *testing.T) {
		t.Parallel()

		cells := []string{"X-cell"}
		pins := map[string][]string{
			"X-cell": {"line:1", "line:2"},
		}
		v := evaluateBijection(cells, pins, nil)
		if !containsViolationKind(v, kindDoublePin, "X-cell") {
			t.Errorf("invariant-3 sabotage: expected double-pin violation naming %q\n  got: %+v", "X-cell", v)
		}
	})

	// Seam-level sabotage tests for collectPinReferences (reviewer
	// S5 fix). Writes synthetic *_test.go files to a temp dir's
	// internal/ subtree, runs collectPinReferences, and asserts
	// the pin extraction matches expectations. Pins the production
	// AST-scan path, not just the evaluateBijection helper.

	t.Run("seam_literal_pinCell_extracted", func(t *testing.T) {
		t.Parallel()
		root := writeAC4Fixture(t, "literal_pinCell_test.go", `package fakefixture

import "testing"

func TestX(t *testing.T) {
	pinCell("branch-cell-fixture-literal", t.Name())
}
`)
		refs := collectPinReferences(t, root)
		if len(refs.Literals["branch-cell-fixture-literal"]) != 1 {
			t.Errorf("seam: literal pinCell extraction failed\n  refs.Literals = %+v", refs.Literals)
		}
	})

	t.Run("seam_qualified_branchtest_Pin_extracted", func(t *testing.T) {
		t.Parallel()
		root := writeAC4Fixture(t, "qualified_pin_test.go", `package fakefixture

import (
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec/branch/branchtest"
)

func TestY(t *testing.T) {
	branchtest.Pin("branch-cell-fixture-qualified", t.Name())
}
`)
		refs := collectPinReferences(t, root)
		if len(refs.Literals["branch-cell-fixture-qualified"]) != 1 {
			t.Errorf("seam: qualified branchtest.Pin extraction failed\n  refs.Literals = %+v", refs.Literals)
		}
	})

	t.Run("seam_dynamic_prefix_extracted", func(t *testing.T) {
		t.Parallel()
		root := writeAC4Fixture(t, "dynamic_prefix_test.go", `package fakefixture

import "testing"

func TestZ(t *testing.T) {
	for _, name := range []string{"a", "b"} {
		pinCell("branch-cell-fixture-prefix-"+name, t.Name())
	}
}
`)
		refs := collectPinReferences(t, root)
		if len(refs.Prefixes) != 1 || refs.Prefixes[0].Prefix != "branch-cell-fixture-prefix-" {
			t.Errorf("seam: dynamic-prefix extraction failed\n  refs.Prefixes = %+v", refs.Prefixes)
		}
	})

	t.Run("seam_CellID_struct_field_extracted", func(t *testing.T) {
		t.Parallel()
		root := writeAC4Fixture(t, "cellid_struct_test.go", `package fakefixture

type Scenario struct {
	CellID string
	Name   string
}

var _ = []Scenario{
	{CellID: "branch-cell-fixture-struct", Name: "a"},
}
`)
		refs := collectPinReferences(t, root)
		if len(refs.Literals["branch-cell-fixture-struct"]) != 1 {
			t.Errorf("seam: CellID struct field extraction failed\n  refs.Literals = %+v", refs.Literals)
		}
	})

	t.Run("seam_no_pin_returns_empty", func(t *testing.T) {
		t.Parallel()
		root := writeAC4Fixture(t, "empty_test.go", `package fakefixture

import "testing"

func TestNothing(t *testing.T) {
}
`)
		refs := collectPinReferences(t, root)
		if len(refs.Literals) != 0 || len(refs.Prefixes) != 0 {
			t.Errorf("seam: empty fixture should produce no pins\n  refs = %+v", refs)
		}
	})

	t.Run("invariant_1_allowlist_exempts_cell", func(t *testing.T) {
		t.Parallel()

		cells := []string{"X-cell", "Y-allowlisted"}
		pins := map[string][]string{"X-cell": {"line:1"}}
		allow := map[string]string{"Y-allowlisted": "exempted in test"}
		v := evaluateBijection(cells, pins, allow)
		if containsViolationKind(v, kindCellWithoutPin, "Y-allowlisted") {
			t.Errorf("invariant-1 allowlist: %q should be exempted but was reported as cell-without-pin\n  got: %+v", "Y-allowlisted", v)
		}
	})
}

// containsViolationKind reports whether v contains a violation of
// the given kind with the given Subject.
func containsViolationKind(v []bijectionViolation, k violationKind, subject string) bool {
	for _, vv := range v {
		if vv.Kind == k && vv.Subject == subject {
			return true
		}
	}
	return false
}

// writeAC4Fixture creates a temp dir layout mirroring the AC-4
// scanner's expected shape (an `internal/` subdir holding the
// fixture *_test.go file) and returns the temp root. Used by the
// seam-level sabotage tests above.
func writeAC4Fixture(t *testing.T, filename, body string) string {
	t.Helper()
	root := t.TempDir()
	internalDir := filepath.Join(root, "internal")
	if err := os.MkdirAll(internalDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", internalDir, err)
	}
	path := filepath.Join(internalDir, filename)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return root
}
