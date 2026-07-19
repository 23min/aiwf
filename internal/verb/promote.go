package verb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// PromoteOptions carries optional fields for Promote — resolver
// pointers (gap.addressed_by / gap.addressed_by_commit, adr.superseded_by)
// that need to be written atomically with the status change so the
// matching check rule (gap-addressed-has-resolver, adr-supersession-mutual)
// is satisfied without a follow-up hand-edit.
//
// AddressedBy / AddressedByCommit are valid only when the target is a
// gap and newStatus is "addressed". SupersededBy is valid only when
// the target is an ADR and newStatus is "superseded". Mismatches return
// a Go error before any disk work — usage misalignment, not a finding.
//
// When a slice or string is set, it replaces the existing field on the
// entity (this is a one-shot setting at status-change time, not a
// merge). Unset fields leave the entity's existing values untouched.
type PromoteOptions struct {
	AddressedBy       []string
	AddressedByCommit []string
	SupersededBy      string
}

func (o PromoteOptions) hasResolverFlag() bool {
	return len(o.AddressedBy) > 0 || len(o.AddressedByCommit) > 0 || o.SupersededBy != ""
}

// Promote advances an entity's status. The transition is validated
// against the kind's FSM (entity.ValidateTransition) before any
// projection runs, so unknown statuses and illegal jumps are rejected
// with a clear error rather than as a `status-valid` finding.
//
// reason is optional free-form prose explaining *why* the transition
// happens. When non-empty, it lands in the commit body (between
// subject and trailers) so future readers can see the why, not just
// the what. Empty reason produces a body-less commit.
//
// force=true relaxes the FSM transition rule so any-to-any moves are
// permitted; coherence (closed-set membership of the target status,
// id format, ref resolution) still runs via projection findings, so
// promoting to an unknown status is still rejected. Force requires a
// non-empty reason; the caller (cmd dispatcher) is responsible for
// enforcing that. When force is set, the standard trailers gain
// `aiwf-force: <reason>` so the audit trail is queryable.
//
// opts carries optional resolver pointers that need to be written in
// the same commit as the status change (see PromoteOptions). The
// resolver is validated against the entity's kind and the target
// status before any disk work — a mismatch is a Go error.
//
// Returns a Go error for "couldn't even start": id not found, illegal
// transition (when not forced), resolver-flag/kind/status mismatch.
// Tree-level findings caused by the change are returned as a Result
// with non-empty Findings.
func Promote(ctx context.Context, t *tree.Tree, id, newStatus, actor, reason string, force bool, opts PromoteOptions) (*Result, error) {
	if entity.IsCompositeID(id) {
		if opts.hasResolverFlag() {
			return nil, fmt.Errorf("resolver flags (--by/--by-commit/--superseded-by) are not valid for AC promotions")
		}
		return promoteAC(t, id, newStatus, actor, reason, force)
	}
	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("entity %q not found", id)
	}
	// Back-fill carve-out (G-0096): if the requested transition is to
	// the entity's current status, that status is a resolution-class
	// terminal (gap addressed / ADR superseded), the entity's resolver
	// field is currently empty, and a resolver flag is provided, treat
	// the call as a metadata-only update — write the resolver, leave
	// status alone, skip ValidateTransition (which would reject the
	// same-status move). M-059's design assumed resolver pointers
	// always rode the status-change commit; back-fill cleans up
	// pre-enforcement strays without needing --force.
	isBackfill := e.Status == newStatus &&
		isResolutionClassStatus(e.Kind, newStatus) &&
		needsResolverBackfill(e, opts)
	if !force && !isBackfill {
		if err := entity.ValidateTransition(e.Kind, e.Status, newStatus); err != nil {
			return nil, err
		}
	}
	if err := validateResolverFlags(e.Kind, newStatus, opts); err != nil {
		return nil, err
	}
	// Require resolver pointer on resolution-class transitions
	// (G-0096): without this, gaps and ADRs can land in the
	// `addressed`/`superseded` terminal state with no resolver and no
	// way back without --force. The check rules
	// gap-addressed-has-resolver and adr-supersession-mutual surface the
	// problem post-hoc but they are warnings, not errors, so the
	// pre-push hook does not block. Verb-time enforcement is the
	// chokepoint. --force bypasses for sovereign overrides.
	if !force {
		if err := requireResolverForResolutionClass(e.Kind, newStatus, opts); err != nil {
			return nil, err
		}
		// Validate that each --by-commit SHA resolves to a real commit
		// in the repo (G-0186), mirroring how --by validates entity ids
		// via tree.ByID. Without this, a well-formed-but-fake SHA (e.g.
		// "8f3c2a1") lands in addressed_by_commit verbatim and reads as
		// authoritative while pointing at nothing. --force bypasses for
		// sovereign overrides — an operator may legitimately reference a
		// commit not yet present locally (an unmerged fixing branch, a
		// cross-repo reference). The validation runs on the normal path
		// only, matching the --force stance of the resolver-requirement
		// and sovereign-act checks that bracket it.
		if err := validateAddressedByCommit(ctx, t.Root, opts); err != nil {
			return nil, err
		}
		// Sovereign-act-shape transitions are human-only by default
		// (the closed-set list lives at entity.IsSovereignActShape;
		// M-0095 is the first entry — epic proposed → active per
		// G-0063). --force is the explicit override (and itself
		// enforces human-only via the existing provenance coherence
		// rule, so non-human + --force already fails at the coherence
		// chokepoint).
		if err := requireHumanActorForSovereignAct(e.Kind, e.Status, newStatus, actor); err != nil {
			return nil, err
		}
		// G-0269: an epic proposed → active or milestone → in_progress
		// promote is a sovereign activating act that must land on
		// ADR-0010's expected parent branch. A concurrent session
		// checking out a different branch in the same shared worktree
		// between the operator's preflight and this commit would
		// otherwise land the commit wherever HEAD now happens to point,
		// silently. --force bypasses (sovereign override, same as the
		// other guards in this block).
		if err := requireExpectedBranchForActivatingTransition(ctx, t, e, newStatus); err != nil {
			return nil, err
		}
		// M-0268/AC-1 (D-0039 point 1): a milestone starting with zero
		// acceptance criteria has no contract yet for what "done"
		// means. --force bypasses (sovereign override), same stance as
		// the resolver-requirement checks above.
		if err := requireNonEmptyACsAtMilestoneStart(e, newStatus); err != nil {
			return nil, err
		}
		// M-0268/AC-2 (G-0216): a milestone starting with an AC whose
		// body is a title-only stub has no real contract for that
		// criterion yet. --force bypasses, same stance as AC-1's guard
		// above.
		if err := requireNonEmptyACBodiesAtMilestoneStart(t, e, newStatus); err != nil {
			return nil, err
		}
	}

	// Epic-terminal-promote cascade guard (G-0393 / G-0394, two
	// independently-filed gaps converging on the same fix), mirroring
	// Cancel's own EpicCancelNonTerminalChildrenError (D-0003): refuse,
	// don't auto-cascade, when an epic is about to reach a terminal
	// status (done or cancelled — both legal Promote targets for
	// KindEpic, not just Cancel's own path) while it still owns a
	// non-terminal child milestone. Runs unconditionally, even under
	// --force, matching Cancel's guard — force relaxes FSM-transition
	// legality, not this structural children precondition. Without it,
	// `aiwf promote <epic> done` (or `cancelled`, bypassing Cancel's own
	// dedicated guard) can leave a non-terminal milestone under a
	// terminal, later archived epic — a state `aiwf check`'s
	// archived-entity-not-terminal rule only catches after the fact.
	// Archive's own independent subtree-terminality guard
	// (internal/verb/archive.go) is the defense-in-depth backstop for a
	// raw frontmatter hand-edit that bypasses this verb entirely.
	if e.Kind == entity.KindEpic && entity.IsTerminal(entity.KindEpic, newStatus) {
		if nonTerminal := nonTerminalEpicChildren(t, e.ID); len(nonTerminal) > 0 {
			return nil, &EpicPromoteNonTerminalChildrenError{Epic: e.ID, NewStatus: newStatus, Children: nonTerminal}
		}
	}

	// Milestone-cancel-promote cascade guard (G-0335), mirroring the
	// epic guard above: refuse, don't auto-cascade, when a milestone is
	// about to reach `cancelled` while it still carries an open
	// acceptance criterion — matching Cancel's own
	// MilestoneCancelNonTerminalACsError (D-0004). Scoped to `cancelled`
	// only, not IsTerminal generally: the `done` target already carries
	// this precondition via the milestone-done-incomplete-acs check-rule
	// that projectionFindings runs below, but that rule only fires on
	// status: done, so it would never catch a cancelled milestone with
	// an open AC — nothing downstream would. Runs unconditionally, even
	// under --force, matching Cancel's guard — force relaxes
	// FSM-transition legality, not this structural AC precondition.
	if e.Kind == entity.KindMilestone && newStatus == entity.StatusCancelled {
		if ok, openACs := entity.MilestoneCanGoDone(e); !ok {
			composite := make([]string, 0, len(openACs))
			for _, ac := range openACs {
				composite = append(composite, e.ID+"/"+ac)
			}
			return nil, &MilestonePromoteNonTerminalACsError{Milestone: e.ID, NewStatus: newStatus, ACs: composite}
		}
	}

	modified := *e
	modified.Status = newStatus
	applyResolverFlags(&modified, opts)

	body, err := readBody(t.Root, e.Path)
	if err != nil {
		return nil, err
	}
	content, err := entity.Serialize(&modified, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", id, err)
	}

	ops := []FileOp{{Type: OpWrite, Path: e.Path, Content: content}}
	proj := projectReplace(t, &modified, filepath.ToSlash(e.Path))

	// Reciprocal supersedes back-link (G-0255): when --superseded-by
	// names another ADR present in the tree, record this ADR in that
	// ADR's supersedes set so adr-supersession-mutual is satisfied on
	// both sides within this one commit. Without the back-link the verb
	// breaks its own promise — requireResolverForResolutionClass demands
	// --superseded-by "so the adr-supersession-mutual rule is satisfied",
	// yet the rule checks a two-sided invariant. Multi-file single-commit
	// mirrors reallocate's cross-reference rewrites.
	recOp, recModified, err := reciprocalSupersedesOp(t, modified.ID, opts.SupersededBy)
	if err != nil {
		//coverage:ignore defensive: reciprocalSupersedesOp errors only on read/serialize of an entity that already round-tripped through the loader
		return nil, err
	}
	if recOp != nil {
		ops = append(ops, *recOp)
		proj = projectReplace(proj, recModified, filepath.ToSlash(recOp.Path))
	}

	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}

	subject := fmt.Sprintf("aiwf promote %s %s -> %s", id, e.Status, newStatus)
	result := plan(&Plan{
		Subject:  subject,
		Body:     reason,
		Trailers: transitionTrailers("promote", id, actor, reason, newStatus, force),
		Ops:      ops,
	})
	result.Metadata = map[string]any{"entity_id": id, "from": e.Status, "to": newStatus}
	return result, nil
}

