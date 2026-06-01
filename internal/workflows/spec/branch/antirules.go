package branch

import (
	"sort"

	"github.com/23min/aiwf/internal/workflows/spec"
)

// AntiRules returns the layer-4 anti-rules — patterns the kernel
// deliberately does NOT police at the branch-choreography layer.
// Sorted by AntiRule.ID for deterministic output. Cycle 1 (this
// commit) returns an empty slice; subsequent cycles register the
// AntiRule cells as they're identified.
//
// Top-level integration: aggregated into spec.AntiRules() via
// `out = append(out, branch.AntiRules()...)`.
func AntiRules() []spec.AntiRule {
	out := []spec.AntiRule{}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}
