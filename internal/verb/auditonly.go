// Package verb — I2.5 audit-only recovery mode (G24).
//
// When a mutating verb fails partway through (e.g., `.git/index.lock`
// contention) and the operator finishes the work with a plain
// `git commit`, the framework currently goes silent — `aiwf history`
// filters the manual commit out and the audit trail has an
// unsignalled hole. The audit-only mode is the recovery path:
//
//	aiwf cancel  <id>  --audit-only --reason "<text>"
//	aiwf promote <id> <state> --audit-only --reason "<text>"
//	aiwf promote <composite-id> --phase <p> --audit-only --reason "<text>"
//
// Each mode produces an empty-diff commit carrying the standard
// trailer block plus `aiwf-audit-only: <reason>` so the commit is
// distinguishable from a normal verb commit at read time. The verb
// refuses unless the entity is *already* at the named target state —
// audit-only records what's already true; it never makes a transition
// (that's --force's job; the two are mutually exclusive per
// coherence.go).
//
// Reference: docs/pocv3/plans/provenance-model-plan.md §"Step 5b" and
// docs/pocv3/gaps.md G24.
package verb

import (
	"context"
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// PromoteAuditOnly records that <id> reached <newStatus> via a path
// that bypassed the kernel (manual commit, import, etc.). Refuses
// when the entity is not already at newStatus — audit-only never
// transitions, only documents.
//
// Composite ids dispatch to promoteACAuditOnly. Top-level ids run
// against the per-kind FSM only insofar as the closed-set membership
// of newStatus must hold (an unknown status is rejected).
func PromoteAuditOnly(ctx context.Context, t *tree.Tree, id, newStatus, actor, reason string) (*Result, error) {
	_ = ctx
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("aiwf promote --audit-only requires a non-empty --reason")
	}
	if entity.IsCompositeID(id) {
		return promoteACAuditOnly(t, id, newStatus, actor, reason)
	}
	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("entity %q not found", id)
	}
	if !isKnownStatus(e.Kind, newStatus) {
		return nil, fmt.Errorf("aiwf promote --audit-only: %q is not a recognized %s status", newStatus, e.Kind)
	}
	if e.Status != newStatus {
		return nil, fmt.Errorf("aiwf promote --audit-only: %s is at %q, not %q (audit-only records what's already true; use --force --reason to transition)", id, e.Status, newStatus)
	}
	trailers := auditOnlyTrailers("promote", id, actor, reason, newStatus)
	if err := finalizeAuditOnlyPlanCheck(trailers); err != nil {
		return nil, err
	}
	subject := fmt.Sprintf("aiwf promote %s %s [audit-only]", id, newStatus)
	return plan(&Plan{
		Subject:    subject,
		Body:       reason,
		Trailers:   trailers,
		AllowEmpty: true,
	}), nil
}

// PromoteACPhaseAuditOnly is the audit-only variant of
// PromoteACPhase: refuses unless the AC's tdd_phase already equals
// newPhase. Same trailer + empty-commit shape as the status variant.
func PromoteACPhaseAuditOnly(ctx context.Context, t *tree.Tree, compositeID, newPhase, actor, reason string) (*Result, error) {
	_ = ctx
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("aiwf promote --audit-only requires a non-empty --reason")
	}
	_, ac, err := lookupAC(t, compositeID)
	if err != nil {
		return nil, err
	}
	if !isKnownTDDPhase(newPhase) {
		return nil, fmt.Errorf("aiwf promote --audit-only --phase: %q is not a recognized tdd_phase", newPhase)
	}
	if ac.TDDPhase != newPhase {
		return nil, fmt.Errorf("aiwf promote --audit-only --phase: %s is at phase %q, not %q (audit-only records what's already true)", compositeID, ac.TDDPhase, newPhase)
	}
	trailers := auditOnlyTrailers("promote", compositeID, actor, reason, newPhase)
	if err := finalizeAuditOnlyPlanCheck(trailers); err != nil {
		return nil, err
	}
	subject := fmt.Sprintf("aiwf promote %s --phase %s [audit-only]", compositeID, newPhase)
	return plan(&Plan{
		Subject:    subject,
		Body:       reason,
		Trailers:   trailers,
		AllowEmpty: true,
	}), nil
}

// CancelAuditOnly records that <id> was cancelled via a path that
// bypassed the kernel. Refuses when the entity is not already at the
// kind's terminal-cancel target. Composite ids dispatch to
// cancelACAuditOnly (which checks against the AC `cancelled` state).
func CancelAuditOnly(ctx context.Context, t *tree.Tree, id, actor, reason string) (*Result, error) {
	_ = ctx
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("aiwf cancel --audit-only requires a non-empty --reason")
	}
	if entity.IsCompositeID(id) {
		return cancelACAuditOnly(t, id, actor, reason)
	}
	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("entity %q not found", id)
	}
	target := entity.CancelTarget(e.Kind)
	if target == "" {
		return nil, fmt.Errorf("kind %q has no cancel target", e.Kind)
	}
	if e.Status != target {
		return nil, fmt.Errorf("aiwf cancel --audit-only: %s is at %q, not the terminal-cancel target %q (audit-only records what's already true)", id, e.Status, target)
	}
	// Cancel does not emit aiwf-to: per the existing convention; the
	// terminal target is implicit per kind.
	trailers := auditOnlyTrailers("cancel", id, actor, reason, "")
	if err := finalizeAuditOnlyPlanCheck(trailers); err != nil {
		return nil, err
	}
	subject := fmt.Sprintf("aiwf cancel %s [audit-only]", id)
	return plan(&Plan{
		Subject:    subject,
		Body:       reason,
		Trailers:   trailers,
		AllowEmpty: true,
	}), nil
}

