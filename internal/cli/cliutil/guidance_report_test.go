package cliutil

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/initrepo"
)

func TestGuidanceReport(t *testing.T) {
	t.Parallel()
	notInstalled := "is selected but not installed; if the catalogue no longer lists it, remove it from guidance.packs or move it to guidance.ignored, otherwise run aiwf update"
	for _, tc := range []struct {
		name    string
		outcome *initrepo.GuidanceOutcome
		want    []string
	}{
		{name: "no section when guidance was not considered"},
		{
			name:    "explicit empty selection, installed packs, every list and a blocker",
			outcome: &initrepo.GuidanceOutcome{Selection: &[]string{}, Installed: []string{"a/one", "b/two"}, Commit: "abc123", Ignored: []string{"c/three"}, Blocker: "cause; fix it, then retry"},
			want: []string{
				"", "Guidance:",
				"  selected:  none (explicit empty selection)",
				"  adopted:   none",
				"  installed: a/one, b/two at abc123",
				"  ignored:   c/three",
				"  blocked:   cause; fix it, then retry",
			},
		},
		{
			name:    "pending installation and selected packs not installed",
			outcome: &initrepo.GuidanceOutcome{Selection: &[]string{"a/one", "gone/one", "gone/two"}, Adopted: []string{"a/one"}, Installed: []string{"a/one"}, Commit: "abc123", Pending: true},
			want: []string{
				"", "Guidance:",
				"  selected:  a/one, gone/one, gone/two",
				"  adopted:   a/one",
				"  installed: a/one at abc123",
				"  ignored:   none",
				"  pending:   installation incomplete; rerun aiwf update before using the pack set",
				"  missing:   gone/one " + notInstalled,
				"  missing:   gone/two " + notInstalled,
			},
		},
		{
			name:    "unreadable installed guidance claims nothing about missing packs",
			outcome: &initrepo.GuidanceOutcome{Selection: &[]string{"a/one"}, InspectError: "invalid ownership digest"},
			want: []string{
				"", "Guidance:",
				"  selected:  a/one",
				"  adopted:   none",
				"  installed: cannot inspect installed policy: invalid ownership digest",
				"  ignored:   none",
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

// TestGuidanceReportHelp_NamesEveryLabel derives the labels from a report
// carrying every optional line, so a label added to the report without help
// fails here.
func TestGuidanceReportHelp_NamesEveryLabel(t *testing.T) {
	t.Parallel()
	// Disabled maintenance suppresses the missing line, so two outcomes
	// between them carry every optional line.
	report := GuidanceReport(&initrepo.GuidanceOutcome{Disabled: true})
	report = append(report, GuidanceReport(&initrepo.GuidanceOutcome{Selection: &[]string{"a/one"}, Pending: true, Blocker: "cause"})...)
	labels := map[string]bool{}
	for _, line := range report {
		if !strings.HasPrefix(line, "  ") {
			continue
		}
		label, _, found := strings.Cut(strings.TrimSpace(line), ":")
		if strings.HasPrefix(line, "  maintenance disabled") {
			label, found = "maintenance is disabled", true
		}
		labels[label] = true
		if !found || !strings.Contains(GuidanceReportHelp, label) {
			t.Errorf("help does not describe report line %q", line)
		}
	}
	for _, label := range []string{"maintenance is disabled", "pending", "blocked", "missing"} {
		if !labels[label] {
			t.Errorf("fixture outcomes produce no %q line", label)
		}
	}
}
