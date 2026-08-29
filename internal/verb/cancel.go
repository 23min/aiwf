package verb

import (
	"context"
	"fmt"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// Cancel promotes an entity to its kind's terminal-cancel status —
// `cancelled` for epic/milestone, `rejected` for adr/decision,
// `wontfix` for gap, `retired` for contract. An entity already at ANY
// terminal status converges to a NoOp rather than erroring: it is
// already disposed, so cancel has nothing to project (ADR-0036,
// M-0281/AC-2). An unknown kind still errors.
//
// For a composite id the same rule applies against the AC FSM's own
// terminal set, which is not the entity FSM's — see cancelAC.
//
// reason is optional free-form prose; when non-empty, it lands in the
// commit body so the cancellation's "why" is preserved for future
// readers. Empty reason matches today's body-less behaviour.
//
// force=true emits an `aiwf-force: <reason>` trailer alongside the
// standard ones so the cancellation is auditable as a forced action.
// Cancel has no FSM transition rule to relax (it always sets status to
// the kind's terminal-cancel target), so force is purely an audit
// signal here. The already-terminal convergence fires even under force —
// there is no diff to write, so there is nothing for a sovereign
// override to re-apply. Force requires a non-empty reason; the caller is
// responsible for enforcing that.
func Cancel(ctx context.Context, t *tree.Tree, id, actor, reason string, force bool) (*Result, error) {
	if entity.IsCompositeID(id) {
		return cancelAC(ctx, t, id, actor, reason, force)
	}
	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("entity %q not found", id)
	}
	if claimErr := guardClaim(ctx, t.Root, id, e.Path); claimErr != nil {
		return nil, claimErr
	}
	// Same-state convergence (M-0281/AC-2, ADR-0036): an entity already at
	// a terminal status is already disposed — cancel has nothing to project
	// to — so a re-run converges to a NoOp at exit 0 rather than an error.
	// Cancel's implicit target is "a terminal end-state", and the entity is
	// already at one — whether it got there via cancel or another path. The
	// message names the actual state, so an operator cancelling a `done`
	// entity is informed rather than misled.
	if entity.IsTerminal(e.Kind, e.Status) {
		return &Result{NoOp: true, NoOpMessage: fmt.Sprintf("%s is already at terminal status %q; nothing to cancel", id, e.Status)}, nil
	}
	// A status outside the kind's closed set is not terminal, so it falls past
	// the guard above. Refusing here keeps cancel from laundering junk into a
	// terminal status under an ordinary cancel trailer. --force gets past this
	// guard, but it is a repair path only for the kinds whose cancel target is
	// status-agnostic (epic, milestone, gap). For adr, decision and contract
	// entity.CancelTarget derives the target from the current status, so an
	// unrecognized one yields no target and the guard below refuses regardless
	// of force — restore a recognized status before cancelling those.
	if !force && !entity.IsAllowedStatus(e.Kind, e.Status) {
		return nil, &fsmTransitionIllegalError{msg: fmt.Sprintf(
			"%s status %q is not a recognized %s status; cannot cancel from it",
			id, e.Status, e.Kind)}
	}
	target := entity.CancelTarget(e.Kind, e.Status)
	if target == "" {
		return nil, fmt.Errorf("%s (kind %q, status %q) has no cancel target", id, e.Kind, e.Status)
	}
	if e.Status == target {
		//coverage:ignore defensive: mathematically unreachable given the IsTerminal check above — CancelTarget only ever returns "" or one of the kind's own terminal statuses, so e.Status == target implies IsTerminal(e.Kind, e.Status) is already true, which the earlier guard already refused
		return nil, fmt.Errorf("%s is already %s", id, target)
	}

	// Sovereign-act gate (ADR-0047): the closed set carries both epic
	// cancel edges, and this is the call site that makes them
	// enforceable here. The history audit is transition-shaped rather
	// than verb-shaped, so it observes a cancel whether or not this
	// verb consults the set — without this call the act would land at
	// exit 0 and fail the next push, which is refusal after the fact
	// rather than prevention (ADR-0040).
	//
	// Placed ahead of the cascade guards, as promote places its own:
	// who may declare the disposition is settled before the tree is
	// asked whether the disposition is currently possible. An actor
	// refused here would still be refused after disposing of every
	// child, so answering the authority question first spares them the
	// work.
	//
	// --force skips it and reaches verb.Apply's coherence guard, which
	// refuses a force trailer from a non-human actor, so both routes
	// into these edges stay closed.
	if !force {
		if err := requireHumanActorForSovereignAct("cancel", e.Kind, e.Status, target, actor); err != nil {
			return nil, err
		}
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
	subject := fmt.Sprintf("aiwf cancel %s -> %s", id, target)
	// The projection net inside planEntityWrite cannot fire for Cancel:
	// the two cascade-shaped findings a status flip could introduce
	// (epic-cancel-non-terminal-children, milestone-cancelled-incomplete-acs)
	// are already refused above by epicChildrenCascadeGuard /
	// milestoneACsCascadeGuard. It runs anyway as the uniform writer net.
	//
	// Cancel does not emit aiwf-to:. The cancel target is implicit per
	// kind (entity.CancelTarget) and the verb name itself communicates
	// the destination — no need for a structured trailer to
	// disambiguate. Only `promote` events carry aiwf-to:.
	return planEntityWrite(t, &modified, e.Path, body, entityWrite{
		subject:  subject,
		body:     reason,
		trailers: transitionTrailers("cancel", id, actor, reason, "", force),
		// Store status as plain string: Metadata is compared in-Go by
		// callers/tests, where an interface-held entity.Status fails a
		// string equality on dynamic-type mismatch (invisible through the
		// --format=json envelope). Mirrors promote.go / auditonly.go.
		metadata: map[string]any{"entity_id": id, "from": string(e.Status), "to": string(target)},
	})
}