func promoteACAuditOnly(t *tree.Tree, compositeID, newStatus, actor, reason string) (*Result, error) {
	_, ac, err := lookupAC(t, compositeID)
	if err != nil {
		return nil, err
	}
	if !isKnownACStatus(newStatus) {
		return nil, fmt.Errorf("aiwf promote --audit-only: %q is not a recognized AC status", newStatus)
	}
	if ac.Status != newStatus {
		return nil, fmt.Errorf("aiwf promote --audit-only: %s is at %q, not %q (audit-only records what's already true)", compositeID, ac.Status, newStatus)
	}
	trailers := auditOnlyTrailers("promote", compositeID, actor, reason, newStatus)
	if err := finalizeAuditOnlyPlanCheck(trailers); err != nil {
		return nil, err
	}
	subject := fmt.Sprintf("aiwf promote %s %s [audit-only]", compositeID, newStatus)
	return plan(&Plan{
		Subject:    subject,
		Body:       reason,
		Trailers:   trailers,
		AllowEmpty: true,
	}), nil
}

func cancelACAuditOnly(t *tree.Tree, compositeID, actor, reason string) (*Result, error) {
	_, ac, err := lookupAC(t, compositeID)
	if err != nil {
		return nil, err
	}
	if ac.Status != entity.StatusCancelled {
		return nil, fmt.Errorf("aiwf cancel --audit-only: %s is at %q, not %q (audit-only records what's already true)", compositeID, ac.Status, entity.StatusCancelled)
	}
	trailers := auditOnlyTrailers("cancel", compositeID, actor, reason, "")
	if err := finalizeAuditOnlyPlanCheck(trailers); err != nil {
		return nil, err
	}
	subject := fmt.Sprintf("aiwf cancel %s [audit-only]", compositeID)
	return plan(&Plan{
		Subject:    subject,
		Body:       reason,
		Trailers:   trailers,
		AllowEmpty: true,
	}), nil
}

// auditOnlyTrailers builds the trailer block for any audit-only verb.
// Includes `aiwf-to:` only for promote (`to != ""`), mirroring the
// existing transitionTrailers convention. Always includes
// aiwf-audit-only with the trimmed reason.
func auditOnlyTrailers(verbName, id, actor, reason, to string) []gitops.Trailer {
	trailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: verbName},
		// Canonical width per AC-1 in M-081.
		{Key: gitops.TrailerEntity, Value: entity.Canonicalize(id)},
		{Key: gitops.TrailerActor, Value: actor},
	}
	if to != "" {
		trailers = append(trailers, gitops.Trailer{Key: gitops.TrailerTo, Value: to})
	}
	trailers = append(trailers, gitops.Trailer{Key: gitops.TrailerAuditOnly, Value: strings.TrimSpace(reason)})
	return trailers
}

// finalizeAuditOnlyPlanCheck runs the I2.5 trailer-coherence rules
// (mutex with force, audit-only-non-human, etc.) before the verb
// commits. Audit-only never carries aiwf-force or aiwf-on-behalf-of,
// but the coherence check is the canonical surface for the human-only
// rule and we want every audit-only commit to pass it.
func finalizeAuditOnlyPlanCheck(trailers []gitops.Trailer) error {
	for _, tr := range trailers {
		if err := gitops.ValidateTrailer(tr.Key, tr.Value); err != nil {
			return fmt.Errorf("aiwf audit-only: %w", err)
		}
	}
	return CheckTrailerCoherence(trailers)
}

// isKnownStatus reports whether status is a recognized status for
// the kind under the entity FSM. Used by audit-only to reject typos
// (`done` vs `Done`) before producing a malformed commit. The check
// hits the FSM map: a status with an entry (even an empty one — i.e.,
// terminal) is known; an absent key is unknown.
func isKnownStatus(k entity.Kind, status string) bool {
	if status == "" {
		return false
	}
	// AllowedTransitions returns nil for terminal states AND for
	// unknown ones; distinguish via a terminal-status probe.
	if entity.AllowedTransitions(k, status) != nil {
		return true
	}
	// Terminal-cancel statuses live in the FSM map but produce empty
	// allowed slices. The CancelTarget table covers the common cases;
	// also accept the kind's other terminals (done / superseded / etc.).
	for _, terminal := range terminalStatusesForKind(k) {
		if terminal == status {
			return true
		}
	}
	return false
}

// terminalStatusesForKind enumerates every terminal status the kind's
// FSM can reach. Hardcoded to avoid an exported reflection of the
// transitions map. Kept tight on purpose — when a new kind/status
// lands in entity/transition.go, this slice gets the new terminal in
// the same commit.
func terminalStatusesForKind(k entity.Kind) []string {
	switch k {
	case entity.KindEpic, entity.KindMilestone:
		return []string{"done", "cancelled"}
	case entity.KindADR, entity.KindDecision:
		return []string{"superseded", "rejected"}
	case entity.KindGap:
		return []string{"addressed", "wontfix"}
	case entity.KindContract:
		return []string{"retired", "rejected"}
	}
	return nil
}

// isKnownACStatus reports whether status is a recognized AC status
// (open / met / deferred / cancelled).
func isKnownACStatus(status string) bool {
	switch status {
	case "open", "met", "deferred", "cancelled":
		return true
	}
	return false
}

// isKnownTDDPhase reports whether phase is a recognized AC tdd_phase
// (red / green / refactor / done; "" is the pre-cycle entry state and
// is intentionally NOT acceptable as an audit-only target).
func isKnownTDDPhase(phase string) bool {
	switch phase {
	case "red", "green", "refactor", "done":
		return true
	}
	return false
}
