package check

import (
	"fmt"
	"sort"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// CodeEpicTerminalNonTerminalChildren is the finding code emitted by
// epicTerminalNonTerminalChildren. Typed per G-0129.
const CodeEpicTerminalNonTerminalChildren = "epic-terminal-non-terminal-children"

// epicTerminalNonTerminalChildren (error) reports any epic whose
// frontmatter status is terminal (done or cancelled) while it still
// owns one or more non-terminal child milestones. This is the standing
// backstop for verb.Promote's and verb.Cancel's own refuse-with-listing
// guards (G-0393 / D-0003): those guards close the two ordinary entry
// points to this state, but neither is the only way an epic's
// frontmatter can end up here — a hand-edit, a pre-guard binary, or a
// tree assembled by another tool can all still produce it. Unlike
// archivedEntityNotTerminal, this rule is not location-keyed to
// archive/: the invalid state (terminal epic, live child) is exactly
// as wrong whether the epic's file has already been swept or still
// lives in the active tree, so the rule scans both.
//
// Second, accidental role (G-0398): every mutating verb runs a
// before/after projection-findings gate (verb.projectionFindings) that
// refuses any commit introducing a new error-severity finding. Since
// this rule fires the instant a fresh non-terminal milestone lands
// under an already-terminal epic, it is — today — also the ONLY thing
// stopping `aiwf add milestone` / `aiwf import` from creating that
// milestone in the first place, via that generic gate rather than a
// guard purpose-built for the creation case. Neither verb has its own
// epic-status precondition (G-0398 tracks adding one). Until that
// lands, this rule's error severity is load-bearing for more than its
// own stated job — don't demote it without checking G-0398 first.
func epicTerminalNonTerminalChildren(t *tree.Tree) []Finding {
	var findings []Finding
	for _, ep := range t.ByKind(entity.KindEpic) {
		if ep.Status == "" || !entity.IsAllowedStatus(ep.Kind, ep.Status) {
			// frontmatterShape / statusValid already report these
			// shapes; skip here so the operator sees one finding, not
			// two, for the same authoring problem.
			continue
		}
		if !entity.IsTerminal(entity.KindEpic, ep.Status) {
			continue
		}
		var nonTerminal []string
		for _, m := range t.ByKind(entity.KindMilestone) {
			if entity.Canonicalize(m.Parent) != entity.Canonicalize(ep.ID) {
				continue
			}
			if m.Status == "" || !entity.IsAllowedStatus(m.Kind, m.Status) {
				continue
			}
			if entity.IsTerminal(entity.KindMilestone, m.Status) {
				continue
			}
			nonTerminal = append(nonTerminal, m.ID)
		}
		if len(nonTerminal) == 0 {
			continue
		}
		sort.Strings(nonTerminal)
		findings = append(findings, Finding{
			Code:     CodeEpicTerminalNonTerminalChildren,
			Severity: SeverityError,
			Message: fmt.Sprintf("epic %s has terminal status %q but still owns non-terminal child milestone(s) [%s]",
				ep.ID, ep.Status, strings.Join(nonTerminal, ", ")),
			Path:     ep.Path,
			EntityID: ep.ID,
			Field:    "status",
		})
	}
	return findings
}
