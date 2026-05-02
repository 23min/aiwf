package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
	"github.com/23min/ai-workflow-v2/tools/internal/scope"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
	"github.com/23min/ai-workflow-v2/tools/internal/verb"
)

// provenanceContext carries the inputs the cmd dispatcher feeds into
// the I2.5 allow-rule and trailer-decoration step. Built once per
// verb invocation.
//
// Actor is the operator (the value `aiwf-actor:` will carry).
// Principal is the human who authorized the act, set via the
// `--principal` flag. For human/... actors Principal must be empty
// (the human is acting directly); for non-human actors Principal is
// required and gates the agent through Allow.
//
// VerbKind discriminates the act for reachability (see verb.VerbKind).
// TargetID, CreationRefs, MoveSource feed the allow-rule's
// reachability check.
//
// IsTerminalPromote is true when the verb is a `promote` whose
// target state is terminal for the entity's kind. Triggers the
// scope-end side effect — writing aiwf-scope-ends for every active
// scope on the target entity.
type provenanceContext struct {
	Actor             string
	Principal         string
	VerbKind          verb.VerbKind
	TargetID          string
	CreationRefs      []string
	MoveSource        string
	IsTerminalPromote bool
}

// gateAndDecorate runs the I2.5 allow-rule against the verb's plan
// (already produced by the verb function) and decorates Plan.Trailers
// with the provenance metadata: aiwf-principal (when actor is non-
// human), aiwf-on-behalf-of and aiwf-authorized-by (when an active
// scope authorized the act), aiwf-scope-ends (one trailer per active
// scope on the entity, when the verb is a terminal promote).
//
// Returns a Go error when the allow-rule denies the act; the cmd
// dispatcher surfaces it as a refusal. Returns nil and decorates the
// plan when allowed.
//
// Pre-verb refusals (principal missing for non-human actor; principal
// supplied for human actor) fire here too. The verb has already run
// at this point; if Allow refuses, we abandon the plan without
// applying it.
func gateAndDecorate(ctx context.Context, root string, t *tree.Tree, plan *verb.Plan, pctx provenanceContext) error {
	if plan == nil {
		return nil
	}
	actor := strings.TrimSpace(pctx.Actor)
	principal := strings.TrimSpace(pctx.Principal)
	actorIsHuman := strings.HasPrefix(actor, "human/")

	// Load the actor's active scopes (only when the actor is non-
	// human — humans bypass scope checks).
	var scopes []*scope.Scope
	if !actorIsHuman {
		s, err := loadActiveScopesForActor(ctx, root, actor)
		if err != nil {
			return fmt.Errorf("loading scopes for actor %q: %w", actor, err)
		}
		scopes = s
	}

	allow := verb.Allow(verb.AllowInput{
		Kind:         pctx.VerbKind,
		TargetID:     pctx.TargetID,
		CreationRefs: pctx.CreationRefs,
		MoveSource:   pctx.MoveSource,
		Actor:        actor,
		Principal:    principal,
		Scopes:       scopes,
		Tree:         t,
	})
	if !allow.Allowed {
		return fmt.Errorf("provenance refused: %s", allow.Reason)
	}

	// Decorate trailers. Order is irrelevant — gitops.SortedTrailers
	// re-sorts on emit — but we append in the order the design doc
	// reads: principal, on-behalf-of, authorized-by, scope-ends.
	if !actorIsHuman {
		plan.Trailers = append(plan.Trailers, gitops.Trailer{
			Key:   gitops.TrailerPrincipal,
			Value: principal,
		})
	}
	if allow.Scope != nil {
		plan.Trailers = append(plan.Trailers,
			gitops.Trailer{Key: gitops.TrailerOnBehalfOf, Value: allow.Scope.Principal},
			gitops.Trailer{Key: gitops.TrailerAuthorizedBy, Value: allow.Scope.AuthSHA},
		)
	}
	if pctx.IsTerminalPromote && pctx.TargetID != "" {
		ends, err := loadActiveScopeAuthSHAsForEntity(ctx, root, pctx.TargetID)
		if err != nil {
			return fmt.Errorf("computing scope-ends for %s: %w", pctx.TargetID, err)
		}
		for _, sha := range ends {
			plan.Trailers = append(plan.Trailers, gitops.Trailer{
				Key:   gitops.TrailerScopeEnds,
				Value: sha,
			})
		}
	}
	return nil
}

