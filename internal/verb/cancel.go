package verb

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// Cancel promotes an entity to its kind's terminal-cancel status —
// `cancelled` for epic/milestone, `rejected` for adr/decision,
// `wontfix` for gap, `retired` for contract. Errors when the entity is
// already in a terminal state or when the kind is unknown.
//
// reason is optional free-form prose; when non-empty, it lands in the
// commit body so the cancellation's "why" is preserved for future
// readers. Empty reason matches today's body-less behaviour.
//
// force=true emits an `aiwf-force: <reason>` trailer alongside the
// standard ones so the cancellation is auditable as a forced action.
// Cancel has no FSM transition rule to relax (it always sets status to
// the kind's terminal-cancel target), so force is purely an audit
// signal here. The "already at target" guard remains in place even
// under force — there is no diff to write. Force requires a non-empty
// reason; the caller is responsible for enforcing that.
func Cancel(ctx context.Context, t *tree.Tree, id, actor, reason string, force bool) (*Result, error) {
	_ = ctx
	if entity.IsCompositeID(id) {
		return cancelAC(t, id, actor, reason, force)
	}
	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("entity %q not found", id)
	}
	// Pre-flight terminal check. Cancel never makes sense on an entity
	// already at a terminal status — there's nothing to project to.
	// Without this guard, the older code silently constructed
	// FSM-illegal projections (e.g., Cancel on a `done` epic set
	// status to `cancelled` even though Epic.done has no outgoing
	// edges); since M-0131's state-aware CancelTarget the trap moved
	// to the empty-return path with a less informative message. This
	// catches the case once, at the verb boundary, with a clear
	// "already at terminal X" error.
	if entity.IsTerminal(e.Kind, e.Status) {
		return nil, &fsmTransitionIllegalError{msg: fmt.Sprintf("%s is already at terminal status %q; nothing to cancel", id, e.Status)}
	}
	target := entity.CancelTarget(e.Kind, e.Status)
	if target == "" {
		return nil, fmt.Errorf("%s (kind %q, status %q) has no cancel target", id, e.Kind, e.Status)
	}
	if e.Status == target {
		//coverage:ignore defensive: mathematically unreachable given the IsTerminal check above — CancelTarget only ever returns "" or one of the kind's own terminal statuses, so e.Status == target implies IsTerminal(e.Kind, e.Status) is already true, which the earlier guard already refused
		return nil, fmt.Errorf("%s is already %s", id, target)
	}

	// Cancel-cascade guards (D-0003 / D-0004): refuse-with-listing when a
	// parent still owns non-terminal children. No auto-cascade — the
	// operator disposes each child first. Runs after the terminal/target
	// checks so "already terminal" / "no cancel target" win, and before
	// any projection so the refusal is a clean typed error, not a
	// finding. Shared with Promote's own terminal-target cascade guards
	// via epicChildrenCascadeGuard/milestoneACsCascadeGuard
	// (cancel_guards.go); target is always terminal by construction
	// (entity.CancelTarget's result), so both guards' terminal-status
	// gates are trivially satisfied here.
	if err := epicChildrenCascadeGuard(t, e, target, func(children []string) error {
		return &EpicCancelNonTerminalChildrenError{Epic: e.ID, Children: children}
	}); err != nil {
		return nil, err
	}
	if err := milestoneACsCascadeGuard(e, target, func(openACs []string) error {
		return &MilestoneCancelNonTerminalACsError{Milestone: e.ID, ACs: openACs}
	}); err != nil {
		return nil, err
	}

	modified := *e
	modified.Status = target

	body, err := readBody(t.Root, e.Path)
	if err != nil {
		//coverage:ignore defensive: e.Path comes from the loaded tree, so the file is present; a read error needs the file to vanish mid-verb
		return nil, err
	}
	content, err := entity.Serialize(&modified, body)
	if err != nil {
		//coverage:ignore defensive: Serialize fails only on a malformed entity; e already round-tripped through the loader
		return nil, fmt.Errorf("serializing %s: %w", id, err)
	}

	proj := projectReplace(t, &modified, filepath.ToSlash(e.Path))
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		//coverage:ignore defensive: Cancel only ever writes a status flip to one of the kind's own terminal-cancel values, and the two known cascade-shaped findings this could otherwise introduce (epic-cancel-non-terminal-children, milestone-cancelled-incomplete-acs) are already refused above by epicChildrenCascadeGuard/milestoneACsCascadeGuard before reaching this projection — this check-rule pass is defense-in-depth for a finding class not yet identified, not a reachable path through the verb's own current call graph
		return findings(fs), nil
	}

	subject := fmt.Sprintf("aiwf cancel %s -> %s", id, target)
	result := plan(&Plan{
		Subject: subject,
		Body:    reason,
		// Cancel does not emit aiwf-to:. The cancel target is implicit
		// per kind (entity.CancelTarget) and the verb name itself
		// communicates the destination — no need for a structured
		// trailer to disambiguate. Only `promote` events carry aiwf-to:.
		Trailers: transitionTrailers("cancel", id, actor, reason, "", force),
		Ops:      []FileOp{{Type: OpWrite, Path: e.Path, Content: content}},
	})
	result.Metadata = map[string]any{"entity_id": id, "from": e.Status, "to": target}
	return result, nil
}
