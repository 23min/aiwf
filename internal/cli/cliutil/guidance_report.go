package cliutil

import (
	"strings"

	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/projectguidance"
)

// GuidanceReportHelp describes the guidance section for the help of the verbs
// that print it.
const GuidanceReportHelp = "The run ends with a Guidance section: selected (guidance.packs, or that it is unset or explicitly empty), adopted (packs this run added), installed (the installed packs and their source commit) and ignored, each shown as none when empty. When they apply, it adds that maintenance is disabled; pending (an interrupted installation); blocked (why installation did not complete and what clears it; other steps and the exit status are unaffected); and missing (a selected pack that is not installed, with its fix). aiwf doctor words these states the same way."

// PrintGuidanceReport prints the guidance section that ends init and update.
func PrintGuidanceReport(outcome *initrepo.GuidanceOutcome) {
	for _, line := range GuidanceReport(outcome) {
		Println(line)
	}
}

// GuidanceReport renders the guidance section. Every list is present, as
// "none" when empty; a run that did not consider project guidance has no
// section.
func GuidanceReport(outcome *initrepo.GuidanceOutcome) []string {
	if outcome == nil {
		return nil
	}
	lines := []string{"", "Guidance:", "  selected:  " + projectguidance.SelectionState(outcome.Selection)}
	if outcome.Disabled {
		lines = append(lines, "  "+projectguidance.DisabledState)
	}
	lines = append(lines, "  adopted:   "+packList(outcome.Adopted))
	switch {
	case outcome.InspectError != "":
		lines = append(lines, "  installed: cannot inspect installed policy: "+outcome.InspectError)
	case len(outcome.Installed) == 0:
		lines = append(lines, "  installed: none")
	default:
		lines = append(lines, "  installed: "+strings.Join(outcome.Installed, ", ")+" at "+outcome.Commit)
	}
	lines = append(lines, "  ignored:   "+packList(outcome.Ignored))
	if outcome.Pending {
		lines = append(lines, "  pending:   "+projectguidance.PendingState)
	}
	if outcome.Blocker != "" {
		lines = append(lines, "  blocked:   "+outcome.Blocker)
	}
	if outcome.InspectError == "" {
		for _, state := range projectguidance.NotInstalledStates(outcome.Selection, !outcome.Disabled, outcome.Installed) {
			lines = append(lines, "  missing:   "+state)
		}
	}
	return lines
}

func packList(ids []string) string {
	if len(ids) == 0 {
		return "none"
	}
	return strings.Join(ids, ", ")
}
