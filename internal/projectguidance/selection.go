package projectguidance

import (
	"context"
	"slices"

	"github.com/23min/aiwf/internal/config"
)

// Selector inspects available policy, reporting suggestions or recording choices
// after an explicit, completed interaction. Reporting and failures must leave the
// caller's configuration unchanged.
type Selector func(context.Context, string, Catalogue, *config.Config) error

// Adopt returns the selection an enabled run records: the current selection
// with every matched pack that is neither selected nor ignored appended in
// match order. Selected packs are never removed, and an unset selection with no
// applicable pack stays unset. It returns packs itself when nothing is added,
// so callers can tell an unchanged selection by identity.
func Adopt(matches []Match, packs *[]string, ignored []string) *[]string {
	var current []string
	if packs != nil {
		current = *packs
	}
	next := slices.Clone(current)
	for _, match := range matches {
		if !slices.Contains(next, match.PackID) && !slices.Contains(ignored, match.PackID) {
			next = append(next, match.PackID)
		}
	}
	if len(next) == len(current) {
		return packs
	}
	return &next
}
