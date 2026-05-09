package verb

import (
	"context"
	"fmt"
	"strings"

	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/gitops"
	"github.com/23min/ai-workflow-v2/internal/scope"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// AuthorizeMode picks one of the three sub-verbs of `aiwf authorize`.
// Each mode produces exactly one commit; mixing modes is a usage error
// caught by the cmd dispatcher before this package sees the call.
type AuthorizeMode int

// Authorize sub-verbs.
const (
	// AuthorizeOpen opens a fresh scope on the named entity, granting
	// the agent identified by AuthorizeOptions.Agent. Refused when the
	// entity is at a terminal status, unless overridden with Force +
	// non-empty Reason.
	AuthorizeOpen AuthorizeMode = iota
	// AuthorizePause pauses the most-recently-opened active scope on
	// the named entity. Reason is required (non-empty after trim).
	AuthorizePause
	// AuthorizeResume resumes the most-recently-paused scope on the
	// named entity. Reason is required (non-empty after trim).
	AuthorizeResume
)

// AuthorizeOptions configures one invocation of the authorize verb.
//
// Scopes carries every scope ever opened on the target entity, in
// open-order (oldest first), with each scope's current State derived
// from the entity's commit history. The cmd dispatcher loads it via
// loadEntityScopes; this package never reads git directly. For
// AuthorizeOpen the slice is unused (a fresh scope doesn't depend on
// existing ones); for AuthorizePause / AuthorizeResume it is the
// source of truth for the most-recently-opened-active /
// most-recently-paused selection.
type AuthorizeOptions struct {
	Mode   AuthorizeMode
	Agent  string
	Reason string
	Force  bool
	Scopes []*scope.Scope
}

// Authorize runs the `aiwf authorize` verb. Refusal rules per
// docs/pocv3/design/provenance-model.md §"The aiwf authorize verb":
//
//   - Actor must be human/...; only humans authorize.
//   - For AuthorizeOpen, the scope-entity must not be in a terminal
//     status (overridable with Force + non-empty Reason).
//   - For AuthorizePause, an active scope on the entity must exist.
//   - For AuthorizeResume, a paused scope on the entity must exist.
//   - Reason is required for pause/resume (non-empty after trim);
//     optional for AuthorizeOpen unless Force is set.
//
// Each invocation produces exactly one commit. The commit's diff is
// empty; Plan.AllowEmpty makes Apply use `git commit --allow-empty`.
// The agent is recorded in `aiwf-to:` (consistent with the existing
// trailer schema: the scope is the "entity" being acted on, with its
// target state encoded by who can act under it).
func Authorize(ctx context.Context, t *tree.Tree, id, actor string, opts AuthorizeOptions) (*Result, error) {
	_ = ctx
	actor = strings.TrimSpace(actor)
	if !strings.HasPrefix(actor, "human/") {
		return nil, fmt.Errorf("aiwf authorize requires a human/ actor (got %q); only humans authorize", actor)
	}
	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("entity %q not found", id)
	}
	switch opts.Mode {
	case AuthorizeOpen:
		return authorizeOpen(e, actor, opts)
	case AuthorizePause:
		return authorizeTransition(e, actor, opts.Reason, opts.Scopes,
			scope.StateActive, "pause", "paused",
			"no active scope on %s to pause")
	case AuthorizeResume:
		return authorizeTransition(e, actor, opts.Reason, opts.Scopes,
			scope.StatePaused, "resume", "resumed",
			"no paused scope on %s to resume")
	default:
		return nil, fmt.Errorf("aiwf authorize: unknown mode %d", opts.Mode)
	}
}

