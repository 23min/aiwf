package cliutil

import (
	"strings"

	"github.com/23min/aiwf/internal/initrepo"
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
	selected := "not adopted (guidance.packs unset)"
	if outcome.Selection != nil {
		selected = strings.Join(*outcome.Selection, ", ")
		if selected == "" {
			selected = "none (explicit empty selection)"
		}
	}
	lines := []string{"", "Guidance:", "  selected:  " + selected}
	if outcome.Disabled {
		lines = append(lines, "  maintenance disabled; installed policy remains in use")
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
	for _, id := range outcome.Missing {
		lines = append(lines, "  missing:   "+id+"; the catalogue no longer lists it: remove it from guidance.packs or move it to guidance.ignored")
	}
	return lines
}

func packList(ids []string) string {
	if len(ids) == 0 {
		return "none"
	}
	return strings.Join(ids, ", ")
}