// fsmTransitionIllegalError wraps a legality refusal that isn't itself
// produced by entity.ValidateTransition — an AC composite-id status or
// tdd_phase transition (ac.go's promoteAC / PromoteACPhase, which
// validate against entity.IsLegalACTransition /
// IsLegalTDDPhaseTransition rather than a Kind-keyed FSM, since AC
// status isn't one of the six entity kinds), or Cancel's own
// already-terminal pre-check below — but is the same class of refusal
// entity.ValidateTransition's own FSMTransitionError reports for
// kind-level transitions. Carrying the same CodeFSMTransitionIllegal
// lets a Coded consumer (entity.Code) recognize an FSM-illegal
// transition refusal uniformly, regardless of which pre-flight check
// caught it, without changing any existing message text (message-
// matching consumers, and Cancel's own documented "already at
// terminal X" phrasing, are unaffected).
type fsmTransitionIllegalError struct{ msg string }

// Error implements error, returning msg unchanged.
func (e *fsmTransitionIllegalError) Error() string { return e.msg }

// Code returns entity.CodeFSMTransitionIllegal's ID, satisfying
// entity.Coded.
func (e *fsmTransitionIllegalError) Code() string { return entity.CodeFSMTransitionIllegal.ID }

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
		return nil, fmt.Errorf("%s is already %s", id, target)
	}

	// Cancel-cascade guards (D-0003 / D-0004): refuse-with-listing when a
	// parent still owns non-terminal children. No auto-cascade — the
	// operator disposes each child first. Runs after the terminal/target
	// checks so "already terminal" / "no cancel target" win, and before
	// any projection so the refusal is a clean typed error, not a finding.
	switch e.Kind {
	case entity.KindEpic:
		if nonTerminal := nonTerminalEpicChildren(t, e.ID); len(nonTerminal) > 0 {
			return nil, &EpicCancelNonTerminalChildrenError{Epic: e.ID, Children: nonTerminal}
		}
	case entity.KindMilestone:
		if ok, openACs := entity.MilestoneCanGoDone(e); !ok {
			composite := make([]string, 0, len(openACs))
			for _, ac := range openACs {
				composite = append(composite, e.ID+"/"+ac)
			}
			return nil, &MilestoneCancelNonTerminalACsError{Milestone: e.ID, ACs: composite}
		}
	default:
		// Other kinds (ADR, gap, decision, contract) own no child
		// entities or ACs, so cancel carries no cascade precondition —
		// the terminal/target checks above are the only guards they need.
	}

	modified := *e
	modified.Status = target

	body, err := readBody(t.Root, e.Path)
	if err != nil {
		return nil, err
	}
	content, err := entity.Serialize(&modified, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", id, err)
	}

	proj := projectReplace(t, &modified, filepath.ToSlash(e.Path))
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
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

