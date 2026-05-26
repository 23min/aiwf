// Package verb — I2.5 allow-rule composition.
//
// Allow gates an agent's verb invocation against the union of
// currently-active scopes attached to the agent. Per
// docs/pocv3/design/provenance-model.md §"Composition with entity
// FSMs":
//
//	allow(verb v on entity e by actor a) =
//	    legalEntityTransition(e, v.target_state)         // existing entity FSM
//	    AND scopeAllows(a, v, e)                          // new scope check
//
// For human/... actors with no --principal flag, scopeAllows is
// skipped entirely — humans need no delegation. For ai/... or other
// non-human actors, at least one active scope must answer "yes" to
// scopeAllows; if none does, the verb refuses with provenance-no-
// active-scope and no commit lands.
//
// The function is intentionally pure: tree (forward-reachability)
// and pre-loaded scopes are passed in. The cmd dispatcher does the
// git I/O (loadActiveScopesForActor) and tree.Load.
package verb

import (
	"errors"
	"strings"

	"github.com/23min/aiwf/internal/scope"
	"github.com/23min/aiwf/internal/tree"
)

// VerbKind discriminates the act being gated. Different kinds use
// different reachability rules per scopeAllows: a creation act
// checks the new entity's outbound references against the scope-
// entity; a move act requires both endpoints to reach scope; every
// other act checks only the target. Step 6's first cut covers the
// simple act (target-only); creation and move become relevant when
// the cmd-level wiring lands them.
type VerbKind int

// VerbKind values.
const (
	// VerbAct is the default: the act has a single target entity
	// (promote, cancel, rename frontmatter changes, etc.). The
	// target must reach the scope-entity.
	VerbAct VerbKind = iota
	// VerbCreate is a creation act: a new entity is added with
	// outbound references. At least one outbound reference (or
	// the parent, for milestones) must reach the scope-entity.
	VerbCreate
	// VerbMove is a relocation act: both source and destination
	// endpoints must reach the scope-entity.
	VerbMove
)

// AllowInput bundles every input the allow-rule consumes. Lets the
// cmd dispatcher build a single struct rather than passing seven
// arguments through a chain of helpers.
//
// Kind picks the reachability variant. TargetID is the entity the
// verb mutates (or the destination, for VerbMove). For VerbCreate,
// CreationRefs lists the outbound reference targets the new entity
// would carry; for VerbMove, MoveSource is the original location.
//
// Actor is the operator (whoever ran the verb). Principal is the
// human on whose behalf the operator is acting (always human/...);
// empty when the actor is acting directly.
//
// Scopes is the union of active scopes attached to Actor — the cmd
// dispatcher loads it via loadActiveScopesForActor (filters
// authorize commits by aiwf-to: <Actor>).
//
// Tree is the in-memory entity tree used for forward reachability.
type AllowInput struct {
	Kind         VerbKind
	TargetID     string
	CreationRefs []string
	MoveSource   string
	Actor        string
	Principal    string
	Scopes       []*scope.Scope
	Tree         *tree.Tree
}

// AllowResult carries the verdict. When Allowed is true and Scope is
// non-nil, the cmd dispatcher decorates the verb's plan with
// aiwf-on-behalf-of: <Scope.Principal> and aiwf-authorized-by:
// <Scope.AuthSHA>. When Allowed is true and Scope is nil, the actor
// is human/... and no scope decoration is needed (direct human act).
//
// Reason carries a one-line explanation when Allowed is false; the
// cmd dispatcher surfaces it as the user-facing error.
type AllowResult struct {
	Allowed bool
	Scope   *scope.Scope
	Reason  string
	// Err is the denial error when Allowed is false (nil when allowed).
	// For the scope-reachability and no-active-scope denials it is a
	// Coded error (entity.Code-extractable: *ScopeOutOfReachError /
	// *NoActiveScopeError); the pre-scope usage denials carry a plain
	// error. The cmd dispatcher returns it (wrapped with %w) so a Coded
	// refusal surfaces as error.code in the --format=json envelope.
	// Invariant: every Allowed==false return sets Err.
	Err error
}

