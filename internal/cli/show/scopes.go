package show

import (
	"context"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entityview"
	"github.com/23min/aiwf/internal/scope"
)

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
// foreign scope (entityview.HasAuthorizedBy). An active direct-scope opener,
// which has no `aiwf-authorized-by`, resolves entirely from source (b)
// without the grep (E-0054 read-verb guard).
//
// Empty / pre-aiwf repos return (nil, nil).
//
// The git-reading orchestration lives here (in the CLI layer, via
// cliutil) rather than in internal/entityview: the neutral package stays
// free of internal/cli dependencies, and cliutil.LoadEntityScopes /
// cliutil.AuthorizeOpeners are full scope-FSM replays, not the kind of
// small self-contained helper worth duplicating across the boundary.
func LoadEntityScopeViews(ctx context.Context, root, id string) ([]entityview.ScopeView, error) {
	if !cliutil.HasCommits(ctx, root) {
		return nil, nil
	}
	events, err := entityview.ReadHistory(ctx, root, id)
	if err != nil {
		return nil, err //coverage:ignore git-read failure unreachable after HasCommits guards a valid repo
	}
	// Source (b): scopes opened directly on id, derived from id's own
	// history (width-tolerant) — but only walk when the already-loaded
	// events show id actually has an own authorize-opener. A scopeless
	// entity skips this walk entirely (E-0054 / M-0223).
	var ownScopes []*scope.Scope
	if entityview.HasOwnScope(events) {
		ownScopes, err = cliutil.LoadEntityScopes(ctx, root, id)
		if err != nil {
			return nil, err //coverage:ignore git-read failure unreachable after HasCommits guards a valid repo
		}
	}
	// Source (a): the repo-wide opener map, loaded only when id was worked
	// under a foreign scope (M-0223 guard). When nil, AssembleScopeViews
	// finds no foreign scope-entity and never invokes the foreign resolver.
	var openers map[string]string
	if entityview.HasAuthorizedBy(events) {
		openers, err = cliutil.AuthorizeOpeners(ctx, root)
		if err != nil {
			return nil, err //coverage:ignore git-read failure unreachable after HasCommits guards a valid repo
		}
	}

	dateCache := map[string]string{}
	return entityview.AssembleScopeViews(id, events, ownScopes, openers,
		func(ent string) ([]*scope.Scope, error) { return cliutil.LoadEntityScopes(ctx, root, ent) },
		func(sha string) string { return entityview.LookupCommitDateCached(ctx, root, sha, dateCache) },
	)
}