// transitionTrailers builds the standard trailer block for a status-
// changing verb. `to` is the target status when relevant (`promote`
// events; emitted as `aiwf-to: <to>`); pass an empty string for verbs
// whose target is implicit in the verb name (cancel). The `aiwf-force`
// trailer is appended only when force is true; its value is the
// trimmed reason (which the dispatcher has already verified is non-
// empty). The standard trailers come first so downstream readers
// (`aiwf history`) find them in a stable order: verb, entity, actor,
// to (when present), force (when present).
func transitionTrailers(verbName, id, actor, reason, to string, force bool) []gitops.Trailer {
	// Canonicalize the entity-id trailer per AC-1 in M-081: new
	// kernel commits never re-emit narrow legacy widths, even when
	// the verb was invoked with a narrow id (which AC-2 tolerates
	// at the lookup layer).
	trailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: verbName},
		{Key: gitops.TrailerEntity, Value: entity.Canonicalize(id)},
		{Key: gitops.TrailerActor, Value: actor},
	}
	if to != "" {
		trailers = append(trailers, gitops.Trailer{Key: gitops.TrailerTo, Value: to})
	}
	if force {
		trailers = append(trailers, gitops.Trailer{Key: gitops.TrailerForce, Value: strings.TrimSpace(reason)})
	}
	return trailers
}

