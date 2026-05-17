package cliutil

import (
	"strings"

	"github.com/23min/aiwf/internal/entity"
)

// JoinKinds renders a slice of entity.Kind as a comma-separated string
// (e.g. for "unknown kind X (known: epic, milestone, …)" error messages).
// Stable ordering: iterates the input slice as-given; callers that need
// canonical order pass entity.AllKinds() (which is canonical).
func JoinKinds(ks []entity.Kind) string {
	parts := make([]string, len(ks))
	for i, k := range ks {
		parts[i] = string(k)
	}
	return strings.Join(parts, ", ")
}
