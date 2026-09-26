package projectguidance

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNotInstalledStates(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		packs     *[]string
		enabled   bool
		installed []string
		want      []string
	}{
		{name: "unset selection reports nothing", enabled: true},
		{name: "disabled maintenance reports nothing", packs: &[]string{"a/one"}, enabled: false},
		{name: "reports each selected pack the index lacks in selection order", packs: &[]string{"b/two", "a/one", "c/three"}, enabled: true, installed: []string{"a/one"}, want: []string{"b/two", "c/three"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got []string
			for _, state := range NotInstalledStates(tc.packs, tc.enabled, tc.installed) {
				id, _, _ := strings.Cut(state, " ")
				got = append(got, id)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("not-installed packs (-want +got):\n%s", diff)
			}
		})
	}
}
