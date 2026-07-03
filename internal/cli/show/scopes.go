package show

import (
	"context"
	"os/exec"
	"sort"
	"strings"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/history"
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

// LoadEntityScopeViews returns every scope that ever applied to id —
// scopes opened ON id (directly), plus scopes from elsewhere that
// authorized work touching id (via `aiwf-authorized-by:`).
//
// Source (b) — scopes opened directly on id — comes from id's own history
// via cliutil.LoadEntityScopes, which is width-tolerant: this is both the
// single source of truth for direct scopes and the fix for the narrow-id
// omission (a raw-id `ent == id` compare against a canonicalized map value
// silently dropped `aiwf show E-14`'s table; M-0223).
//
// Source (a) — scopes opened elsewhere that authorized work on id — is the
// only reason to run the repo-wide authorize-opener grep
// (cliutil.AuthorizeOpeners), and only when id was actually worked under a
// foreign scope (history.HasAuthorizedBy). An active direct-scope opener,
// which has no `aiwf-authorized-by`, resolves entirely from source (b)
// without the grep (E-0054 read-verb guard).
//
// Empty / pre-aiwf repos return (nil, nil).
func LoadEntityScopeViews(ctx context.Context, root, id string) ([]ScopeView, error) {
	if !cliutil.HasCommits(ctx, root) {
		return nil, nil
	}
	events, err := history.ReadHistory(ctx, root, id)
	if err != nil {
		return nil, err //coverage:ignore git-read failure unreachable after HasCommits guards a valid repo
	}
	// Source (b): scopes opened directly on id, derived from id's own
	// history (width-tolerant) — but only walk when the already-loaded
	// events show id actually has an own authorize-opener. A scopeless
	// entity skips this walk entirely (E-0054 / M-0223).
	var ownScopes []*scope.Scope
	if history.HasOwnScope(events) {
		ownScopes, err = cliutil.LoadEntityScopes(ctx, root, id)
		if err != nil {
			return nil, err //coverage:ignore git-read failure unreachable after HasCommits guards a valid repo
		}
	}

	interested := map[string]struct{}{}
	for i := range events {
		if events[i].AuthorizedBy != "" {
			interested[events[i].AuthorizedBy] = struct{}{}
		}
	}
	for _, s := range ownScopes {
		interested[s.AuthSHA] = struct{}{}
	}
	if len(interested) == 0 {
		return nil, nil
	}

	allScopes := ownScopes
	// Source (a): resolve foreign scope-entities only when id carries
	// `aiwf-authorized-by` events. This is the guarded grep — skipped for
	// scopeless entities and direct-scope openers alike.
	if history.HasAuthorizedBy(events) {
		openers, err := cliutil.AuthorizeOpeners(ctx, root)
		if err != nil {
			return nil, err //coverage:ignore git-read failure unreachable after HasCommits guards a valid repo
		}
		foreignNeeded := map[string]struct{}{}
		for sha := range interested {
			if ent, ok := openers[sha]; ok && ent != id {
				foreignNeeded[ent] = struct{}{}
			}
		}
		for ent := range foreignNeeded {
			scopes, err := cliutil.LoadEntityScopes(ctx, root, ent)
			if err != nil {
				return nil, err //coverage:ignore git-read failure unreachable after HasCommits guards a valid repo
			}
			allScopes = append(allScopes, scopes...)
		}
	}

	dateCache := map[string]string{}
	var views []ScopeView
	for _, s := range allScopes {
		if _, ok := interested[s.AuthSHA]; !ok {
			continue
		}
		opened := LookupCommitDateCached(ctx, root, s.AuthSHA, dateCache)
		var ended string
		if s.State == scope.StateEnded {
			if last := LastEventSHA(s, scope.StateEnded); last != "" {
				ended = LookupCommitDateCached(ctx, root, last, dateCache)
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

// LookupCommitDateCached returns the ISO-8601 author date of the
// commit at sha, caching results so we never hit `git show` twice
// for the same SHA in one show call. Errors fall back to an empty
// string (the caller renders dates as omitempty in JSON).
func LookupCommitDateCached(ctx context.Context, root, sha string, cache map[string]string) string {
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

// LastEventSHA returns the SHA of the latest event in s whose state
// equals match, or "" when none. Used by ScopeView assembly to look
// up the ending commit's date (when the scope is ended).
func LastEventSHA(s *scope.Scope, match scope.State) string {
	for i := len(s.Events) - 1; i >= 0; i-- {
		if s.Events[i].State == match {
			return s.Events[i].SHA
		}
	}
	return ""
}
