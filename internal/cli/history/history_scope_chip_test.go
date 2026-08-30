package history_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/history"
	"github.com/23min/aiwf/internal/entityview"
)

// TestRenderScopeChips_EndedChipNamesTheScopeOnAuthorizeRows pins which
// of the two spellings the ended-chip takes, and it is a pair because
// the row is what decides.
//
// A promote or cancel that ends a scope also took the entity terminal,
// so naming the entity is true of both. `aiwf authorize --end` ends the
// scope and leaves the entity running — a commit shape that did not
// exist before this mode — where the same chip would assert a closure
// that never happened, on the surface an operator reads to reconstruct
// what occurred.
func TestRenderScopeChips_EndedChipNamesTheScopeOnAuthorizeRows(t *testing.T) {
	t.Parallel()

	const authSHA = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	scopeEntities := map[string]string{authSHA: "E-0001"}

	cases := []struct {
		name string
		verb string
		want string
	}{
		{
			name: "an operator end leaves the entity running",
			verb: "authorize",
			want: "[scope: " + entityview.ShortHash(authSHA) + " ended]",
		},
		{
			name: "a closing promote ended the entity too",
			verb: "promote",
			want: "[E-0001 ended]",
		},
		{
			name: "a closing cancel ended the entity too",
			verb: "cancel",
			want: "[E-0001 ended]",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := history.RenderScopeChips(
				entityview.HistoryEvent{Verb: tc.verb, ScopeEnds: []string{authSHA}},
				scopeEntities, false)
			if got != "  "+tc.want {
				t.Errorf("chip = %q, want %q", got, "  "+tc.want)
			}
		})
	}
}

// TestRenderScopeChips_UnknownScopeEntityFallsBackToTheSHA covers the
// arm where the ending row is not an authorize and the auth-sha maps to
// no entity — a scope opened outside the window the caller walked. The
// chip degrades to the abbreviation rather than printing an empty
// bracket.
func TestRenderScopeChips_UnknownScopeEntityFallsBackToTheSHA(t *testing.T) {
	t.Parallel()

	const authSHA = "cafef00dcafef00dcafef00dcafef00dcafef00d"
	got := history.RenderScopeChips(
		entityview.HistoryEvent{Verb: "promote", ScopeEnds: []string{authSHA}},
		map[string]string{}, false)
	if want := "  [" + entityview.ShortHash(authSHA) + " ended]"; got != want {
		t.Errorf("chip = %q, want %q", got, want)
	}
}
