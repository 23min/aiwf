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

// Adopt returns the selection an enabled run records: an unset selection
// becomes every matched pack that is not ignored, in match order, and stays
// unset when none applies. An existing selection is returned as is, so callers
// can tell an unchanged selection by identity.
func Adopt(matches []Match, packs *[]string, ignored []string) *[]string {
	if packs != nil {
		return packs
	}
	var next []string
	for _, match := range matches {
		if !slices.Contains(ignored, match.PackID) {
			next = append(next, match.PackID)
		}
	}
	if next == nil {
		return nil
	}
	return &next
}
