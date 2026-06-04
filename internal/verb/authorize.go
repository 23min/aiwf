package verb

import (
	"context"
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/branchparse"
	"github.com/23min/aiwf/internal/codes"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
	"github.com/23min/aiwf/internal/tree"
)

// CodeAuthorizeKindNotAllowed is the typed kernel-code descriptor carried
// by [AuthorizeKindError] when aiwf authorize refuses a scope-entity that
// is not an epic or milestone (D-0007). It declares [codes.ClassLegality],
// the marker the closed legality set is enumerated from (D-0011).
// Consumers see its [codes.Code.ID] string via [AuthorizeKindError.Code]
// and in the message text.
var CodeAuthorizeKindNotAllowed = codes.Code{ID: "authorize-kind-not-allowed", Class: codes.ClassLegality}

// AuthorizeKindError reports an aiwf authorize refused because the
// scope-entity is not an epic or milestone (D-0007). It implements
// [entity.Coded], carrying CodeAuthorizeKindNotAllowed. Error preserves
// the established message text (including the code) so message-matching
// consumers keep working while machine consumers use Code.
type AuthorizeKindError struct {
	Kind entity.Kind
}

// Error implements error.
func (e *AuthorizeKindError) Error() string {
	return fmt.Sprintf("aiwf authorize: kind %q is not allowed (%s); only epic and milestone carry autonomous-work scopes", e.Kind, CodeAuthorizeKindNotAllowed.ID)
}

// Code returns CodeAuthorizeKindNotAllowed's ID, satisfying [entity.Coded].
func (e *AuthorizeKindError) Code() string { return CodeAuthorizeKindNotAllowed.ID }

// CodePreflightBranchContextRequired is the typed kernel-code descriptor
// carried by [PreflightBranchContextRequiredError] when aiwf authorize
// refuses opening a scope on an ai/* agent because neither --branch was
// passed nor the current checkout matches a ritual shape (M-0103 /
// ADR-0010). Class is [codes.ClassLegality].
var CodePreflightBranchContextRequired = codes.Code{ID: "branch-context-required", Class: codes.ClassLegality}

// CodePreflightBranchNotFound is the typed kernel-code descriptor
// carried by [PreflightBranchNotFoundError] when aiwf authorize refuses
// opening a scope on an ai/* agent because --branch <name> was passed
// but no local branch by that name exists (M-0103 / ADR-0010). Class is
// [codes.ClassLegality].
var CodePreflightBranchNotFound = codes.Code{ID: "branch-not-found", Class: codes.ClassLegality}

// PreflightBranchContextRequiredError reports an aiwf authorize refused
// because the AI-target preflight (M-0103) found no ritual branch
// context in play — neither was --branch supplied, nor does the current
// checkout match a ritual shape recognized by internal/branchparse/.
// Implements [entity.Coded] via Code; carries
// CodePreflightBranchContextRequired.
type PreflightBranchContextRequiredError struct {
	// Agent is the --to ai/<id> value that triggered the preflight.
	Agent string
	// CurrentBranch is whatever the caller reported as the current
	// checkout (empty when HEAD is detached or git failed); included
	// in the message so the operator can see what was checked.
	CurrentBranch string
}

// Error implements error.
func (e *PreflightBranchContextRequiredError) Error() string {
	return fmt.Sprintf(
		"aiwf authorize: opening a scope on %q requires a ritual branch context (%s); current checkout %q does not match a ritual shape. Run `aiwfx-start-epic` / `aiwfx-start-milestone` to land on a recognized ritual branch (epic/E-NNNN-<slug> / milestone/M-NNNN-<slug> / patch/g-NNNN-<slug>), or pass `--branch <name>` naming an existing branch. To override this preflight as a sovereign act, use `--force --reason \"<one-sentence justification>\"`.",
		e.Agent, CodePreflightBranchContextRequired.ID, e.CurrentBranch,
	)
}

// Code returns CodePreflightBranchContextRequired's ID, satisfying [entity.Coded].
func (e *PreflightBranchContextRequiredError) Code() string {
	return CodePreflightBranchContextRequired.ID
}

// PreflightBranchNotFoundError reports an aiwf authorize refused
// because the AI-target preflight (M-0103) was given --branch <name>
// but the named branch does not exist locally. Implements [entity.Coded]
// via Code; carries CodePreflightBranchNotFound.
type PreflightBranchNotFoundError struct {
	// Branch is the --branch value that did not resolve under refs/heads/.
	Branch string
}

// Error implements error.
func (e *PreflightBranchNotFoundError) Error() string {
	return fmt.Sprintf(
		"aiwf authorize: --branch %q refers to a non-existent local branch (%s). Pass a name that resolves under refs/heads/, or omit --branch to use the current checkout (which must already be on a ritual-shape branch). From `main` or a ritual-shape current branch (epic/milestone/patch), naming a ritual-shape future --branch is accepted (the step-7 pattern of aiwfx-start-epic per M-0104/AC-4 — or step-4 of aiwfx-start-milestone per M-0105/AC-6). To override this preflight as a sovereign act, use `--force --reason \"...\"`.",
		e.Branch, CodePreflightBranchNotFound.ID,
	)
}

