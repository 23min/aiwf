package check

import (
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// priorityValid reports any entity whose `priority` frontmatter value
// is present but outside the closed set (G-0078, E-0066). Unlike
// statusValid, the closed set is not kind-scoped — the same four
// levels apply everywhere the field is legal — so this check does not
// consult kind at all; whether the field is legal *for this kind* is
// the separate priority-not-applicable check.
//
// Absence (empty `priority`) is never evaluated: absent and
// explicit-empty both deserialize to "" and only a present value can
// be invalid.
func priorityValid(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if e.Priority == "" {
			continue
		}
		if entity.IsAllowedPriorityLevel(e.Priority) {
			continue
		}
		findings = append(findings, Finding{
			Code:     CodePriorityValid,
			Severity: SeverityError,
			Message: fmt.Sprintf(
				"priority %q is not in the closed set (allowed: %s)",
				e.Priority, strings.Join(entity.AllowedPriorityLevels(), ", ")),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "priority",
		})
	}
	return findings
}
