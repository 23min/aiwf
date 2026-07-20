package history

import (
	"context"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entityview"
)

// scopeguard.go — the read-verb guard for the repo-wide authorize-opener
// grep (E-0054 / M-0223). `aiwf history` (text) used to run an
// unconditional `git log --grep 'aiwf-verb: authorize'` on every
// invocation to label scope chips. This guard lets the caller skip that
// grep when the entity's loaded events carry no scope data the grep
// would resolve. The predicate keys off the loaded event slice, not
// entity frontmatter, and the events are already loaded before the
// grep, so the check is free.

// ScopeMapFor returns the authorize-opener → scope-entity map the history
// text renderer hands to RenderScopeChips, or nil when the loaded events
// carry no scope data. When nil, no row reads the map (an opener's own
// [scope: opened] chip renders from e.Scope), so the repo-wide
// cliutil.AuthorizeOpeners grep is skipped entirely — the E-0054 / M-0223
// guard. A grep failure yields nil, so chips render "?" rather than blocking
// the verb, exactly as the old best-effort BuildScopeEntityMap did.
func ScopeMapFor(ctx context.Context, root string, events []entityview.HistoryEvent) map[string]string {
	if !entityview.HasScopeData(events) {
		return nil
	}
	m, _ := cliutil.AuthorizeOpeners(ctx, root)
	return m
}
