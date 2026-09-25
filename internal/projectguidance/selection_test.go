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
			if diff := cmp.Diff(tc.want, Adopt(tc.matches, nil, tc.ignored)); diff != "" {
				t.Fatalf("Adopt (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAdopt_SelectionCoveringEveryMatchIsReturnedUnchanged(t *testing.T) {
	t.Parallel()
	packs := &[]string{"go/base"}
	if got := Adopt([]Match{{PackID: "go/base"}}, packs, nil); got != packs {
		t.Fatalf("Adopt replaced the existing selection: %v", *got)
	}
}