// validateResolverFlags returns an error if any field in opts is set
// for a kind/target-status combination it doesn't apply to. Resolver
// fields are tied to specific transitions: addressed_by/addressed_by_commit
// to gap → addressed, superseded_by to adr → superseded. Any other
// combination is a usage misalignment — return a Go error so the
// dispatcher exits with the right code rather than producing a
// projection finding the user has to interpret.
func validateResolverFlags(k entity.Kind, newStatus string, opts PromoteOptions) error {
	if len(opts.AddressedBy) > 0 || len(opts.AddressedByCommit) > 0 {
		if k != entity.KindGap {
			return fmt.Errorf("--by/--by-commit are only valid for gap entities; got kind %q", k)
		}
		if newStatus != entity.StatusAddressed {
			return fmt.Errorf("--by/--by-commit are only valid when promoting to %q; got %q", entity.StatusAddressed, newStatus)
		}
	}
	if opts.SupersededBy != "" {
		if k != entity.KindADR {
			return fmt.Errorf("--superseded-by is only valid for ADR entities; got kind %q", k)
		}
		if newStatus != entity.StatusSuperseded {
			return fmt.Errorf("--superseded-by is only valid when promoting to %q; got %q", entity.StatusSuperseded, newStatus)
		}
	}
	return nil
}

