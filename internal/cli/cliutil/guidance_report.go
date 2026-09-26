package cliutil

import (
	"strings"

	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/projectguidance"
)

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
