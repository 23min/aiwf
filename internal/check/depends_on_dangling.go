package check

import (
	"fmt"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// CodeDependsOnCancelled is the finding code emitted by
// dependsOnCancelled. Typed per G-0129.
const CodeDependsOnCancelled = "depends-on-cancelled"

// dependsOnCancelled (error) reports any non-terminal milestone whose
// depends_on lists a milestone that has since reached the negative
// terminal status (cancelled) — the only negative terminal in the
// milestone FSM. The dependency can never be satisfied: the edge is
// either permanently unsatisfiable or silently means nothing, so this
// fires at error severity rather than warning (G-0437, extracted from
// G-0073's friction-evidence log). A positive terminal referent (done)
// is a perfectly ordinary depends_on target and is not flagged; nor is
// a cancelled referent once the dependent itself reaches a terminal
// status — it isn't waiting on anything anymore. An unresolved
// referent id is refsResolve's concern, not this rule's.
func dependsOnCancelled(t *tree.Tree) []Finding {
	var findings []Finding
	for _, m := range t.ByKind(entity.KindMilestone) {
		if m.Status == "" || !entity.IsAllowedStatus(m.Kind, m.Status) {
			// frontmatterShape / statusValid already report these
			// shapes; skip here so the operator sees one finding, not
			// two, for the same authoring problem.
			continue
		}
		if entity.IsTerminal(entity.KindMilestone, m.Status) {
			continue
		}
		for _, dep := range m.DependsOn {
			ref := t.ByID(dep)
			if ref == nil {
				continue
			}
			if ref.Status != entity.StatusCancelled {
				continue
			}
			findings = append(findings, Finding{
				Code:     CodeDependsOnCancelled,
				Severity: SeverityError,
				Message: fmt.Sprintf("milestone %s depends on %s, which is cancelled — the dependency can never be satisfied",
					m.ID, dep),
				Path:     m.Path,
				EntityID: m.ID,
				Field:    "depends_on",
			})
		}
	}
	return findings
}
