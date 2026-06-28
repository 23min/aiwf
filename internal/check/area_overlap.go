package check

import (
	"fmt"
	"os"
	"sort"

	"github.com/23min/aiwf/internal/areamatch"
	"github.com/23min/aiwf/internal/tree"
)

// CodeAreaOverlap is the finding code emitted by AreaOverlap. Typed per G-0129
// so the compiler closes on rename / retire across the emit site and tests.
const CodeAreaOverlap = "area-overlap"

// AreaOverlap (warning) reports any directory claimed by more than one
// declared area — ambiguous attribution: two areas whose `paths:` globs both
// match a common path. The ambiguity would make the entity-touching checks
// (mistag M-0181, auto-derive M-0182) behave non-deterministically there, so
// the path oracle must be a partition (each directory claimed by at most one
// area). One finding per overlapping area-pair, naming both areas and a
// representative shared path.
//
// Reads the filesystem read-only through the area-glob matcher (areamatch,
// M-0180/AC-1) and never fails on IO: an empty or unreadable root, or a glob
// that errors during the walk, is silently skipped (the roadmapCaseCollision
// precedent). Malformed globs are owned by config-load validation (Tier 1,
// areamatch.Validate), not re-reported here.
//
// Composed at the CLI layer (internal/cli/check) with the declared areas
// sourced from config — the same seam AreaUnknown and AreaDeadGlob use — so
// the pure check.Run stays config-agnostic. Inert with fewer than two areas
// carrying paths. Severity is warning, escalated to error under
// areas.required by ApplyAreaRequiredStrict.
func AreaOverlap(t *tree.Tree, areas []AreaPaths) []Finding {
	if t.Root == "" {
		return nil
	}
	// Never fail on IO: a missing or unreadable root yields no findings
	// (the roadmapCaseCollision precedent), the same guard AreaDeadGlob uses.
	if _, err := os.Stat(t.Root); err != nil {
		return nil
	}
	fsys := os.DirFS(t.Root)
	// Each area's matched path set (files and directories alike). MatchFS,
	// not MatchesAny: overlap needs the full sets to intersect them.
	matchSets := make([]map[string]bool, len(areas))
	for i, a := range areas {
		set := make(map[string]bool)
		for _, glob := range a.Paths {
			matches, err := areamatch.MatchFS(fsys, glob)
			if err != nil {
				continue
			}
			for _, m := range matches {
				set[m] = true
			}
		}
		matchSets[i] = set
	}
	var findings []Finding
	for i := range areas {
		for j := i + 1; j < len(areas); j++ {
			shared := firstSharedPath(matchSets[i], matchSets[j])
			if shared == "" {
				continue
			}
			findings = append(findings, Finding{
				Code:     CodeAreaOverlap,
				Severity: SeverityWarning,
				Message: fmt.Sprintf(
					"areas %q and %q both claim %q — a path must belong to at most one area",
					areas[i].Name, areas[j].Name, shared),
				Field: "areas.members.paths",
			})
		}
	}
	return findings
}

// firstSharedPath returns the lexically-smallest path present in both sets, or
// "" if the sets are disjoint. Sorting makes the chosen representative
// deterministic regardless of map iteration order.
func firstSharedPath(a, b map[string]bool) string {
	var shared []string
	for p := range a {
		if b[p] {
			shared = append(shared, p)
		}
	}
	if len(shared) == 0 {
		return ""
	}
	sort.Strings(shared)
	return shared[0]
}