// Allow runs the I2.5 allow-rule over the given inputs. Pure: no
// I/O, no git access. Returns a verdict the cmd dispatcher acts on.
//
// Decision tree:
//
//   - Empty actor → denied (kernel never lets an unidentified
//     operator commit; the cmd dispatcher should have refused
//     earlier, but defensive). Reason: "actor is required".
//
//   - human/... actor:
//     -> Principal must be empty (humans act directly; principal
//     forbidden per the trailer-coherence rules). When non-empty,
//     denied with "principal forbidden for human actor".
//     -> Otherwise: Allowed = true, Scope = nil.
//
//   - non-human actor (ai/... / bot/...):
//     -> Principal must be set (every agent needs a human accountor;
//     enforced again by trailer coherence). When empty, denied with
//     "principal required for non-human actor".
//     -> Iterate Scopes (the union of active scopes attached to this
//     actor). Pick the most-recently-opened that satisfies
//     scopeAllows for the given (Kind, TargetID). On match: Allowed
//     = true, Scope = the matching scope. On no match the denial is
//     split (D-0014): if at least one active scope existed but none
//     reached the target, Err is a *ScopeOutOfReachError
//     (provenance-authorization-out-of-scope); if there was no active
//     scope at all, Err is a *NoActiveScopeError
//     (provenance-no-active-scope).
//
// Per the design's "if multiple match, pick the most-recently-opened
// deterministically" rule, scopes are walked in reverse insertion
// order so the latest open wins.
func Allow(in AllowInput) AllowResult {
	actor := strings.TrimSpace(in.Actor)
	if actor == "" {
		return denied(errors.New("actor is required"))
	}
	if strings.HasPrefix(actor, "human/") {
		if in.Principal != "" {
			return denied(errors.New("principal forbidden for human/ actor (humans act directly)"))
		}
		return AllowResult{Allowed: true}
	}
	if in.Principal == "" {
		return denied(errors.New("principal required for non-human actor (set --principal human/<id>)"))
	}
	// Distinguish the two scope-denial cases (D-0014): an active scope
	// exists but none reach the target (out-of-reach) vs. no active scope
	// at all. Each carries its own structured code.
	hasActive := false
	for i := len(in.Scopes) - 1; i >= 0; i-- {
		s := in.Scopes[i]
		if s == nil || s.State != scope.StateActive {
			continue
		}
		hasActive = true
		if scopeAllowsAct(s, in) {
			return AllowResult{Allowed: true, Scope: s}
		}
	}
	if hasActive {
		return denied(&ScopeOutOfReachError{Actor: actor, Target: in.TargetID, Refs: in.CreationRefs})
	}
	return denied(&NoActiveScopeError{Actor: actor})
}

// denied builds a refusal AllowResult from a denial error, mirroring the
// error text into Reason for callers that surface the reason string
// directly. Every Allowed==false path routes through here so the Err
// invariant (AllowResult.Err non-nil on denial) holds by construction.
func denied(err error) AllowResult {
	return AllowResult{Reason: err.Error(), Err: err}
}

// scopeAllowsAct mirrors the scopeAllows function in
// docs/pocv3/design/provenance-model.md §"Scope check". The scope
// must be active (already filtered by the caller); the verb's
// reachability rule depends on Kind:
//
//   - VerbAct: TargetID must reach Scope.Entity.
//   - VerbCreate: any of CreationRefs must reach Scope.Entity.
//   - VerbMove: both MoveSource and TargetID must reach Scope.Entity.
//
// Reachability is D-0006's three-edge scope tree — parent-forward,
// composite-id containment, discovered_in-reverse — and explicitly NOT
// the full reference graph: governance edges (depends_on, addressed_by,
// relates_to, supersedes, superseded_by, linked_adrs) do not punch
// through a scope boundary. tree.ReachesScope encapsulates the walk.
func scopeAllowsAct(s *scope.Scope, in AllowInput) bool {
	t := in.Tree
	if t == nil {
		return false
	}
	switch in.Kind {
	case VerbCreate:
		return t.ReachesScopeAny(in.CreationRefs, s.Entity)
	case VerbMove:
		return t.ReachesScope(in.MoveSource, s.Entity) && t.ReachesScope(in.TargetID, s.Entity)
	default: // VerbAct
		return t.ReachesScope(in.TargetID, s.Entity)
	}
}
