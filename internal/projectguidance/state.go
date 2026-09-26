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

// NotInstalledStates words each selected pack the installed index does not
// route. Disabled maintenance installs nothing, so it reports none.
func NotInstalledStates(packs *[]string, enabled bool, installed []string) []string {
	if packs == nil || !enabled {
		return nil
	}
	var states []string
	for _, id := range *packs {
		if !slices.Contains(installed, id) {
			states = append(states, id+" is selected but not installed; if the catalogue no longer lists it, remove it from guidance.packs or move it to guidance.ignored, otherwise run aiwf update")
		}
	}
	return states
}
