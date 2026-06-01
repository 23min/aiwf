package branch

import (
	"sort"

	"github.com/23min/aiwf/internal/workflows/spec"
)

// Rules returns the layer-4 branch-choreography cells, sorted by cell
// id for deterministic output. The closed set is the 12 corner cases
// from E-0030 §"Corner cases" plus the 4 override-surface rows from
// §"Sovereign override surface" — Cycle 1 (this commit) lands the
// scaffold returning an empty slice; Cycles 2 + 3 populate the
// catalog.
//
// Top-level integration: aggregated into spec.Rules() via
// `out = append(out, branch.Rules()...)` so per-cell consumers
// (m0124/m0125 coverage drivers, schema-invariants drift test,
// key-uniqueness) iterate the union seamlessly.
func Rules() []spec.Rule {
	out := []spec.Rule{}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}
