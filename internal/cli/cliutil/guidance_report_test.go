package cliutil

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/projectguidance"
)

func TestGuidanceReport(t *testing.T) {
	t.Parallel()
	selected := &[]string{"a/one", "gone/one", "gone/two"}
	missing := projectguidance.NotInstalledStates(selected, true, projectguidance.Inspection{IndexIntact: true, Packs: []string{"a/one"}})
	for _, tc := range []struct {
		name    string
		outcome *initrepo.GuidanceOutcome
		want    []string
	}{
		{name: "no section when guidance was not considered"},
		{
			name: "explicit empty selection, installed packs, every list and a multi-line blocker",
			outcome: &initrepo.GuidanceOutcome{
				Selection: &[]string{}, Ignored: []string{"c/three"}, Blocker: "git clone: exit status 128\nfatal: not allowed\n",
				Installed: projectguidance.Inspection{IndexIntact: true, Revision: "abc123", Packs: []string{"a/one", "b/two"}},
			},
			want: []string{
				"", "Guidance:",
				"  selected:  " + projectguidance.SelectionState(&[]string{}),
				"  adopted:   none",
				"  installed: a/one, b/two at abc123",
				"  ignored:   c/three",
				"  blocked:   git clone: exit status 128 fatal: not allowed",
			},
		},
		{
			name: "pending installation and selected packs not installed",
			outcome: &initrepo.GuidanceOutcome{
				Selection: selected, Adopted: []string{"a/one"},
				Installed: projectguidance.Inspection{IndexIntact: true, Revision: "abc123", Packs: []string{"a/one"}, Pending: true},
			},
			want: []string{
				"", "Guidance:",
				"  selected:  a/one, gone/one, gone/two",
				"  adopted:   a/one",
				"  installed: a/one at abc123",
				"  ignored:   none",
				"  pending:   " + projectguidance.PendingState,
				"  missing:   " + missing[0],
				"  missing:   " + missing[1],
			},
		},
		{
			name:    "disabled maintenance reports no missing pack",
			outcome: &initrepo.GuidanceOutcome{Selection: &[]string{"a/one"}, Disabled: true, Installed: projectguidance.Inspection{IndexIntact: true}},
			want: []string{
				"", "Guidance:",
				"  selected:  a/one",
				"  " + projectguidance.DisabledState,
				"  adopted:   none",
				"  installed: none",
				"  ignored:   none",
			},
		},
		{
			name:    "unreadable installed guidance claims nothing about missing packs",
			outcome: &initrepo.GuidanceOutcome{Selection: &[]string{"a/one"}, InspectError: "cannot inspect installed policy: invalid ownership digest", Installed: projectguidance.Inspection{IndexIntact: true}},
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

// TestGuidanceReportHelp_NamesEveryLabel derives the labels from reports that
// between them carry every optional line, so a label added to the report
// without help fails here.
func TestGuidanceReportHelp_NamesEveryLabel(t *testing.T) {
	t.Parallel()
	report := GuidanceReport(&initrepo.GuidanceOutcome{Disabled: true})
	report = append(report, GuidanceReport(&initrepo.GuidanceOutcome{
		Selection: &[]string{"a/one"}, Blocker: "cause",
		Installed: projectguidance.Inspection{IndexIntact: true, Pending: true},
	})...)
	labels := map[string]bool{}
	for _, line := range report {
		if !strings.HasPrefix(line, "  ") {
			continue
		}
		clause := ""
		if label, _, found := strings.Cut(strings.TrimSpace(line), ":"); found && !strings.Contains(label, " ") {
			labels[label] = true
			clause = label + " ("
		} else if line == "  "+projectguidance.DisabledState {
			labels["disabled"] = true
			clause = "maintenance is disabled"
		}
		if clause == "" || !strings.Contains(GuidanceReportHelp, clause) {
			t.Errorf("help does not describe report line %q", line)
		}
	}
	for _, label := range []string{"selected", "adopted", "installed", "ignored", "disabled", "pending", "blocked", "missing"} {
		if !labels[label] {
			t.Errorf("fixture reports produce no %q line", label)
		}
	}
}
