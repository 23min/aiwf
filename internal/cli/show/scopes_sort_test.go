package show_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/show"
	"github.com/23min/aiwf/internal/scope"
)

// TestAssembleScopeViews_SortsChronologicallyAcrossTimezones pins
// M-0269/AC-3 (G-0428): the shared sort in AssembleScopeViews (the
// pure, git-free core both show and render funnel through) must order
// by the true chronological instant, not by lexical string comparison
// of the `%aI`-formatted (author-local-offset) date strings.
//
// "later" carries a lexically SMALLER string (day 01) but a
// chronologically LATER instant (2024-01-02T07:00:00Z, from the
// -08:00 offset); "earlier" carries a lexically LARGER string (day
// 02) but a chronologically EARLIER instant (2024-01-02T01:00:00Z).
// A lexical sort places "later" first (wrong); a correct chronological
// sort places "earlier" first.
func TestAssembleScopeViews_SortsChronologicallyAcrossTimezones(t *testing.T) {
	t.Parallel()
	ownScopes := []*scope.Scope{
		{AuthSHA: "sha-later", Entity: "E-0001", State: scope.StateActive},
		{AuthSHA: "sha-earlier", Entity: "E-0001", State: scope.StateActive},
	}
	dates := map[string]string{
		"sha-later":   "2024-01-01T23:00:00-08:00", // instant: 2024-01-02T07:00:00Z
		"sha-earlier": "2024-01-02T01:00:00+00:00", // instant: 2024-01-02T01:00:00Z
	}
	dateOf := func(sha string) string { return dates[sha] }
	foreignScopes := func(ent string) ([]*scope.Scope, error) {
		t.Fatalf("foreignScopes(%s) called; no event references a foreign scope-entity in this fixture", ent)
		return nil, nil
	}

	views, err := show.AssembleScopeViews("E-0001", nil, ownScopes, nil, foreignScopes, dateOf)
	if err != nil {
		t.Fatalf("AssembleScopeViews: %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("views = %+v, want exactly 2", views)
	}
	if views[0].AuthSHA != "sha-earlier" || views[1].AuthSHA != "sha-later" {
		t.Errorf("order = [%s, %s], want [sha-earlier, sha-later] (true chronological order)",
			views[0].AuthSHA, views[1].AuthSHA)
	}
}

// TestAssembleScopeViews_EmptyOpenedSortsFirst pins parseOpened's
// fallback for an empty/unparseable date string (the shape
// LookupCommitDateCached itself falls back to on a `git show`
// failure): it parses to the zero time, so it sorts before any
// successfully-dated entry — matching the previous lexical
// comparison's behavior, where "" < any non-empty RFC3339 string.
func TestAssembleScopeViews_EmptyOpenedSortsFirst(t *testing.T) {
	t.Parallel()
	ownScopes := []*scope.Scope{
		{AuthSHA: "sha-dated", Entity: "E-0001", State: scope.StateActive},
		{AuthSHA: "sha-unparseable", Entity: "E-0001", State: scope.StateActive},
	}
	dates := map[string]string{
		"sha-dated": "2024-01-02T01:00:00+00:00",
		// sha-unparseable deliberately absent: dateOf returns "".
	}
	dateOf := func(sha string) string { return dates[sha] }
	foreignScopes := func(ent string) ([]*scope.Scope, error) {
		t.Fatalf("foreignScopes(%s) called; no event references a foreign scope-entity in this fixture", ent)
		return nil, nil
	}

	views, err := show.AssembleScopeViews("E-0001", nil, ownScopes, nil, foreignScopes, dateOf)
	if err != nil {
		t.Fatalf("AssembleScopeViews: %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("views = %+v, want exactly 2", views)
	}
	if views[0].AuthSHA != "sha-unparseable" || views[1].AuthSHA != "sha-dated" {
		t.Errorf("order = [%s, %s], want [sha-unparseable, sha-dated] (empty Opened sorts first)",
			views[0].AuthSHA, views[1].AuthSHA)
	}
}