// isResolutionClassStatus reports whether (kind, status) is a
// "resolution-class" terminal — one whose semantics require a resolver
// pointer (gap.addressed_by[_commit], adr.superseded_by) per M-059.
// Today there are exactly two such pairs: (gap, addressed) and
// (adr, superseded). G-0096 introduces verb-time enforcement that
// resolver pointers ride these transitions; the same set drives the
// same-status back-fill carve-out.
func isResolutionClassStatus(k entity.Kind, status string) bool {
	return (k == entity.KindGap && status == entity.StatusAddressed) ||
		(k == entity.KindADR && status == entity.StatusSuperseded)
}

// needsResolverBackfill reports whether e is in a resolution-class
// terminal status with an empty resolver field AND opts provides a
// resolver flag of the matching kind. The empty-current-resolver
// condition is what keeps the carve-out from re-purposing this verb
// path into a generic "rewrite the resolver" surface — once a resolver
// is set, further changes need a deliberate verb (or --force).
func needsResolverBackfill(e *entity.Entity, opts PromoteOptions) bool {
	switch e.Kind {
	case entity.KindGap:
		currentEmpty := len(e.AddressedBy) == 0 && len(e.AddressedByCommit) == 0
		incoming := len(opts.AddressedBy) > 0 || len(opts.AddressedByCommit) > 0
		return currentEmpty && incoming
	case entity.KindADR:
		return e.SupersededBy == "" && opts.SupersededBy != ""
	default:
		// Other kinds have no resolver fields; back-fill is not
		// applicable. The caller's isResolutionClassStatus check
		// already returned false for these kinds, so this branch is
		// unreached in practice — kept for the design-intent check.
		return false
	}
}

// requireResolverForResolutionClass returns an error when the target
// (kind, newStatus) is a resolution-class terminal but opts carries no
// matching resolver flag. M-059 made the flags possible; G-0096 makes
// them mandatory at the verb chokepoint so the gap-addressed-has-resolver
// and adr-supersession-mutual warnings cannot be reached via the verb.
// --force bypasses (sovereign override path); the caller checks force
// before invoking this.
func requireResolverForResolutionClass(k entity.Kind, newStatus string, opts PromoteOptions) error {
	switch {
	case k == entity.KindGap && newStatus == entity.StatusAddressed:
		if len(opts.AddressedBy) == 0 && len(opts.AddressedByCommit) == 0 {
			return fmt.Errorf("promoting a gap to %q requires --by <entity-id> or --by-commit <sha> so the gap-addressed-has-resolver rule is satisfied; pass --force to override", entity.StatusAddressed)
		}
	case k == entity.KindADR && newStatus == entity.StatusSuperseded:
		if opts.SupersededBy == "" {
			return fmt.Errorf("promoting an ADR to %q requires --superseded-by <ADR-id> so the adr-supersession-mutual rule is satisfied; pass --force to override", entity.StatusSuperseded)
		}
	}
	return nil
}

// requireNonEmptyACsAtMilestoneStart returns an error when a
// milestone's draft -> in_progress promote would start a milestone
// with no acceptance criteria at all (D-0039 point 1 / M-0268/AC-1).
// Zero ACs means there is no contract yet for what "done" means; the
// operator either writes at least one AC first or overrides with
// --force. Scoped narrowly to the draft -> in_progress transition —
// no other legal milestone transition carries this precondition.
//
// This is a soft precondition, not a structural invariant like
// MilestonePromoteNonTerminalACsError / EpicPromoteNonTerminalChildren-
// Error (which run unconditionally, even under --force): D-0039 point
// 2 explicitly permits a milestone to reach done with zero ACs (only a
// warning fires), so "permanently AC-less" is a legitimate end state,
// not an inconsistency force would be papering over. The caller checks
// force before invoking this, matching requireResolverForResolutionClass's
// own --force stance.
func requireNonEmptyACsAtMilestoneStart(e *entity.Entity, newStatus string) error {
	if e.Kind != entity.KindMilestone || e.Status != entity.StatusDraft || newStatus != entity.StatusInProgress {
		return nil
	}
	if len(e.ACs) == 0 {
		return fmt.Errorf("cannot promote %s to %q: milestone has no acceptance criteria; add one with `aiwf add ac %s --title \"...\"` first, or pass --force to override", e.ID, newStatus, e.ID)
	}
	return nil
}

