package check

import (
	"testing"

	codespkg "github.com/23min/aiwf/internal/codes"
)

// TestIsolationEscape_AC13_TypedCodeDescriptor pins M-0106/AC-13:
// the isolation-escape finding-code descriptor lands in
// internal/check/ as a typed [codes.Code] value (per the G-0129
// pattern adopted for CodeProvenanceAuthorizationOutOfScope), with
// a stable ID and the correct Class.
//
// The structural assertions:
//   - The ID is exactly "isolation-escape" — the stable wire string
//     that messages, JSON envelopes, and downstream consumers key
//     on. A typo regression that drifts the ID fires this test.
//   - The Class is ClassBranchChoreography — the new layer-4 carve-
//     out introduced for this milestone. A regression that reuses
//     ClassStructural (the default zero value) fires this test.
//   - The value is a [codes.Code], not a bare string constant —
//     enforces the G-0129 typed-code shape. The compile-time check
//     would catch a bare-string drift, but pinning the type via a
//     non-trivial assertion (Class field access) gives explicit
//     evidence in the test set.
func TestIsolationEscape_AC13_TypedCodeDescriptor(t *testing.T) {
	t.Parallel()

	if got, want := CodeIsolationEscape.ID, "isolation-escape"; got != want {
		t.Errorf("CodeIsolationEscape.ID = %q; want %q", got, want)
	}
	if got, want := CodeIsolationEscape.Class, codespkg.ClassBranchChoreography; got != want {
		t.Errorf("CodeIsolationEscape.Class = %v; want %v (ClassBranchChoreography)", got, want)
	}
}

// TestIsolationEscape_AC13_ClassBranchChoreographyDistinct pins
// that ClassBranchChoreography is a NEW enum value distinct from
// the prior two classes (ClassStructural=0, ClassLegality=1). A
// regression that re-orders the enum so the new class collides
// with an existing one fires this test.
//
// The assertion is positional rather than literal: we don't pin
// the numeric value (Class is an iota — its concrete int can shift
// in principle), but we DO pin that the three values are pairwise
// distinct. That's the load-bearing invariant: callers
// distinguishing between classes can do so.
func TestIsolationEscape_AC13_ClassBranchChoreographyDistinct(t *testing.T) {
	t.Parallel()

	classes := []codespkg.Class{
		codespkg.ClassStructural,
		codespkg.ClassLegality,
		codespkg.ClassBranchChoreography,
	}
	seen := make(map[codespkg.Class]int)
	for i, c := range classes {
		if prior, ok := seen[c]; ok {
			t.Errorf("class at index %d collides with class at index %d (both = %v)", i, prior, c)
		}
		seen[c] = i
	}
}
