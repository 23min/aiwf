package entityview_test

import (
	"testing"

	"github.com/23min/aiwf/internal/entityview"
)

// TestHasScopeData covers the history-side guard: the grep is needed iff
// some loaded event carries an aiwf-authorized-by reference OR an
// aiwf-scope-ends terminator. The load-bearing case is the active
// direct-scope opener (Scope set, but no AuthorizedBy / ScopeEnds): the
// predicate must return false there, because RenderScopeChips renders the
// opener's own [scope: opened] chip from e.Scope without the map.
func TestHasScopeData(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		events []entityview.HistoryEvent
		want   bool
	}{
		{"nil slice", nil, false},
		{"empty slice", []entityview.HistoryEvent{}, false},
		{
			"plain events, no scope data",
			[]entityview.HistoryEvent{
				{Verb: "add", Actor: "human/peter"},
				{Verb: "promote", To: "active"},
			},
			false,
		},
		{
			"active opener only — Scope set, no AuthorizedBy/ScopeEnds",
			[]entityview.HistoryEvent{
				{Verb: "authorize", Scope: "opened"},
			},
			false,
		},
		{
			"authorized-by present",
			[]entityview.HistoryEvent{
				{Verb: "add"},
				{Verb: "promote", AuthorizedBy: "deadbeef"},
			},
			true,
		},
		{
			"scope-ends present",
			[]entityview.HistoryEvent{
				{Verb: "promote", ScopeEnds: []string{"deadbeef"}},
			},
			true,
		},
		{
			"scope-ends present alongside plain events",
			[]entityview.HistoryEvent{
				{Verb: "add"},
				{Verb: "authorize", Scope: "opened"},
				{Verb: "promote", To: "done", ScopeEnds: []string{"abc1234"}},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := entityview.HasScopeData(tt.events); got != tt.want {
				t.Errorf("HasScopeData(%+v) = %v, want %v", tt.events, got, tt.want)
			}
		})
	}
}

// TestHasOwnScope covers show's guard for the per-entity LoadEntityScopes
// walk: true iff the entity's own history carries an authorize-opener
// (Verb == authorize && Scope == opened). A promote worked *under* a foreign
// scope (AuthorizedBy set, but no own authorize verb) must return false — its
// scope table comes from the global grep, not a direct walk.
func TestHasOwnScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		events []entityview.HistoryEvent
		want   bool
	}{
		{"nil slice", nil, false},
		{"empty slice", []entityview.HistoryEvent{}, false},
		{
			"plain events, no authorize",
			[]entityview.HistoryEvent{{Verb: "add"}, {Verb: "promote", To: "active"}},
			false,
		},
		{
			"worked under a foreign scope — AuthorizedBy but no own authorize",
			[]entityview.HistoryEvent{{Verb: "promote", AuthorizedBy: "deadbeef", OnBehalfOf: "human/peter"}},
			false,
		},
		{
			"authorize event but not opened (paused)",
			[]entityview.HistoryEvent{{Verb: "authorize", Scope: "paused"}},
			false,
		},
		{
			"own authorize-opener present",
			[]entityview.HistoryEvent{{Verb: "add"}, {Verb: "authorize", Scope: "opened"}},
			true,
		},
		{
			"scope-ended entity still has its opener",
			[]entityview.HistoryEvent{
				{Verb: "authorize", Scope: "opened"},
				{Verb: "promote", To: "done", ScopeEnds: []string{"abc1234"}},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := entityview.HasOwnScope(tt.events); got != tt.want {
				t.Errorf("HasOwnScope(%+v) = %v, want %v", tt.events, got, tt.want)
			}
		})
	}
}

// TestHasAuthorizedBy covers the show-side guard: the global grep is needed
// iff some loaded event carries an aiwf-authorized-by reference (source (a),
// resolving a foreign scope). Unlike HasScopeData, an aiwf-scope-ends
// terminator does NOT make the grep needed — show's scope table is built
// from AuthorizedBy events plus direct scopes, never from scope-ends. The
// active-opener case must return false so its own scopes still come from the
// width-tolerant direct derivation, not the global grep.
func TestHasAuthorizedBy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		events []entityview.HistoryEvent
		want   bool
	}{
		{"nil slice", nil, false},
		{"empty slice", []entityview.HistoryEvent{}, false},
		{
			"plain events, no scope data",
			[]entityview.HistoryEvent{
				{Verb: "add"},
				{Verb: "promote", To: "active"},
			},
			false,
		},
		{
			"active opener only — no AuthorizedBy",
			[]entityview.HistoryEvent{
				{Verb: "authorize", Scope: "opened"},
			},
			false,
		},
		{
			"scope-ends present but no AuthorizedBy — show does not need the grep",
			[]entityview.HistoryEvent{
				{Verb: "promote", To: "done", ScopeEnds: []string{"abc1234"}},
			},
			false,
		},
		{
			"authorized-by present",
			[]entityview.HistoryEvent{
				{Verb: "add"},
				{Verb: "promote", AuthorizedBy: "deadbeef"},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := entityview.HasAuthorizedBy(tt.events); got != tt.want {
				t.Errorf("HasAuthorizedBy(%+v) = %v, want %v", tt.events, got, tt.want)
			}
		})
	}
}
