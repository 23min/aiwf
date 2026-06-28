package check

import (
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// CodeAreaRequired is the finding code emitted by AreaRequired. Typed
// (like CodeAreaUnknown, per G-0129) so the compiler closes on rename /
// retire across the emit site and tests.
const CodeAreaRequired = "area-required"

// AreaRequired (error) reports any non-archived entity of a self-tagging
// root kind (epic, ADR, gap, decision, contract) whose `area` frontmatter
// is empty, when the consumer has opted into strictness via
// `aiwf.yaml: areas.required: true` (M-0178). It is the present-at-all
// chokepoint for the 1:1 monorepo where every entity belongs to exactly
// one project — orthogonal to AreaUnknown, which polices present-⇒-declared.
//
// The knob is the gate: with required false the rule emits nothing (no
// warning→error bump — it simply does not fire). The remediation is
// `aiwf set-area <id> <member>` (M-0183).
//
// Three guards, in order:
//   - Inert when required is false OR declared is empty (no `areas`
//     block). The empty-declared guard is defensive — config.Load rejects
//     required:true with zero members, so it is unreachable through the
//     normal path, but the rule stays self-contained.
//   - Milestone skip (load-bearing): a milestone's `area` is blanked at
//     load and derived from its parent epic (tree.go, E-0043 / M-0171/AC-3),
//     so an untagged milestone always reads area=="". Skipping KindMilestone
//     is what makes an untagged epic fire exactly once rather than once per
//     untagged milestone underneath it.
//   - Archive scoping (ADR-0004 §"check shape rules"): archived entities
//     are out of scope for active linting, matching the other
//     shape-and-health rules.
//
// Composed at the CLI layer (internal/cli/check) with the declared set and
// the required bool sourced from config — the same seam AreaUnknown uses —
// so the pure check.Run stays config-agnostic.
func AreaRequired(t *tree.Tree, declared []string, required bool) []Finding {
	if !required || len(declared) == 0 {
		return nil
	}
	var findings []Finding
	for _, e := range t.Entities {
		if !entity.CarriesOwnArea(e.Kind) {
			continue
		}
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		if e.Area != "" {
			continue
		}
		findings = append(findings, Finding{
			Code:     CodeAreaRequired,
			Severity: SeverityError,
			Message: fmt.Sprintf(
				"%s has no area but aiwf.yaml: areas.required is set (declared: %s)",
				e.ID, strings.Join(declared, ", ")),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "area",
		})
	}
	return findings
}
