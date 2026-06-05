//go:build testpins

package policies

import (
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
