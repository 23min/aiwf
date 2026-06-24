package check

import (
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// CodeAreaUnknown is the finding code emitted by AreaUnknown. Typed per
// G-0129 so the compiler closes on rename / retire across the emit site
// and tests.
const CodeAreaUnknown = "area-unknown"

// AreaUnknown (warning) reports any non-archived entity whose `area`
// frontmatter value is present and non-empty but not a member of the
// declared set (`aiwf.yaml: areas.members`). It is the present-⇒-declared
// chokepoint for E-0043 — typo protection for the optional grouping tag,
// the authoritative surface a creation-time flag alone can't cover (a
// hand-edit or an `aiwf import` can introduce an undeclared area without
// passing through `aiwf add --area`).
//
// Three behaviors fall out of the guards, in order:
//   - Inert when declared is empty (no `areas` block): present `area`
//     values parse but nothing validates, per M-0171's "the field is
//     inert until a block is declared" contract.
//   - Absence (empty `area`) is never evaluated: absent / explicit-null /
//     empty all deserialize to "" and only a present, non-empty value can
//     be "unknown".
//   - Archive scoping (ADR-0004 §"check shape rules"): archived entities
//     are out of scope for active linting, matching the other
//     shape-and-health rules.
//
// The rule reads the *stored* area, so only root kinds that carry their
// own `area` can fire; a milestone (area blanked at load, derived from
// its parent epic) never double-reports under a bad-area epic.
//
// Composed at the CLI layer (internal/cli/check) with the declared set
// sourced from config — the same seam TreeDiscipline, the contract
// checks, and the tests-metrics check use — so the pure check.Run stays
// config-agnostic (the boundary M-0171/AC-4's metamorphic guard pins).
// Severity is warning with no strictness knob (E-0043 / M-0172 decision).
func AreaUnknown(t *tree.Tree, declared []string) []Finding {
	if len(declared) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(declared))
	for _, m := range declared {
		set[m] = struct{}{}
	}
	var findings []Finding
	for _, e := range t.Entities {
		if e.Area == "" {
			continue
		}
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		if _, ok := set[e.Area]; ok {
			continue
		}
		findings = append(findings, Finding{
			Code:     CodeAreaUnknown,
			Severity: SeverityWarning,
			Message: fmt.Sprintf(
				"%s declares area %q which is not in the declared set (aiwf.yaml: areas.members: %s)",
				e.ID, e.Area, strings.Join(declared, ", ")),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "area",
		})
	}
	return findings
}
