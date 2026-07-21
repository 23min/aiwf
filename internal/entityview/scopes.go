package entityview

import (
	"context"
	"os/exec"
	"sort"
	"strings"
	"time"

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

// AssembleScopeViews is the pure, git-free core of `aiwf show`'s scope
// table: given an entity's loaded history events, its own scopes (source
// b), the repo-wide authorize-opener map, a resolver for a foreign
// scope-entity's scopes (source a), and a commit-date resolver, it
// assembles the scope-view list. The git-touching gather is the caller's
// job, so render's single pass (E-0054 / M-0221) assembles byte-identical
// views from its shared HEAD walk (opener map + replayed scopes + %aI
// dates) through this exact code path — no fourth copy of the assembly
// logic.
//
// The M-0223 cost gates live in the caller (only load ownScopes / openers
// when the events warrant it); Assemble stays gate-free because the gates
// are pure optimizations — an empty openers map yields no foreign
// lookups, and empty ownScopes contributes nothing, so the assembled
// views are identical whether or not the caller gated.
//
// foreignScopes is invoked only for a scope-entity that (a) an
// aiwf-authorized-by event references and (b) differs from id, so a
// caller that passes a nil openers map never triggers a foreign resolve.
// dateOf resolves a commit SHA to its author date (git show for the verb
// path; a lookup into the shared pass's %aI map for render).
func AssembleScopeViews(
	id string,
	events []HistoryEvent,
	ownScopes []*scope.Scope,
	openers map[string]string,
	foreignScopes func(ent string) ([]*scope.Scope, error),
	dateOf func(sha string) string,
) ([]ScopeView, error) {
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
	foreignNeeded := map[string]struct{}{}
	for sha := range interested {
		if ent, ok := openers[sha]; ok && ent != id {
			foreignNeeded[ent] = struct{}{}
		}
	}
	for ent := range foreignNeeded {
		scopes, err := foreignScopes(ent)
		if err != nil {
			return nil, err //coverage:ignore foreign-resolver error is git-read failure, unreachable after HasCommits in the production caller
		}
		allScopes = append(allScopes, scopes...)
	}

	var views []ScopeView
	for _, s := range allScopes {
		if _, ok := interested[s.AuthSHA]; !ok {
			continue
		}
		opened := dateOf(s.AuthSHA)
		var ended string
		if s.State == scope.StateEnded {
			if last := LastEventSHA(s, scope.StateEnded); last != "" {
				ended = dateOf(last)
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
		return parseOpened(views[i].Opened).Before(parseOpened(views[j].Opened))
	})
	return views, nil
}

// parseOpened parses a ScopeView.Opened timestamp (git's %aI format —
// RFC3339 with the commit author's local UTC offset preserved, not
// normalized to UTC) into a comparable time.Time so the sort above
// compares true chronological instants rather than
// lexical strings (G-0428): two commits from authors in different
// timezones can carry a lexically-earlier string for a
// chronologically-later instant. Empty or malformed input parses to
// the zero time, sorting first — matching the previous lexical
// comparison's empty-string-sorts-first behavior.
func parseOpened(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
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