// Code returns CodePreflightBranchNotFound's ID, satisfying [entity.Coded].
func (e *PreflightBranchNotFoundError) Code() string {
	return CodePreflightBranchNotFound.ID
}

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
	Mode AuthorizeMode
	// Agent is the role/<id> the scope authorizes (e.g. "ai/claude").
	Agent string
	// Reason is the rationale; required for pause/resume, optional for
	// open (required when Force is set).
	Reason string
	// Branch (M-0102 / ADR-0010) is the ritual branch a scope is bound
	// to. Optional: when empty *and* the target agent is non-ai/*, no
	// aiwf-branch: trailer is emitted (backward-compatible). For ai/*
	// targets, the M-0103 preflight resolves an empty Branch from
	// CurrentBranch when the current checkout is a ritual shape, and
	// refuses with PreflightBranchContextRequiredError when it is not.
	// Validated against the git-ref shape rule in gitops.ValidateTrailer
	// at trailer-assembly time.
	Branch string
	// CurrentBranch (M-0103) is the short name of the consumer's
	// currently-checked-out branch at verb-call time, as the CLI layer
	// resolves it via `git symbolic-ref --short HEAD`. Empty when HEAD
	// is detached, when the CLI's git invocation fails, or when this
	// package is exercised by a verb-level test that does not set it.
	// Read by the M-0103 preflight only when the target agent is ai/*
	// and --force is not set; in every other code path the field is
	// ignored, so leaving it empty in unrelated tests is harmless.
	CurrentBranch string
	// BranchExists (M-0103) reports whether Branch refers to a local
	// branch that resolves under refs/heads/<Branch>, as the CLI layer
	// determines via `git show-ref --verify`. The CLI sets this iff
	// Branch is non-empty (when Branch is empty there is nothing to
	// check). Read by the M-0103 preflight only when the target agent
	// is ai/* and --force is not set; in every other code path the
	// field is ignored.
	BranchExists bool
	// TrunkShort (M-0161/AC-1, G-0200) is the consumer's configured
	// trunk short-name as derived from `aiwf.yaml.allocate.trunk` via
	// `Config.TrunkBranchShortName()`. Used by the AI-target
	// preflight's "trunk + ritual --branch" carve-out so the predicate
	// honors the operator's configured trunk rather than the literal
	// `"main"`. Populated by the CLI layer via
	// `cliutil.ConfiguredTrunkBranchShortName`; empty (e.g., from a
	// verb-level test that doesn't set it) is treated as "no
	// resolvable trunk; do not match" — the carve-out's left arm
	// fails and preflight falls through to the implicit-ritual-
	// current path.
	TrunkShort string
	Force      bool
	Scopes     []*scope.Scope
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
	// Kind allowlist: per D-0007 (authorize-scope) only scope-entities
	// (Epic + Milestone) carry autonomous-work scopes. Gap, Decision,
	// Contract, and ADR are kernel ledgers — they have no notion of
	// "open scope for an agent." Spec cell R-AUDIT-0122 / R-FP-0133 /
	// D-0007 captures the rule; this guard is its verb-time chokepoint.
	if e.Kind != entity.KindEpic && e.Kind != entity.KindMilestone {
		return nil, &AuthorizeKindError{Kind: e.Kind}
	}

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

	// Refusal-ordering: terminal-status (line above) → force-requires-
	// reason → M-0103 preflight (below). The preflight is the LAST
	// refusal layer so an operator hitting both "entity is terminal"
	// AND "no ritual branch" gets the terminal error first (closer to
	// root cause). The AC-6 test pins the *error-message-identity*
	// invariant — operators with (Force=true, Reason="") see the
	// --reason error rather than branch-context-required — which the
	// preflight's `!opts.Force` short-circuit (below) guarantees
	// independent of literal source order. A reorder that preserves
	// `!opts.Force` therefore does NOT regress the user-observable
	// contract. The order is preserved here for *readability* (the
	// gates are ordered cheapest-cause-first); refactor freely as
	// long as the `!opts.Force` clause stays on the preflight.
	//
	// M-0103: AI-target preflight. Per ADR-0010 the branch model treats
	// AI multi-commit work as bound to a ritual branch context; opening
	// a scope on an ai/* agent without that context defeats the
	// chokepoint that M-0106 (post-hoc finding) and the rituals
	// (M-0104 / M-0105) rely on. Two signals satisfy the gate:
	//
	//   - opts.Branch is set and opts.BranchExists is true (the caller
	//     explicitly named an existing local branch), OR
	//   - opts.Branch is empty and opts.CurrentBranch matches a ritual
	//     shape per internal/branchparse/ (the current checkout is
	//     already a ritual branch).
	//
	// When the implicit-current-branch signal succeeds, opts.Branch is
	// promoted to the current branch's name so the trailer-emission
	// code below stamps aiwf-branch: with that value — making the
	// implicit binding explicit in the commit record (per AC-3).
	//
	// Force + Reason path bypasses the preflight as a sovereign act.
	// The trailer-coherence rules already forbid Force for non-human
	// actors (see internal/verb/coherence.go), so the override is
	// structurally human-sovereign — no new gate needed here.
	if strings.HasPrefix(agent, "ai/") && !opts.Force {
		branchExplicit := strings.TrimSpace(opts.Branch)
		if branchExplicit != "" {
			if !opts.BranchExists {
				// Future-branch carve-out (M-0104/AC-4 + M-0105/AC-6):
				// an explicit --branch naming a ritual-shape ref is
				// accepted even when the branch does not yet exist,
				// provided the operator's current checkout is a
				// valid "place from which to cut the future ritual
				// branch". Two such places are recognized:
				//
				//   - main (M-0104/AC-4 — the step-7 pattern of
				//     aiwfx-start-epic; the epic branch is cut at
				//     step 8).
				//   - any ritual shape per branchparse (M-0105/AC-6
				//     — the step-4 pattern of aiwfx-start-milestone,
				//     where the operator is on the parent epic
				//     branch and --branch names the future milestone
				//     branch; the milestone branch is cut at step 5).
				//
				// The carve-out is intentionally tight:
				//   - trunk-or-ritual-current only (a plain feature
				//     branch is not a "place from which to cut a
				//     ritual"; the implicit-ritual-current path or
				//     --force --reason cover legitimate exceptions).
				//     M-0161/AC-1 (G-0200) generalized "main" to
				//     opts.TrunkShort (sourced from
				//     aiwf.yaml.allocate.trunk via
				//     Config.TrunkBranchShortName()), so
				//     `master`/`dev`/operator-chosen trunks all
				//     work uniformly. Empty TrunkShort (e.g., a
				//     verb-level test that doesn't populate it, or
				//     a malformed config that produces no
				//     parseable short-name) is treated as "no
				//     trunk match"; the carve-out's left arm fails
				//     and preflight falls through.
				//   - ritual-shape --branch only (otherwise the gate
				//     becomes a no-op — any string under --branch
				//     would authorize from any qualifying current).
				//
				// The carve-out does NOT enforce hierarchical
				// consistency between CurrentBranch and the --branch
				// shape (e.g., "current=epic/X-7 implies --branch
				// must be milestone/<child of X-7>"). YAGNI: the
				// looser check covers every legitimate ritual
				// invocation and refuses the loudest mistakes; a
				// hierarchical check would be more code for a
				// narrower window. Cross-rung mismatches (different
				// epic, up-the-tree, epic→patch skipping milestone)
				// syntactically accept; the trailer records the
				// operator's stated intent. Parked as G-0201
				// pending evidence the looseness becomes an incident
				// class.
				currentIsRitualContext := (opts.TrunkShort != "" && opts.CurrentBranch == opts.TrunkShort) ||
					branchparse.ParseEntityFromBranch(opts.CurrentBranch) != ""
				futureBindingAccepted := currentIsRitualContext &&
					branchparse.ParseEntityFromBranch(branchExplicit) != ""
				if !futureBindingAccepted {
					return nil, &PreflightBranchNotFoundError{Branch: branchExplicit}
				}
			}
		} else {
			if branchparse.ParseEntityFromBranch(opts.CurrentBranch) == "" {
				return nil, &PreflightBranchContextRequiredError{
					Agent:         agent,
					CurrentBranch: opts.CurrentBranch,
				}
			}
			// Safe: opts is passed by value into authorizeOpen (signature
			// at line 226), so this write is local to this call. If opts
			// is ever migrated to a pointer for unrelated reasons, the
			// mutation would leak back to the CLI caller — replace with
			// a local `effectiveBranch` and use that downstream.
			opts.Branch = opts.CurrentBranch
		}
	}

	trailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "authorize"},
		// Canonical width per AC-1 in M-081.
		{Key: gitops.TrailerEntity, Value: entity.Canonicalize(e.ID)},
		{Key: gitops.TrailerActor, Value: actor},
		{Key: gitops.TrailerTo, Value: agent},
		{Key: gitops.TrailerScope, Value: "opened"},
	}
	// M-0102 / AC-3: aiwf-branch trailer is emitted iff --branch was
	// passed; empty Branch keeps the trailer absent (backward-compatible).
	// validateAuthorizeTrailers below catches shape violations via the
	// AC-2 git-ref rule, so a malformed Branch fails the verb before any
	// commit lands.
	if b := strings.TrimSpace(opts.Branch); b != "" {
		trailers = append(trailers, gitops.Trailer{Key: gitops.TrailerBranch, Value: b})
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
