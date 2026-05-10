package verb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// PromoteOptions carries optional fields for Promote — resolver
// pointers (gap.addressed_by / gap.addressed_by_commit, adr.superseded_by)
// that need to be written atomically with the status change so the
// matching check rule (gap-resolved-has-resolver, adr-supersession-mutual)
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
	_ = ctx
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
	// gap-resolved-has-resolver and adr-supersession-mutual surface the
	// problem post-hoc but they are warnings, not errors, so the
	// pre-push hook does not block. Verb-time enforcement is the
	// chokepoint. --force bypasses for sovereign overrides.
	if !force {
		if err := requireResolverForResolutionClass(e.Kind, newStatus, opts); err != nil {
			return nil, err
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

	proj := projectReplace(t, &modified, filepath.ToSlash(e.Path))
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}

	subject := fmt.Sprintf("aiwf promote %s %s -> %s", id, e.Status, newStatus)
	return plan(&Plan{
		Subject:  subject,
		Body:     reason,
		Trailers: transitionTrailers("promote", id, actor, reason, newStatus, force),
		Ops:      []FileOp{{Type: OpWrite, Path: e.Path, Content: content}},
	}), nil
}

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
	target := entity.CancelTarget(e.Kind)
	if target == "" {
		return nil, fmt.Errorf("kind %q has no cancel target", e.Kind)
	}
	if e.Status == target {
		return nil, fmt.Errorf("%s is already %s", id, target)
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
	return plan(&Plan{
		Subject: subject,
		Body:    reason,
		// Cancel does not emit aiwf-to:. The cancel target is implicit
		// per kind (entity.CancelTarget) and the verb name itself
		// communicates the destination — no need for a structured
		// trailer to disambiguate. Only `promote` events carry aiwf-to:.
		Trailers: transitionTrailers("cancel", id, actor, reason, "", force),
		Ops:      []FileOp{{Type: OpWrite, Path: e.Path, Content: content}},
	}), nil
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
// them mandatory at the verb chokepoint so the gap-resolved-has-resolver
// and adr-supersession-mutual warnings cannot be reached via the verb.
// --force bypasses (sovereign override path); the caller checks force
// before invoking this.
func requireResolverForResolutionClass(k entity.Kind, newStatus string, opts PromoteOptions) error {
	switch {
	case k == entity.KindGap && newStatus == entity.StatusAddressed:
		if len(opts.AddressedBy) == 0 && len(opts.AddressedByCommit) == 0 {
			return fmt.Errorf("promoting a gap to %q requires --by <entity-id> or --by-commit <sha> so the gap-resolved-has-resolver rule is satisfied; pass --force to override", entity.StatusAddressed)
		}
	case k == entity.KindADR && newStatus == entity.StatusSuperseded:
		if opts.SupersededBy == "" {
			return fmt.Errorf("promoting an ADR to %q requires --superseded-by <ADR-id> so the adr-supersession-mutual rule is satisfied; pass --force to override", entity.StatusSuperseded)
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
