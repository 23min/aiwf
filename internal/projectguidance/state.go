package projectguidance

import (
	"slices"
	"strings"
)

// The update report and aiwf doctor word guidance state from this file, so
// both describe the same repository in the same words.

// DisabledState words guidance.enabled: false.
const DisabledState = "maintenance disabled; installed policy remains in use"

// PendingState words an installation an interrupted run left incomplete.
const PendingState = "installation incomplete; rerun aiwf update before using the pack set"

// InspectErrorState words installed guidance that cannot be read.
func InspectErrorState(err error) string {
	return "cannot inspect installed policy: " + err.Error()
}

// UnknownRevision words an installation whose owned index is not intact.
const UnknownRevision = "unknown (no intact owned index)"

// InstalledState words the installed set: its packs and source commit, none,
// or unknown when an owned index is recorded but not intact.
func InstalledState(inspection Inspection) string {
	switch {
	case inspection.IndexIntact && len(inspection.Packs) > 0:
		return strings.Join(inspection.Packs, ", ") + " at " + inspection.Revision
	case !inspection.IndexIntact && slices.Contains(inspection.Files, indexFile):
		return UnknownRevision
	default:
		return "none"
	}
}

// SelectionState words guidance.packs, telling an unset selection from an
// explicitly empty one.
func SelectionState(packs *[]string) string {
	if packs == nil {
		return "not adopted (guidance.packs unset)"
	}
	if len(*packs) == 0 {
		return "none (explicit empty selection)"
	}
	return strings.Join(*packs, ", ")
}

// NotInstalledStates words each selected pack an intact installed index does
// not route. Without an intact index the installed set is unknown, and
// disabled maintenance installs nothing, so neither reports any.
func NotInstalledStates(packs *[]string, enabled bool, inspection Inspection) []string {
	if packs == nil || !enabled || !inspection.IndexIntact {
		return nil
	}
	var states []string
	for _, id := range *packs {
		if !slices.Contains(inspection.Packs, id) {
			states = append(states, id+" is selected but not installed; if the catalogue no longer lists it, remove it from guidance.packs or move it to guidance.ignored, otherwise run aiwf update")
		}
	}
	return states
}
