package check

import (
	"fmt"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// CodeMilestoneTDDUndeclared is the finding code emitted by
// milestoneTDDUndeclared. Typed per G-0129 so the compiler closes on
// rename / retire across the emit site, the strict bumper, and tests.
const CodeMilestoneTDDUndeclared = "milestone-tdd-undeclared"

// milestoneTDDUndeclared (warning) fires for any non-archived
// milestone whose frontmatter lacks a `tdd:` policy. Absent `tdd:` is
// silently treated as `tdd: none` (per design-decisions §"Acceptance
// criteria and TDD"), so the AC TDD audit never engages and the
// policy decision was never recorded.
//
// This is the defense-in-depth backstop (G-0268) for the hard `--tdd`
// requirement at `aiwf add milestone` (G-0055 layer 1): the creation
// verb is the chokepoint, but it cannot see a field stripped by a
// later hand-edit, nor a milestone brought in by a path that bypasses
// the verb (`aiwf import`, a raw file write). The check is the
// authoritative surface that catches that post-creation drift.
//
// Archive scoping (M-0086, ADR-0004 §"Check shape rules"): archived
// milestones are out of scope for active linting, so the
// grandfathered done milestones — all under `<kind>/archive/` — stay
// silent. The rule is independent of acsTDDAudit: an absent `tdd:`
// produces zero acs-tdd-audit findings (that rule skips tdd: none /
// absent), so a grandfathered milestone with already-`met` ACs is not
// retroactively re-audited — it just surfaces this one warning.
//
// Severity escalation to error under `aiwf.yaml: tdd.strict: true` is
// applied separately via ApplyTDDStrict (the single source of truth
// for which codes the strict flag covers), keeping this emission
// config-agnostic.
func milestoneTDDUndeclared(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if e.Kind != entity.KindMilestone {
			continue
		}
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		// Empty string covers absent, explicit null, and empty-value
		// frontmatter alike — all three deserialize to "". Out-of-set
		// values are a parse-time concern, not this rule's.
		if e.TDD != "" {
			continue
		}
		findings = append(findings, Finding{
			Code:     CodeMilestoneTDDUndeclared,
			Severity: SeverityWarning,
			Message: fmt.Sprintf(
				"milestone %s declares no tdd: policy (absent is silently treated as tdd: none)",
				e.ID),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "tdd",
		})
	}
	return findings
}