// requireNonEmptyACBodiesAtMilestoneStart returns an error when a
// milestone's draft -> in_progress promote would start a milestone
// while at least one AC's body subsection carries no non-heading
// prose (G-0216 / M-0268/AC-2). A title-only AC stub is no real
// contract for that criterion yet. Scoped narrowly to draft ->
// in_progress, same as requireNonEmptyACsAtMilestoneStart.
//
// An AC with NO `### AC-N` heading in the body at all is a different
// problem — a frontmatter/body desync the acs-body-coherence/
// missing-heading check rule already covers — so it is skipped here,
// not double-flagged as "empty."
//
// Soft precondition, not a structural invariant: --force bypasses,
// matching requireNonEmptyACsAtMilestoneStart's own stance. The
// caller checks force before invoking this.
func requireNonEmptyACBodiesAtMilestoneStart(t *tree.Tree, e *entity.Entity, newStatus string) error {
	if e.Kind != entity.KindMilestone || e.Status != entity.StatusDraft || newStatus != entity.StatusInProgress {
		return nil
	}
	body, err := readBody(t.Root, e.Path)
	if err != nil {
		//coverage:ignore defensive: e.Path comes from the loaded tree, so the file is present; a read error needs the file to vanish mid-verb
		return fmt.Errorf("reading body of %s: %w", e.ID, err)
	}
	sections := entity.ParseACSections(body)
	for _, ac := range e.ACs {
		content, found := sections[ac.ID]
		if !found {
			continue
		}
		if entity.ACSectionIsEmpty(content) {
			return fmt.Errorf("cannot promote %s to %q: %s/%s has no body content; write prose under its `### %s` heading first, or pass --force to override", e.ID, newStatus, e.ID, ac.ID, ac.ID)
		}
	}
	return nil
}

// validateAddressedByCommit returns an error if any --by-commit SHA in
// opts fails either of two checks on the non-force path:
//
//  1. Existence (G-0186): the SHA resolves to a commit in the repo at
//     root, via gitops.CommitExists (`git rev-parse --verify --quiet
//     <sha>^{commit}`, which accepts abbreviated SHAs natively — the
//     legitimate value f7fd1f99 is a short SHA). A resolver pointer that
//     points at nothing is worse than an empty field, since it reads as
//     authoritative.
//
//  2. Reachability (G-0355): the SHA is an ancestor of HEAD, via
//     gitops.IsAncestor. A gap's addressed_by_commit is a claim that the
//     gap is closed by that commit; the claim is only truthful if the
//     commit is in the history we are recording the closure onto. HEAD —
//     not a configured trunk ref — is the right anchor: the verb may run
//     on any branch, and "the history this promote commits onto" is
//     exactly what must contain the commit. In the wf-patch wrap the
//     tracker closure runs *after* the merge to mainline, so HEAD is
//     mainline and this asserts trunk-reachability directly. This is the
//     mechanical backstop for the G-0346 failure: a merge that did not
//     land leaves the fixing commit off HEAD, so recording the closure
//     is refused rather than silently corrupting the resolver.
//
// The caller invokes this on the non-force path only; --force is the
// sovereign override for the documented exceptions (a cross-repo
// reference, or a commit on an unmerged fixing branch the operator
// records on their own authority).
//
// A nil/empty AddressedByCommit makes this a no-op — the loop body never
// runs, so a promote without --by-commit is unaffected.
func validateAddressedByCommit(ctx context.Context, root string, opts PromoteOptions) error {
	for _, sha := range opts.AddressedByCommit {
		ok, err := gitops.CommitExists(ctx, root, sha)
		if err != nil {
			//coverage:ignore defensive: CommitExists maps an unresolvable sha to (false,nil); a non-nil err needs git absent or a broken workdir, not reachable deterministically in-process
			return fmt.Errorf("checking --by-commit %q resolves: %w", sha, err)
		}
		if !ok {
			return fmt.Errorf("--by-commit %q does not resolve to a commit in this repo; pass a real commit SHA, or --force to record it anyway (sovereign override)", sha)
		}
		reachable, err := gitops.IsAncestor(ctx, root, sha, "HEAD")
		if err != nil {
			//coverage:ignore defensive: IsAncestor maps "not an ancestor" to (false,nil); a non-nil err needs a bad ref or broken workdir, not reachable deterministically in-process
			return fmt.Errorf("checking --by-commit %q is reachable from HEAD: %w", sha, err)
		}
		if !reachable {
			return fmt.Errorf("--by-commit %q resolves to a commit not reachable from HEAD; the closure would record a commit this branch does not contain (did a merge fail to land?). Reconcile and merge first, or pass --force to record it anyway (sovereign override)", sha)
		}
	}
	return nil
}