func authorizeOpen(e *entity.Entity, actor string, opts AuthorizeOptions) (*Result, error) {
	agent := strings.TrimSpace(opts.Agent)
	if agent == "" {
		return nil, fmt.Errorf("aiwf authorize --to <agent>: agent is required")
	}
	if err := gitops.ValidateTrailer(gitops.TrailerTo, agent); err != nil {
		// aiwf-to has no shape rule today (free-string), so this is a
		// no-op now; the explicit call documents intent and future-proofs
		// against tightening the rule.
		return nil, fmt.Errorf("aiwf authorize --to: %w", err)
	}
	if !strings.Contains(agent, "/") {
		return nil, fmt.Errorf("aiwf authorize --to: agent %q must match <role>/<id>", agent)
	}

	terminal := isTerminalStatus(e.Kind, e.Status)
	if terminal && !opts.Force {
		return nil, fmt.Errorf("%s is at terminal status %q; pass --force --reason \"...\" to authorize work on a terminal entity", e.ID, e.Status)
	}
	if opts.Force && strings.TrimSpace(opts.Reason) == "" {
		return nil, fmt.Errorf("aiwf authorize --force requires --reason \"...\" (non-empty after trim)")
	}

	trailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "authorize"},
		// Canonical width per AC-1 in M-081.
		{Key: gitops.TrailerEntity, Value: entity.Canonicalize(e.ID)},
		{Key: gitops.TrailerActor, Value: actor},
		{Key: gitops.TrailerTo, Value: agent},
		{Key: gitops.TrailerScope, Value: "opened"},
	}
	if r := strings.TrimSpace(opts.Reason); r != "" {
		trailers = append(trailers, gitops.Trailer{Key: gitops.TrailerReason, Value: r})
	}
	if opts.Force {
		trailers = append(trailers, gitops.Trailer{Key: gitops.TrailerForce, Value: strings.TrimSpace(opts.Reason)})
	}

	if err := validateAuthorizeTrailers(trailers); err != nil {
		return nil, err
	}
	if err := CheckTrailerCoherence(trailers); err != nil {
		return nil, err
	}

	subject := fmt.Sprintf("aiwf authorize %s --to %s", entity.Canonicalize(e.ID), agent)
	return plan(&Plan{
		Subject:    subject,
		Body:       opts.Reason,
		Trailers:   trailers,
		AllowEmpty: true,
	}), nil
}

// authorizeTransition handles --pause and --resume: both pick the
// most-recently-opened scope in the source state and emit one commit
// recording the transition. Reason is required for both.
func authorizeTransition(
	e *entity.Entity,
	actor, reason string,
	scopes []*scope.Scope,
	source scope.State,
	modeWord, scopeValue, missingFmt string,
) (*Result, error) {
	r := strings.TrimSpace(reason)
	if r == "" {
		return nil, fmt.Errorf("aiwf authorize --%s requires a non-empty reason", modeWord)
	}
	if mostRecentScopeInState(scopes, source) == nil {
		return nil, fmt.Errorf(missingFmt, e.ID)
	}

	trailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "authorize"},
		// Canonical width per AC-1 in M-081.
		{Key: gitops.TrailerEntity, Value: entity.Canonicalize(e.ID)},
		{Key: gitops.TrailerActor, Value: actor},
		{Key: gitops.TrailerScope, Value: scopeValue},
		{Key: gitops.TrailerReason, Value: r},
	}
	if err := validateAuthorizeTrailers(trailers); err != nil {
		return nil, err
	}
	if err := CheckTrailerCoherence(trailers); err != nil {
		return nil, err
	}

	subject := fmt.Sprintf("aiwf authorize %s --%s", entity.Canonicalize(e.ID), modeWord)
	return plan(&Plan{
		Subject:    subject,
		Body:       r,
		Trailers:   trailers,
		AllowEmpty: true,
	}), nil
}

// mostRecentScopeInState returns the most-recently-opened scope whose
// current state matches `state`. Scopes are passed in open-order
// (oldest first); we walk backward so the latest match wins. Returns
// nil when no scope matches — the caller refuses the verb.
func mostRecentScopeInState(scopes []*scope.Scope, state scope.State) *scope.Scope {
	for i := len(scopes) - 1; i >= 0; i-- {
		if scopes[i] != nil && scopes[i].State == state {
			return scopes[i]
		}
	}
	return nil
}

// isTerminalStatus reports whether the (kind, status) pair has no
// outgoing entity-FSM transitions — i.e., is a terminal state. The
// PoC's per-kind FSM is in entity.transitions (a closed set with no
// outgoing edges from `done`/`cancelled`/`rejected`/`wontfix`/etc.).
// AllowedTransitions returns nil for an unknown kind/status pair AND
// for a known terminal state; the latter is intentional — we treat
// "no defined moves out" as terminal. Verb refusal happens before
// this for entities not in the tree.
func isTerminalStatus(k entity.Kind, status string) bool {
	return len(entity.AllowedTransitions(k, status)) == 0
}

// validateAuthorizeTrailers runs gitops.ValidateTrailer on every
// trailer in the set so write-time shape rules (aiwf-actor regex,
// aiwf-scope closed set, aiwf-reason non-empty) fire here rather than
// only inside the standing rule. The first violation is returned.
func validateAuthorizeTrailers(trailers []gitops.Trailer) error {
	for _, tr := range trailers {
		if err := gitops.ValidateTrailer(tr.Key, tr.Value); err != nil {
			return fmt.Errorf("aiwf authorize: %w", err)
		}
	}
	return nil
}
