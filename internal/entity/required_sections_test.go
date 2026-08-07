package entity

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestBodyTemplate_RendersExactlyTheRequiredSections pins that the scaffold
// `aiwf add` commits is rendered from the owned per-kind section set rather
// than a literal standing beside it. Both sides read RequiredSections, so a
// kind carried by one and not the other fails here — where two independent
// copies would pass for as long as they happened to agree.
//
// Order is part of the claim: the set is the canonical render order, and a
// scaffold that emits the right headings in the wrong sequence is a body no
// reader recognizes.
func TestBodyTemplate_RendersExactlyTheRequiredSections(t *testing.T) {
	t.Parallel()
	for _, k := range AllKinds() {
		t.Run(string(k), func(t *testing.T) {
			t.Parallel()
			want := RequiredSections(k)
			if len(want) == 0 {
				t.Fatalf("RequiredSections(%s) is empty; every kind carries a section set", k)
			}
			var got []string
			for _, s := range ParseBodySectionsOrdered(BodyTemplate(k)) {
				got = append(got, s.Heading)
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("BodyTemplate(%s) headings (-want +got):\n%s", k, diff)
			}
		})
	}
}

// TestRequiredSections_CoversEveryKind pins that the owned set is total over
// the closed kind set. A kind added to AllKinds without a section set would
// otherwise reach the scaffold as a bare body and the check rule as an
// unvalidated one, both silently.
func TestRequiredSections_CoversEveryKind(t *testing.T) {
	t.Parallel()
	for _, k := range AllKinds() {
		if got := RequiredSections(k); len(got) == 0 {
			t.Errorf("RequiredSections(%s) = empty; every kind in AllKinds carries a section set", k)
		}
	}
}

// TestBodyTemplate_KindWithNoSectionSet pins the scaffold's behavior for a
// Kind carrying no section set. The tree loader does not produce such a kind,
// so this arm is defensive — but it ships, and a caller handed a bare "\n"
// gets a parseable body rather than an empty file.
func TestBodyTemplate_KindWithNoSectionSet(t *testing.T) {
	t.Parallel()
	for _, k := range []Kind{Kind("widget"), Kind("")} {
		t.Run(string(k), func(t *testing.T) {
			t.Parallel()
			if got, want := string(BodyTemplate(k)), "\n"; got != want {
				t.Errorf("BodyTemplate(%q) = %q, want %q", k, got, want)
			}
		})
	}
}

// TestRequiredSections_ReturnsACopy pins that a caller cannot reach through
// the returned slice and mutate the owned set. The set is process-wide state
// read by the scaffold and by every check run, so an aliased return would let
// one caller silently rewrite what every other caller sees.
func TestRequiredSections_ReturnsACopy(t *testing.T) {
	t.Parallel()
	first := RequiredSections(KindEpic)
	if len(first) == 0 {
		t.Fatal("RequiredSections(epic) is empty")
	}
	first[0] = "Mutated"
	if second := RequiredSections(KindEpic); second[0] == "Mutated" {
		t.Errorf("RequiredSections returned an aliased slice; mutation leaked to the owned set")
	}
}
