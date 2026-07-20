package render

import (
	"strings"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/entityview"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
)

// singlepass.go — E-0054 / M-0221. render's per-entity git-history
// fan-out (resolver.history per epic/milestone/AC + a re-walk and the
// authorize-opener grep per milestone inside show.LoadEntityScopeViews)
// collapses into ONE HEAD walk. check.WalkHeadCommits (extended with
// %aI + %s) supplies every commit once; historyIndex buckets them into
// the exact per-entity views the old greps produced, so every resolver
// read is an in-memory lookup — zero further git.
//
// The index reproduces the old queries exactly (the AC-3 differential
// pins this against the untouched ReadHistoryChain / LoadEntityScopeViews
// oracles):
//
//   - events[canon(id)] reproduces ReadHistoryChain([id]): commits whose
//     aiwf-entity OR aiwf-prior-entity trailer matches id (width-tolerant),
//     with a composite M-NNNN/AC-N folded into BOTH its own bucket and the
//     parent milestone's bucket — the bare-milestone path-prefix match.
//   - scopesByEntity[canon(id)] reproduces cliutil.LoadEntityScopes(id):
//     the scope FSM replayed over commits with an exact aiwf-entity: id
//     trailer (no prior-entity, no composite fold), matching
//     readEntityScopeCommits' `^aiwf-entity: <id>$` grep.
//   - openers reproduces cliutil.AuthorizeOpeners over the whole repo.
//   - dateBySHA supplies the %aI the scope views used a per-SHA `git show`
//     to fetch.
//
// The shared cliutil.ReplayScopes / cliutil.OpenersFrom and
// entityview.EventFromCommit are the single source of truth for the replay
// and event construction — render reuses them rather than adding a copy.
type historyIndex struct {
	events         map[string][]entityview.HistoryEvent // canon(id) → ReadHistoryChain([id]) reproduction
	scopesByEntity map[string][]*scope.Scope            // canon(id) → LoadEntityScopes(id) reproduction
	openers        map[string]string                    // auth-SHA → canonical scope-entity
	dateBySHA      map[string]string                    // full SHA → %aI author date
}

// buildHistoryIndex buckets one HEAD walk into the per-entity views the
// resolver reads. Pure: no git, no error — the single walk already
// happened (the fail-loud error path lives at its call site in RunSite).
func buildHistoryIndex(head []check.HeadCommit) *historyIndex {
	idx := &historyIndex{
		events:         map[string][]entityview.HistoryEvent{},
		scopesByEntity: map[string][]*scope.Scope{},
		openers:        map[string]string{},
		dateBySHA:      make(map[string]string, len(head)),
	}
	exactCommits := map[string][]cliutil.CommitTrailers{}
	all := make([]cliutil.CommitTrailers, 0, len(head))

	for i := range head {
		c := &head[i]
		idx.dateBySHA[c.SHA] = c.AuthorDate
		all = append(all, cliutil.CommitTrailers{SHA: c.SHA, Trailers: c.Trailers})

		// History-event buckets (entity + prior-entity + composite fold).
		// historyBucketKeys already returns de-duplicated keys, and each
		// commit is visited once (WalkHeadCommits is --reverse, oldest-first),
		// so no bucket receives this event twice.
		if ev, ok := entityview.EventFromCommit(c.SHA, c.AuthorDate, c.Subject, c.Body, c.Trailers); ok {
			for _, key := range historyBucketKeys(c.Trailers) {
				idx.events[key] = append(idx.events[key], ev)
			}
		}

		// Exact-entity buckets for scope replay: a real aiwf-entity trailer
		// only (no prior-entity, no composite fold), matching the
		// `^aiwf-entity: <id>$` grep readEntityScopeCommits uses. Dedup is
		// per-commit — trailerValues does NOT dedup, so a commit carrying the
		// same id twice (a doubled trailer, or mixed narrow/canonical widths
		// that canonicalize alike) would otherwise be replayed twice; the grep
		// matches such a commit once, so bucket it once. Distinct commits never
		// collide (each SHA is visited once), so a per-commit set suffices.
		seen := map[string]bool{}
		for _, v := range trailerValues(c.Trailers, gitops.TrailerEntity) {
			key := entity.Canonicalize(v)
			if seen[key] {
				continue
			}
			seen[key] = true
			exactCommits[key] = append(exactCommits[key], cliutil.CommitTrailers{SHA: c.SHA, Trailers: c.Trailers})
		}
	}

	idx.openers = cliutil.OpenersFrom(all)
	for key, commits := range exactCommits {
		idx.scopesByEntity[key] = cliutil.ReplayScopes(commits)
	}
	return idx
}

// historyBucketKeys returns the canonical bucket keys a commit belongs to
// for the history-event buckets: canon(v) for each aiwf-entity and
// aiwf-prior-entity value v, plus canon(parent) when v is a composite
// (M-NNNN/AC-N) so its events fold into the parent milestone's bucket.
// Keys dedupe so a commit naming one bucket twice buckets once.
func historyBucketKeys(trailers []gitops.Trailer) []string {
	seen := map[string]bool{}
	var keys []string
	// canon(v) of a non-empty trailer value is never empty, so no blank-key
	// guard is needed here (trailerValues already drops empty values).
	add := func(k string) {
		if seen[k] {
			return
		}
		seen[k] = true
		keys = append(keys, k)
	}
	for _, key := range []string{gitops.TrailerEntity, gitops.TrailerPriorEntity} {
		for _, v := range trailerValues(trailers, key) {
			add(entity.Canonicalize(v))
			if parent, _, ok := entity.ParseCompositeID(v); ok {
				add(entity.Canonicalize(parent))
			}
		}
	}
	return keys
}

// trailerValues returns every non-empty value of the named trailer key,
// in trailer order.
func trailerValues(trailers []gitops.Trailer, key string) []string {
	var out []string
	for _, tr := range trailers {
		if tr.Key == key {
			if v := strings.TrimSpace(tr.Value); v != "" {
				out = append(out, v)
			}
		}
	}
	return out
}

// scopeViewsFor reproduces show.LoadEntityScopeViews(id) from the index,
// with no git. It mirrors that verb's cost gates (HasOwnScope →
// ownScopes, HasAuthorizedBy → openers) so the assembled views are
// byte-identical to the grep-based path for every id width, not just
// canonical ones — the gates are pure filters over already-loaded data,
// and AssembleScopeViews is the shared assembly both paths run through.
func (r *Resolver) scopeViewsFor(id string) []entityview.ScopeView {
	canon := entity.Canonicalize(id)
	events := r.index.events[canon]

	var ownScopes []*scope.Scope
	if entityview.HasOwnScope(events) {
		ownScopes = r.index.scopesByEntity[canon]
	}
	var openers map[string]string
	if entityview.HasAuthorizedBy(events) {
		openers = r.index.openers
	}

	views, _ := entityview.AssembleScopeViews(id, events, ownScopes, openers,
		func(ent string) ([]*scope.Scope, error) {
			return r.index.scopesByEntity[entity.Canonicalize(ent)], nil
		},
		func(sha string) string { return r.index.dateBySHA[sha] },
	)
	return views
}