// loadActiveScopesForActor returns every scope currently active and
// attached to the given actor, in open-order (oldest first). Walks
// `git log` once filtering for `aiwf-to: <actor>` on commits whose
// `aiwf-verb:` is `authorize` and `aiwf-scope:` is `opened` (the
// opener commits), then augments each scope by reading
// pause/resume/scope-ends events from the entity's history (via
// loadEntityScopes, which the cmd dispatcher already uses for the
// authorize verb).
//
// Returns only scopes whose State is StateActive — paused or ended
// scopes do not authorize work. The caller can still inspect Scope's
// Entity / Principal for trailer decoration.
func loadActiveScopesForActor(ctx context.Context, root, actor string) ([]*scope.Scope, error) {
	if !hasCommits(ctx, root) {
		return nil, nil
	}
	openers, err := readActorOpenerEntities(ctx, root, actor)
	if err != nil {
		return nil, err
	}
	if len(openers) == 0 {
		return nil, nil
	}
	seen := map[string]bool{}
	var result []*scope.Scope
	for _, entityID := range openers {
		if seen[entityID] {
			continue
		}
		seen[entityID] = true
		scopes, err := loadEntityScopes(ctx, root, entityID)
		if err != nil {
			return nil, err
		}
		for _, s := range scopes {
			if s.State == scope.StateActive && s.Agent == actor {
				result = append(result, s)
			}
		}
	}
	return result, nil
}

// readActorOpenerEntities returns the set of scope-entity ids
// referenced by `aiwf-verb: authorize / aiwf-scope: opened` commits
// whose `aiwf-to:` matches the actor. Used by loadActiveScopesForActor
// to know which entities to materialize scopes for. Order matches
// `git log` (newest-first by default; we reverse so the caller sees
// chronological order — though the per-entity walk is order-
// independent).
func readActorOpenerEntities(ctx context.Context, root, actor string) ([]string, error) {
	const sep = "\x1f"
	const recSep = "\x1e\n"
	args := []string{
		"log",
		"--reverse",
		"-E",
		"--grep", "^aiwf-verb: authorize$",
		"--grep", "^aiwf-to: " + regexp.QuoteMeta(actor) + "$",
		"--all-match",
		"--pretty=tformat:%H" + sep +
			"%(trailers:key=aiwf-entity,valueonly=true,unfold=true)" + sep +
			"%(trailers:key=aiwf-scope,valueonly=true,unfold=true)" + sep +
			"%(trailers:key=aiwf-to,valueonly=true,unfold=true)\x1e",
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git log: %w\n%s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git log: %w", err)
	}
	var entities []string
	for _, rec := range strings.Split(string(out), recSep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, sep, 4)
		if len(parts) < 4 {
			continue
		}
		entityID := strings.TrimSpace(parts[1])
		scopeKind := strings.TrimSpace(parts[2])
		toValue := strings.TrimSpace(parts[3])
		if scopeKind != "opened" || toValue != actor || entityID == "" {
			continue
		}
		entities = append(entities, entityID)
	}
	return entities, nil
}

// loadActiveScopeAuthSHAsForEntity returns the auth-SHAs of every
// active scope on entityID, in open-order. Used by the terminal-
// promote scope-end side effect — the verb's commit must carry one
// `aiwf-scope-ends: <auth-sha>` per matched scope, ending each scope
// atomically with the entity's transition to a terminal state.
func loadActiveScopeAuthSHAsForEntity(ctx context.Context, root, entityID string) ([]string, error) {
	scopes, err := loadEntityScopes(ctx, root, entityID)
	if err != nil {
		return nil, err
	}
	var shas []string
	for _, s := range scopes {
		if s.State == scope.StateActive {
			shas = append(shas, s.AuthSHA)
		}
	}
	return shas, nil
}

// isTerminalPromote reports whether `(kind, newStatus)` is a state
// transition into a terminal status — the trigger for the scope-end
// side effect. Mirrors entity.AllowedTransitions semantics: a status
// with no outgoing edges is terminal.
//
// Returns false for unknown kinds or unknown statuses (cautious; the
// trigger should fire on real terminal moves only).
func isTerminalPromote(k entity.Kind, newStatus string) bool {
	allowed := entity.AllowedTransitions(k, newStatus)
	if allowed != nil {
		return len(allowed) == 0
	}
	// Unknown status: be conservative.
	return false
}

// decorateAndFinish wraps the verb's post-execution path: when the
// verb produced a Plan, it runs gateAndDecorate (which enforces the
// I2.5 allow-rule and stamps provenance trailers), then hands off to
// finishVerb to apply the plan and report the outcome.
//
// On allow-rule denial, the plan is abandoned (verb output is
// validate-then-write, so no disk state has been mutated yet) and
// the dispatcher exits with the findings code so the user sees the
// refusal as a clean error.
func decorateAndFinish(
	ctx context.Context,
	root, label string,
	t *tree.Tree,
	result *verb.Result,
	vErr error,
	pctx provenanceContext,
) int {
	if vErr != nil || result == nil || result.Plan == nil {
		return finishVerb(ctx, root, label, result, vErr)
	}
	if err := gateAndDecorate(ctx, root, t, result.Plan, pctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", label, err)
		return exitFindings
	}
	return finishVerb(ctx, root, label, result, nil)
}
