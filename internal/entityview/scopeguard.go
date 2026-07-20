package entityview

// scopeguard.go — the read-verb guards for the repo-wide authorize-opener
// grep (E-0054 / M-0223). `aiwf history` (text) and `aiwf show` both used
// to run an unconditional `git log --grep 'aiwf-verb: authorize'` on every
// invocation to label scope chips / build the scope table. These predicates
// let each caller skip that grep when the entity's loaded events carry no
// scope data the grep would resolve. The predicate keys off the loaded
// event slice, not entity frontmatter: `aiwf-scope-ends` and the
// authorize-opener reference are commit trailers with no frontmatter
// counterpart, and the events are already loaded before the grep, so the
// check is free.

// HasAuthorizedBy reports whether any event carries an `aiwf-authorized-by`
// reference — i.e. the entity was worked under a foreign authorization
// scope. It is the guard for `aiwf show`'s global authorize-opener grep:
// source (a) — resolving which foreign scope authorized this entity's work
// — is the only consumer that needs the repo-wide opener map. Scopes opened
// directly on the entity (source (b)) come from the entity's own history via
// cliutil.LoadEntityScopes, so an active direct-scope opener (which has no
// `aiwf-authorized-by`) must NOT trigger the global grep.
func HasAuthorizedBy(events []HistoryEvent) bool {
	for i := range events {
		if events[i].AuthorizedBy != "" {
			return true
		}
	}
	return false
}

// HasScopeData reports whether any event carries scope provenance the
// history text renderer resolves via the global authorize-opener map: an
// `aiwf-authorized-by` reference or an `aiwf-scope-ends` terminator. It is
// the guard for `aiwf history`'s scope-entity map grep. When it returns
// false, the chip renderer never reads the map — an authorize opener's own
// `[scope: opened]` chip renders from e.Scope without the map — so the grep
// is pure waste.
func HasScopeData(events []HistoryEvent) bool {
	for i := range events {
		if events[i].AuthorizedBy != "" || len(events[i].ScopeEnds) > 0 {
			return true
		}
	}
	return false
}

// HasOwnScope reports whether any event is an authorize-opener on the entity
// itself (aiwf-verb: authorize + aiwf-scope: opened) — i.e. at least one scope
// was opened directly on it. It is show's guard for the per-entity
// cliutil.LoadEntityScopes(id) walk (source (b)): an entity whose own history
// carries no authorize-opener has no direct scopes, so that walk is skipped.
// Combined with the HasAuthorizedBy guard on the global grep, a scopeless
// entity resolves no scope table without any git read beyond the history it
// already loaded — the walk skipped by this guard is what makes `aiwf show`
// on a scopeless entity as cheap as `aiwf history` (E-0054 / M-0223).
//
// The predicate is exact: cliutil.LoadEntityScopes builds a scope only from an
// authorize+opened commit carrying aiwf-entity: id, and ReadHistory greps the
// same aiwf-entity: id (width-tolerantly), so the opener appears in both — if
// LoadEntityScopes would return a scope, HasOwnScope sees its opener event.
func HasOwnScope(events []HistoryEvent) bool {
	for i := range events {
		if events[i].Verb == "authorize" && events[i].Scope == "opened" {
			return true
		}
	}
	return false
}
