package projectguidance

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAdopt_UnsetSelection(t *testing.T) {
	t.Parallel()
	matches := []Match{{PackID: "go/base"}, {PackID: "python/astral"}, {PackID: "go/cobra"}}
	for _, tc := range []struct {
		name    string
		matches []Match
		ignored []string
		want    *[]string
	}{
		{name: "adopts every match in match order", matches: matches, want: &[]string{"go/base", "python/astral", "go/cobra"}},
		{name: "skips ignored packs", matches: matches, ignored: []string{"python/astral"}, want: &[]string{"go/base", "go/cobra"}},
		{name: "stays unset when every match is ignored", matches: matches[:1], ignored: []string{"go/base"}},
		{name: "stays unset when nothing matches"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, added := Adopt(tc.matches, nil, tc.ignored)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("Adopt (-want +got):\n%s", diff)
			}
			var wantAdded []string
			if tc.want != nil {
				wantAdded = *tc.want
			}
			if diff := cmp.Diff(wantAdded, added); diff != "" {
				t.Fatalf("added (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAdopt_SelectionCoveringEveryMatchIsReturnedUnchanged(t *testing.T) {
	t.Parallel()
	packs := &[]string{"go/base"}
	got, added := Adopt([]Match{{PackID: "go/base"}}, packs, nil)
	if got != packs || added != nil {
		t.Fatalf("Adopt changed a covering selection: %v, added %v", *got, added)
	}
}

func TestAdopt_ExistingSelectionGainsNewMatchesAndKeepsItsOwn(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		packs   []string
		matches []Match
		ignored []string
		want    []string
		added   []string
	}{
		{name: "appends new matches after existing entries", packs: []string{"python/astral", "go/base"}, matches: []Match{{PackID: "go/base"}, {PackID: "go/cobra"}}, want: []string{"python/astral", "go/base", "go/cobra"}, added: []string{"go/cobra"}},
		{name: "keeps selected packs that no longer match", packs: []string{"python/astral"}, matches: []Match{{PackID: "go/base"}}, want: []string{"python/astral", "go/base"}, added: []string{"go/base"}},
		{name: "an explicitly empty selection gains matches", packs: []string{}, matches: []Match{{PackID: "go/base"}}, want: []string{"go/base"}, added: []string{"go/base"}},
		{name: "skips ignored matches", packs: []string{"python/astral"}, matches: []Match{{PackID: "go/base"}, {PackID: "go/cobra"}}, ignored: []string{"go/cobra"}, want: []string{"python/astral", "go/base"}, added: []string{"go/base"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			packs := tc.packs
			got, added := Adopt(tc.matches, &packs, tc.ignored)
			if got == nil {
				t.Fatal("Adopt returned no selection")
			}
			if diff := cmp.Diff(tc.want, *got); diff != "" {
				t.Fatalf("Adopt (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.added, added); diff != "" {
				t.Fatalf("added (-want +got):\n%s", diff)
			}
			if got == &packs {
				t.Fatal("a changed selection must be a new value so callers see the change")
			}
		})
	}
}