// applyResolverFlags writes any set resolver fields from opts onto e.
// Unset fields are left alone. Replacement, not append: the flag value
// is the new state of the field.
func applyResolverFlags(e *entity.Entity, opts PromoteOptions) {
	if len(opts.AddressedBy) > 0 {
		e.AddressedBy = opts.AddressedBy
	}
	if len(opts.AddressedByCommit) > 0 {
		e.AddressedByCommit = opts.AddressedByCommit
	}
	if opts.SupersededBy != "" {
		e.SupersededBy = opts.SupersededBy
	}
}

// reciprocalSupersedesOp returns the file write and the projected
// entity that records the back-link `supersedes` entry on the
// superseding ADR named by supersededBy, so a
// `promote <supersededID> superseded --superseded-by <supersededBy>`
// records the link on both ADRs in the verb's single commit (G-0255).
//
// Returns (nil, nil, nil) — no reciprocal write — when:
//   - supersededBy is empty (no flag passed);
//   - the named entity is absent, is not an ADR, or names the
//     superseded entity itself — the refs-resolve and no-cycles checks
//     surface those at projection time; skipping here avoids a nil-deref
//     and a self-write that would clobber the status-change op; or
//   - the superseding ADR already lists supersededID in its supersedes
//     set (idempotent: nothing to add).
func reciprocalSupersedesOp(t *tree.Tree, supersededID, supersededBy string) (*FileOp, *entity.Entity, error) {
	if supersededBy == "" {
		return nil, nil, nil
	}
	b := t.ByID(supersededBy)
	if b == nil || b.Kind != entity.KindADR || b.ID == supersededID {
		return nil, nil, nil
	}
	if slices.Contains(b.Supersedes, supersededID) {
		return nil, nil, nil
	}

	modified := *b
	// Clone before append so the loaded tree's slice is not mutated in
	// place — projectReplace swaps the entity pointer, but the backing
	// array would otherwise be shared with the original entity.
	modified.Supersedes = append(slices.Clone(b.Supersedes), supersededID)

	body, err := readBody(t.Root, b.Path)
	if err != nil {
		//coverage:ignore defensive: b.Path comes from the loaded tree, so the file is present; a read error needs the file to vanish mid-verb
		return nil, nil, fmt.Errorf("reading body of %s: %w", b.ID, err)
	}
	content, err := entity.Serialize(&modified, body)
	if err != nil {
		//coverage:ignore defensive: Serialize fails only on a malformed entity; b already round-tripped through the loader
		return nil, nil, fmt.Errorf("serializing %s: %w", b.ID, err)
	}
	return &FileOp{Type: OpWrite, Path: b.Path, Content: content}, &modified, nil
}

// readBody reads the body bytes from an existing entity file. Returns
// an empty body if the file lacks frontmatter (a freshly-edited file
// with no closing ---). Used by promote/cancel/reallocate to preserve
// body prose during frontmatter rewrites.
func readBody(root, relPath string) ([]byte, error) {
	full := filepath.Join(root, relPath)
	content, err := os.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", relPath, err)
	}
	_, body, ok := entity.Split(content)
	if !ok {
		return []byte{}, nil
	}
	return body, nil
}
