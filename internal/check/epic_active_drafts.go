package check

import (
	"fmt"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// epicActiveNoDraftedMilestones (warning) reports any epic at status
// `active` with zero child milestones at status `draft`. The rule is
// the kernel-side preflight signal G-0063 calls out: an active epic
// without queued draft work is a forward-motion gap — either drift
// the next milestone or wrap the epic.
//
// Reading is strict-literal (the rule's name): the trigger is "zero
// drafts among children", not "zero non-terminal children". An epic
// with milestones in `in_progress` but no `draft`-status entry still
// fires; this is intentional — the rule asks "what's queued next?",
// not "is anything in flight?". Silence the warning by drafting one
// more milestone or wrapping.
func epicActiveNoDraftedMilestones(t *tree.Tree) []Finding {
	var findings []Finding
	for _, ep := range t.ByKind(entity.KindEpic) {
		if ep.Status != entity.StatusActive {
			continue
		}
		hasDraft := false
		for _, m := range t.ByKind(entity.KindMilestone) {
			if entity.Canonicalize(m.Parent) != entity.Canonicalize(ep.ID) {
				continue
			}
			if m.Status == entity.StatusDraft {
				hasDraft = true
				break
			}
		}
		if hasDraft {
			continue
		}
		findings = append(findings, Finding{
			Code:     "epic-active-no-drafted-milestones",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("epic %s is active but has no milestones at status draft", ep.ID),
			Path:     ep.Path,
			EntityID: ep.ID,
			Field:    "status",
		})
	}
	return findings
}
