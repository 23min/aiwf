package check

import (
	"fmt"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// terminalEntityNotArchived reports any entity whose frontmatter
// status is terminal but whose file still lives in an active dir
// (i.e., not yet swept into archive/). This is the normal transient
// state under ADR-0004's decoupled model — entities reach a terminal
// status in their active location and are swept later by an explicit
// `aiwf archive --apply` invocation.
//
// Severity is warning (advisory). Per ADR-0004 §"Drift control" layer
// (1) and §"Check shape rules": "Advisory by default; not blocking."
// The threshold knob (`archive.sweep_threshold`) lands in M-0088 and
// will flip the severity to error past N. Until then, the rule is
// purely informational — it counts the pending sweep, and the
// archive-sweep-pending aggregate finding (AC-3) summarizes the count.
//
// The rule is location-keyed to active dirs: a terminal entity that
// has already been swept into archive/ is the in-the-clear case and
// stays silent.
func terminalEntityNotArchived(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		if e.Status == "" {
			continue
		}
		if !entity.IsAllowedStatus(e.Kind, e.Status) {
			continue
		}
		if !entity.IsTerminal(e.Kind, e.Status) {
			continue
		}
		findings = append(findings, Finding{
			Code:     "terminal-entity-not-archived",
			Severity: SeverityWarning,
			Message: fmt.Sprintf("entity %s has terminal status %q but file is still in the active tree; awaiting `aiwf archive --apply` sweep",
				e.ID, e.Status),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "status",
		})
	}
	return findings
}

// archiveSweepPending is the aggregate finding that summarizes the
// count of pending sweeps. Per ADR-0004 §"Drift control" layer (1)
// and §"Check shape rules":
//
//	archive-sweep-pending — aggregate finding reporting the count
//	of terminal-entity-not-archived instances. Advisory;
//	configurable to blocking past archive.sweep_threshold.
//	"Hidden when zero." (§Drift control)
//
// The threshold knob (M-0088) escalates the aggregate to error via
// ApplyArchiveSweepThreshold, applied by the verb dispatcher after
// Run. The rule's own emission stays config-agnostic.
//
// The aggregate is per-tree, so it carries no Path or EntityID. The
// per-file `terminal-entity-not-archived` findings remain alongside
// — the aggregate summarizes; it does not replace the leaves.
func archiveSweepPending(t *tree.Tree) []Finding {
	count := CountPendingSweep(t)
	if count == 0 {
		return nil
	}
	return []Finding{{
		Code:     "archive-sweep-pending",
		Severity: SeverityWarning,
		Message: fmt.Sprintf("%d terminal entities awaiting `aiwf archive --apply`. Set `archive.sweep_threshold` in aiwf.yaml to escalate to blocking past N",
			count),
	}}
}

// CountPendingSweep returns the number of terminal-status entities
// still in the active tree (i.e. the count of pending sweeps). The
// same predicate as archiveSweepPending and terminalEntityNotArchived
// — extracted so the verb dispatcher can compute the count once and
// hand it to ApplyArchiveSweepThreshold without duplicating the
// iteration logic.
//
// Exported so callers outside the check package can read the value;
// it is the same number the aggregate finding's Message names.
func CountPendingSweep(t *tree.Tree) int {
	var count int
	for _, e := range t.Entities {
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		if e.Status == "" {
			continue
		}
		if !entity.IsAllowedStatus(e.Kind, e.Status) {
			continue
		}
		if !entity.IsTerminal(e.Kind, e.Status) {
			continue
		}
		count++
	}
	return count
}

// ApplyArchiveSweepThreshold bumps the aggregate `archive-sweep-pending`
// finding from warning to error when the consumer has set
// `archive.sweep_threshold` in aiwf.yaml and the pending-sweep count
// **strictly exceeds** that threshold (M-0088 AC-2). Mutates the
// findings slice in place; no-op when set=false (default-permissive)
// or when count ≤ threshold (consumer's declared ceiling not breached).
//
// The escalation rewrites the aggregate's Message so the human
// reading `aiwf check` output sees both the count and their declared
// threshold cited explicitly. Per-file `terminal-entity-not-archived`
// leaf findings are NOT escalated — the aggregate is the single
// actionable signal; escalating leaves would flood the operator with
// duplicate "this gap is pending" warnings once they have already
// seen the aggregate.
//
// Per ADR-0004 §"Drift control" (layer 2): "Configurable hard
// threshold. aiwf.yaml's archive.sweep_threshold (default unset)
// flips the advisory finding to blocking past the named count."
//
// Callers run this AFTER Run() (or after appending the rule's
// findings to their own slice) so the rule's emission stays config-
// agnostic and the strictness bump is a separate, testable transform.
// The threshold is read via config.Config.ArchiveSweepThreshold; the
// count is the number of pending sweeps (i.e. the same value the
// aggregate's Message already names).
func ApplyArchiveSweepThreshold(findings []Finding, threshold int, set bool, count int) {
	if !set {
		return
	}
	if count <= threshold {
		return
	}
	for i := range findings {
		if findings[i].Code != "archive-sweep-pending" {
			continue
		}
		findings[i].Severity = SeverityError
		findings[i].Message = fmt.Sprintf(
			"%d terminal entities awaiting `aiwf archive --apply` (exceeds `archive.sweep_threshold: %d` in aiwf.yaml; the threshold is the consumer-declared ceiling past which the aggregate finding blocks)",
			count, threshold,
		)
	}
}

// archivedEntityNotTerminal reports any entity whose file lives under
// a per-kind `archive/` subdirectory but whose frontmatter status is
// not terminal. This is the hand-edit-drift case ADR-0004 §"Reversal"
// describes: someone took an archived entity's status off-terminal in
// the markdown, and the loader picked the file up from its archive
// location at the next read.
//
// Severity is error (blocking). Per ADR-0004 §"Reversal": there is no
// auto-reconciliation in the active→terminal direction triggered by
// hand-edits in the reverse direction. The remediation the hint names
// is to revert the hand-edit, not to relocate the file.
//
// The rule is location-keyed: it only fires for paths under archive/.
// The inverse drift (terminal status, active dir) is covered by
// terminalEntityNotArchived as an advisory finding.
func archivedEntityNotTerminal(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if !entity.IsArchivedPath(e.Path) {
			continue
		}
		// frontmatterShape already reports an empty status; skip it
		// here so the user sees one finding, not two, for the same
		// authoring problem.
		if e.Status == "" {
			continue
		}
		// statusValid already reports an unknown status; same
		// rationale as the empty-status skip above.
		if !entity.IsAllowedStatus(e.Kind, e.Status) {
			continue
		}
		if entity.IsTerminal(e.Kind, e.Status) {
			continue
		}
		findings = append(findings, Finding{
			Code:     "archived-entity-not-terminal",
			Severity: SeverityError,
			Message: fmt.Sprintf("entity %s lives under archive/ but status %q is not terminal; archive is the structural projection of FSM-terminality (ADR-0004 §Reversal)",
				e.ID, e.Status),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "status",
		})
	}
	return findings
}
