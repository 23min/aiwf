package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/scope"
)

// ScopeView is one scope's projection on `aiwf show`. It captures
// the authorization grant's metadata (SHA, agent, principal) and its
// current FSM state, plus the open/end dates and the count of
// transitions the scope has gone through.
//
// Auth SHA is the full git SHA of the authorize-opened commit;
// callers that want a short form truncate. Entity is the scope-
// entity id at the time the scope was opened (rename-chain
// resolution lives in the verb gate, not here — show is descriptive,
// not gating).
type ScopeView struct {
	AuthSHA    string `json:"auth_sha"`
	Entity     string `json:"entity"`
	Agent      string `json:"agent"`
	Principal  string `json:"principal"`
	State      string `json:"state"`
	Opened     string `json:"opened,omitempty"`
	EndedAt    string `json:"ended_at,omitempty"`
	EventCount int    `json:"event_count"`
}

// loadEntityScopeViews returns every scope that ever applied to id —
// scopes opened ON id (directly), plus scopes from elsewhere that
// authorized work touching id (via `aiwf-authorized-by:`).
//
// Implementation: one global `git log` pass over authorize-opened
// commits to build authSHA → scope-entity. Then we walk id's
// history (readHistory) and collect every distinct auth-SHA the
// entity references (its own opener SHAs plus authorized-by
// references). For each scope-entity touched, loadEntityScopes
// materializes the FSM; we then filter to the interested SHAs and
// convert to ScopeView.
//
// Empty / pre-aiwf repos return (nil, nil).
func loadEntityScopeViews(ctx context.Context, root, id string) ([]ScopeView, error) {
	if !hasCommits(ctx, root) {
		return nil, nil
	}
	events, err := readHistory(ctx, root, id)
	if err != nil {
		return nil, err
	}
	scopeEntityByAuthSHA, err := readAllAuthorizeOpeners(ctx, root)
	if err != nil {
		return nil, err
	}

	interested := map[string]struct{}{}
	for i := range events {
		if events[i].AuthorizedBy != "" {
			interested[events[i].AuthorizedBy] = struct{}{}
		}
	}
	for sha, ent := range scopeEntityByAuthSHA {
		if ent == id {
			interested[sha] = struct{}{}
		}
	}
	if len(interested) == 0 {
		return nil, nil
	}

	scopeEntitiesNeeded := map[string]struct{}{}
	for sha := range interested {
		if ent, ok := scopeEntityByAuthSHA[sha]; ok {
			scopeEntitiesNeeded[ent] = struct{}{}
		}
	}

	var allScopes []*scope.Scope
	for ent := range scopeEntitiesNeeded {
		scopes, err := loadEntityScopes(ctx, root, ent)
		if err != nil {
			return nil, err
		}
		allScopes = append(allScopes, scopes...)
	}

	dateCache := map[string]string{}
	var views []ScopeView
	for _, s := range allScopes {
		if _, ok := interested[s.AuthSHA]; !ok {
			continue
		}
		opened := lookupCommitDateCached(ctx, root, s.AuthSHA, dateCache)
		var ended string
		if s.State == scope.StateEnded {
			if last := lastEventSHA(s, scope.StateEnded); last != "" {
				ended = lookupCommitDateCached(ctx, root, last, dateCache)
			}
		}
		views = append(views, ScopeView{
			AuthSHA:    s.AuthSHA,
			Entity:     s.Entity,
			Agent:      s.Agent,
			Principal:  s.Principal,
			State:      string(s.State),
			Opened:     opened,
			EndedAt:    ended,
			EventCount: len(s.Events),
		})
	}
	sort.Slice(views, func(i, j int) bool {
		return views[i].Opened < views[j].Opened
	})
	return views, nil
}

// readAllAuthorizeOpeners returns a map from each authorize-opener
// commit's full SHA to its scope-entity id. Used by show to know
// which entity a scope was opened against without per-row lookups.
func readAllAuthorizeOpeners(ctx context.Context, root string) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, "git", "log",
		"-E",
		"--grep", "^aiwf-verb: authorize$",
		"--grep", "^aiwf-scope: opened$",
		"--all-match",
		"--pretty=tformat:%H\x1f%(trailers:key=aiwf-entity,valueonly=true,unfold=true)\x1e")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git log: %w\n%s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git log: %w", err)
	}
	result := map[string]string{}
	for _, rec := range strings.Split(string(out), "\x1e") {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, "\x1f", 2)
		if len(parts) < 2 {
			continue
		}
		sha := strings.TrimSpace(parts[0])
		ent := strings.TrimSpace(parts[1])
		if sha == "" || ent == "" {
			continue
		}
		// Canonicalize the trailer-stored entity id so callers comparing
		// against tree-loaded ids never have to disambiguate widths.
		// Per AC-2 in M-081: the read side is the chokepoint for
		// width tolerance.
		result[sha] = entity.Canonicalize(ent)
	}
	return result, nil
}

// lookupCommitDateCached returns the ISO-8601 author date of the
// commit at sha, caching results so we never hit `git show` twice
// for the same SHA in one show call. Errors fall back to an empty
// string (the caller renders dates as omitempty in JSON).
func lookupCommitDateCached(ctx context.Context, root, sha string, cache map[string]string) string {
	if d, ok := cache[sha]; ok {
		return d
	}
	cmd := exec.CommandContext(ctx, "git", "show", "-s", "--format=%aI", sha)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		cache[sha] = ""
		return ""
	}
	d := strings.TrimSpace(string(out))
	cache[sha] = d
	return d
}

// lastEventSHA returns the SHA of the latest event in s whose state
// equals match, or "" when none. Used by ScopeView assembly to look
// up the ending commit's date (when the scope is ended).
func lastEventSHA(s *scope.Scope, match scope.State) string {
	for i := len(s.Events) - 1; i >= 0; i-- {
		if s.Events[i].State == match {
			return s.Events[i].SHA
		}
	}
	return ""
}
