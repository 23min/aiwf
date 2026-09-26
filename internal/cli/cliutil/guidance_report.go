package cliutil

import (
	"strings"

	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/projectguidance"
)

// GuidanceReportHelp describes the guidance section for the help of the verbs
// that print it.
const GuidanceReportHelp = "The run ends with a Guidance section: selected (guidance.packs, or that it is unset or explicitly empty), adopted (packs this run added and recorded), installed (the installed packs and their source commit, or unknown when the installed index is not intact) and ignored (packs never adopted), each shown as none when empty. When they apply, it adds that maintenance is disabled, pending (an interrupted installation), blocked (why guidance was not fully installed and recorded and, where the installer names one, the action that clears it; other steps and the exit status are unaffected) and missing (one line with its fix for each selected pack the installed index does not list, while maintenance is enabled). The section also ends a run that stops on a later error. aiwf doctor words the selection, disabled maintenance, pending installation and missing packs the same way."

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
	installed := outcome.InspectError
	if installed == "" {
		installed = projectguidance.InstalledState(outcome.Installed)
	}
	lines = append(lines, "  installed: "+installed, "  ignored:   "+packList(outcome.Ignored))
	if outcome.Installed.Pending {
		lines = append(lines, "  pending:   "+projectguidance.PendingState)
	}
	if outcome.Blocker != "" {
		// Wrapped tool output can span lines; the section keeps one line per label.
		lines = append(lines, "  blocked:   "+strings.Join(strings.Fields(outcome.Blocker), " "))
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
