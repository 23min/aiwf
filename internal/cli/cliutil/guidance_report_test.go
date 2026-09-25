package cliutil

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/initrepo"
)

func TestGuidanceReport(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		outcome *initrepo.GuidanceOutcome
		want    []string
	}{
		{name: "no section when guidance was not considered"},
		{
			name:    "explicit empty selection, installed packs and every list",
			outcome: &initrepo.GuidanceOutcome{Selection: &[]string{}, Installed: []string{"a/one", "b/two"}, Commit: "abc123", Ignored: []string{"c/three"}},
			want: []string{
				"", "Guidance:",
				"  selected:  none (explicit empty selection)",
				"  adopted:   none",
				"  installed: a/one, b/two at abc123",
				"  ignored:   c/three",
			},
		},
		{
			name:    "unreadable installed guidance and packs the catalogue lacks",
			outcome: &initrepo.GuidanceOutcome{Selection: &[]string{"a/one", "gone/one", "gone/two"}, Adopted: []string{"a/one"}, InspectError: "invalid ownership digest", Missing: []string{"gone/one", "gone/two"}},
			want: []string{
				"", "Guidance:",
				"  selected:  a/one, gone/one, gone/two",
				"  adopted:   a/one",
				"  installed: cannot inspect installed policy: invalid ownership digest",
				"  ignored:   none",
				"  missing:   gone/one; the catalogue no longer lists it: remove it from guidance.packs or move it to guidance.ignored",
				"  missing:   gone/two; the catalogue no longer lists it: remove it from guidance.packs or move it to guidance.ignored",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if diff := cmp.Diff(tc.want, GuidanceReport(tc.outcome)); diff != "" {
				t.Fatalf("report (-want +got):\n%s", diff)
			}
		})
	}
}
