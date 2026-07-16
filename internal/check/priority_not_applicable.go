package check

import (
	"fmt"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// priorityNotApplicable reports any entity whose `priority` frontmatter
// value is present on a kind that does not carry its own priority
// (G-0078, E-0066) — every kind except gap and decision
// (entity.CarriesOwnPriority). It is the mechanical backstop for the
// "priority applies to gap and decision only" design decision.
//
// Unlike area's requiredness-only enforcement, this checks *presence*:
// the tree loader does not blank an out-of-scope kind's stored value
// (see the Priority field's doc comment on entity.Entity), so the
// value survives to load time for this check to report. Severity is
// warning, consistent with area-unknown's advisory posture — no
// strictness knob, since priority carries no `aiwf.yaml: required`
// analog.
//
// Absence (empty `priority`) is never evaluated: absent and
// explicit-empty both deserialize to "" and only a present value can
// be out of scope.
func priorityNotApplicable(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if e.Priority == "" {
			continue
		}
		if entity.CarriesOwnPriority(e.Kind) {
			continue
		}
		findings = append(findings, Finding{
			Code:     CodePriorityNotApplicable,
			Severity: SeverityWarning,
			Message: fmt.Sprintf(
				"%s carries priority %q but kind %s does not carry its own priority (only gap and decision do)",
				e.ID, e.Priority, e.Kind),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "priority",
		})
	}
	return findings
}
