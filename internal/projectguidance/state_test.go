package projectguidance

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNotInstalledStates(t *testing.T) {
	t.Parallel()
	intact := func(packs ...string) Inspection { return Inspection{IndexIntact: true, Packs: packs} }
	for _, tc := range []struct {
		name       string
		packs      *[]string
		enabled    bool
		inspection Inspection
		want       []string
	}{
		{name: "unset selection reports nothing", enabled: true, inspection: intact()},
		{name: "disabled maintenance reports nothing", packs: &[]string{"a/one"}, enabled: false, inspection: intact()},
		{name: "an index that is not intact leaves the installed set unknown", packs: &[]string{"a/one"}, enabled: true, inspection: Inspection{Files: []string{indexFile}}},
		{name: "reports each selected pack the index lacks in selection order", packs: &[]string{"b/two", "a/one", "c/three"}, enabled: true, inspection: intact("a/one"), want: []string{"b/two", "c/three"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got []string
			for _, state := range NotInstalledStates(tc.packs, tc.enabled, tc.inspection) {
				id, _, _ := strings.Cut(state, " ")
				got = append(got, id)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("not-installed packs (-want +got):\n%s", diff)
			}
		})
	}
}

func TestInstalledState(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		inspection Inspection
		want       string
	}{
		{name: "nothing installed", want: "none"},
		{name: "intact index routing no pack", inspection: Inspection{IndexIntact: true, Revision: "abc", Files: []string{indexFile}}, want: "none"},
		{name: "intact index", inspection: Inspection{IndexIntact: true, Revision: "abc", Packs: []string{"a/one", "b/two"}, Files: []string{indexFile}}, want: "a/one, b/two at abc"},
		{name: "recorded index that is not intact", inspection: Inspection{Files: []string{indexFile}}, want: UnknownRevision},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := InstalledState(tc.inspection); got != tc.want {
				t.Fatalf("InstalledState = %q, want %q", got, tc.want)
			}
		})
	}
}
