package scope

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

func TestIsLegalScopeTransition(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		from State
		to   State
		want bool
	}{
		// Legal.
		{"active → paused", StateActive, StatePaused, true},
		{"active → ended", StateActive, StateEnded, true},
		{"paused → active", StatePaused, StateActive, true},
		{"paused → ended", StatePaused, StateEnded, true},

		// Illegal: self-loops (every transition is meaningful).
		{"active → active", StateActive, StateActive, false},
		{"paused → paused", StatePaused, StatePaused, false},

		// Illegal: ended is terminal.
		{"ended → active", StateEnded, StateActive, false},
		{"ended → paused", StateEnded, StatePaused, false},
		{"ended → ended", StateEnded, StateEnded, false},

		// Illegal: skipping pause.
		{"active → active via something", StateActive, StateActive, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsLegalScopeTransition(tt.from, tt.to); got != tt.want {
				t.Errorf("IsLegalScopeTransition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

// trailerSet builds a small []gitops.Trailer for tests. Order doesn't
// matter for index-based lookup but we keep a consistent shape so the
// fixtures read like real commits.
func trailerSet(verb, entity, actor, scope, reason, to string, scopeEnds ...string) []gitops.Trailer {
	out := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: verb},
		{Key: gitops.TrailerEntity, Value: entity},
		{Key: gitops.TrailerActor, Value: actor},
	}
	if to != "" {
		out = append(out, gitops.Trailer{Key: gitops.TrailerTo, Value: to})
	}
	if scope != "" {
		out = append(out, gitops.Trailer{Key: gitops.TrailerScope, Value: scope})
	}
	if reason != "" {
		out = append(out, gitops.Trailer{Key: gitops.TrailerReason, Value: reason})
	}
	for _, end := range scopeEnds {
		out = append(out, gitops.Trailer{Key: gitops.TrailerScopeEnds, Value: end})
	}
	return out
}

// TestLoadScope_OpenerOnlyIsActive: the simplest case — just the
// authorize commit, no transitions yet.
func TestLoadScope_OpenerOnlyIsActive(t *testing.T) {
	t.Parallel()
	auth := "4b13a0f"
	history := []Commit{
		{SHA: auth, Trailers: trailerSet("authorize", "E-0003", "human/peter", "opened", "implement the engine", "ai/claude")},
	}
	s, err := LoadScope(auth, history)
	if err != nil {
		t.Fatalf("LoadScope: %v", err)
	}
	if s.State != StateActive {
		t.Errorf("State = %s, want %s", s.State, StateActive)
	}
	if s.Entity != "E-0003" || s.Agent != "ai/claude" || s.Principal != "human/peter" {
		t.Errorf("scope envelope mismatch: entity=%q agent=%q principal=%q", s.Entity, s.Agent, s.Principal)
	}
	if len(s.Events) != 1 {
		t.Fatalf("Events len = %d, want 1", len(s.Events))
	}
	if s.Events[0].State != StateActive || s.Events[0].SHA != auth {
		t.Errorf("opener event = %+v", s.Events[0])
	}
	if s.Events[0].Reason != "implement the engine" {
		t.Errorf("opener reason = %q, want from --reason", s.Events[0].Reason)
	}
}

// TestLoadScope_PauseResumeCycle: full cycle pause → resume → pause →
// resume — every transition lands in Events with the correct state.
func TestLoadScope_PauseResumeCycle(t *testing.T) {
	t.Parallel()
	auth := "4b13a0f"
	history := []Commit{
		{SHA: auth, Trailers: trailerSet("authorize", "E-0003", "human/peter", "opened", "", "ai/claude")},
		{SHA: "aaa1111", Trailers: trailerSet("authorize", "E-0003", "human/peter", "paused", "blocked by E-09", "")},
		{SHA: "bbb2222", Trailers: trailerSet("authorize", "E-0003", "human/peter", "resumed", "back to E-03", "")},
		{SHA: "ccc3333", Trailers: trailerSet("authorize", "E-0003", "human/peter", "paused", "stuck again", "")},
		{SHA: "ddd4444", Trailers: trailerSet("authorize", "E-0003", "human/peter", "resumed", "good now", "")},
	}
	s, err := LoadScope(auth, history)
	if err != nil {
		t.Fatalf("LoadScope: %v", err)
	}
	if s.State != StateActive {
		t.Errorf("final State = %s, want active", s.State)
	}
	wantStates := []State{StateActive, StatePaused, StateActive, StatePaused, StateActive}
	if len(s.Events) != len(wantStates) {
		t.Fatalf("Events len = %d, want %d", len(s.Events), len(wantStates))
	}
	for i, want := range wantStates {
		if s.Events[i].State != want {
			t.Errorf("Events[%d].State = %s, want %s", i, s.Events[i].State, want)
		}
	}
}

// TestLoadScope_AutoEndOnTerminalPromote: the scope-entity reaches a
// terminal status and the promote commit carries aiwf-scope-ends:
// pointing at the scope's auth SHA. Final state is ended.
func TestLoadScope_AutoEndOnTerminalPromote(t *testing.T) {
	t.Parallel()
	auth := "4b13a0f"
	history := []Commit{
		{SHA: auth, Trailers: trailerSet("authorize", "E-0003", "human/peter", "opened", "", "ai/claude")},
		// Agent acts under the scope (no transition).
		{SHA: "work111", Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "promote"},
			{Key: gitops.TrailerEntity, Value: "M-0007"},
			{Key: gitops.TrailerActor, Value: "ai/claude"},
			{Key: gitops.TrailerPrincipal, Value: "human/peter"},
			{Key: gitops.TrailerOnBehalfOf, Value: "human/peter"},
			{Key: gitops.TrailerAuthorizedBy, Value: auth},
		}},
		// Terminal-promote of the scope-entity, with auto-end trailer.
		{SHA: "endcom1", Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "promote"},
			{Key: gitops.TrailerEntity, Value: "E-0003"},
			{Key: gitops.TrailerActor, Value: "ai/claude"},
			{Key: gitops.TrailerPrincipal, Value: "human/peter"},
			{Key: gitops.TrailerOnBehalfOf, Value: "human/peter"},
			{Key: gitops.TrailerAuthorizedBy, Value: auth},
			{Key: gitops.TrailerTo, Value: "done"},
			{Key: gitops.TrailerScopeEnds, Value: auth},
		}},
	}
	s, err := LoadScope(auth, history)
	if err != nil {
		t.Fatalf("LoadScope: %v", err)
	}
	if s.State != StateEnded {
		t.Errorf("State = %s, want ended", s.State)
	}
	// Opener + end event = 2 (the work commit doesn't transition).
	if len(s.Events) != 2 {
		t.Fatalf("Events len = %d, want 2", len(s.Events))
	}
	if s.Events[1].State != StateEnded || s.Events[1].SHA != "endcom1" {
		t.Errorf("end event = %+v, want ended at endcom1", s.Events[1])
	}
}

// TestLoadScope_UnCancelDoesNotResurrect: after the scope-entity is
// auto-ended (e.g., by promote-to-cancelled), a later un-cancel
// commit on the same entity does NOT resurrect the scope. Strict
// end-on-terminal per provenance-model.md §"Scope states".
func TestLoadScope_UnCancelDoesNotResurrect(t *testing.T) {
	t.Parallel()
	auth := "4b13a0f"
	history := []Commit{
		{SHA: auth, Trailers: trailerSet("authorize", "E-0003", "human/peter", "opened", "", "ai/claude")},
		// End via terminal-promote.
		{SHA: "endcom1", Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "promote"},
			{Key: gitops.TrailerEntity, Value: "E-0003"},
			{Key: gitops.TrailerActor, Value: "human/peter"},
			{Key: gitops.TrailerTo, Value: "cancelled"},
			{Key: gitops.TrailerScopeEnds, Value: auth},
		}},
		// Human un-cancels later — but the scope stays ended.
		{SHA: "rev0001", Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "promote"},
			{Key: gitops.TrailerEntity, Value: "E-0003"},
			{Key: gitops.TrailerActor, Value: "human/peter"},
			{Key: gitops.TrailerTo, Value: "active"},
		}},
		// And someone tries a pause on the original auth SHA — must
		// not be applied (the walker stops at ended).
		{SHA: "pause11", Trailers: trailerSet("authorize", "E-0003", "human/peter", "paused", "after-the-fact", "")},
	}
	s, err := LoadScope(auth, history)
	if err != nil {
		t.Fatalf("LoadScope: %v", err)
	}
	if s.State != StateEnded {
		t.Errorf("State = %s, want ended (un-cancel must not resurrect)", s.State)
	}
	// Events: opener + end. The post-end commits are ignored.
	if len(s.Events) != 2 {
		t.Errorf("Events len = %d, want 2 (opener + end only)", len(s.Events))
	}
}

// TestLoadScope_RejectsBadOpener: history[0] must be the auth commit
// AND must carry aiwf-verb: authorize + aiwf-scope: opened.
func TestLoadScope_RejectsBadOpener(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		authSHA   string
		history   []Commit
		wantError string
	}{
		{
			name:      "empty history",
			authSHA:   "4b13a0f",
			history:   nil,
			wantError: "empty history",
		},
		{
			name:    "history[0] SHA mismatch",
			authSHA: "4b13a0f",
			history: []Commit{
				{SHA: "different", Trailers: trailerSet("authorize", "E-0003", "human/peter", "opened", "", "ai/claude")},
			},
			wantError: "does not match authSHA",
		},
		{
			name:    "opener verb is not authorize",
			authSHA: "4b13a0f",
			history: []Commit{
				{SHA: "4b13a0f", Trailers: trailerSet("promote", "E-0003", "human/peter", "", "", "")},
			},
			wantError: "not an authorize commit",
		},
		{
			name:    "opener missing aiwf-scope: opened",
			authSHA: "4b13a0f",
			history: []Commit{
				{SHA: "4b13a0f", Trailers: trailerSet("authorize", "E-0003", "human/peter", "paused", "", "ai/claude")},
			},
			wantError: "aiwf-scope: opened",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := LoadScope(tt.authSHA, tt.history)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantError)
			}
		})
	}
}

// TestLoadScope_RejectsIllegalTransition: a hand-crafted history with
// a resume on an active scope (no preceding pause) is malformed.
// LoadScope catches it via the FSM check.
func TestLoadScope_RejectsIllegalTransition(t *testing.T) {
	t.Parallel()
	auth := "4b13a0f"
	history := []Commit{
		{SHA: auth, Trailers: trailerSet("authorize", "E-0003", "human/peter", "opened", "", "ai/claude")},
		// Resume while still active — no pause before this.
		{SHA: "bad0001", Trailers: trailerSet("authorize", "E-0003", "human/peter", "resumed", "wrong", "")},
	}
	_, err := LoadScope(auth, history)
	if err == nil {
		t.Fatal("expected illegal-transition error, got nil")
	}
	if !strings.Contains(err.Error(), "illegal transition") {
		t.Errorf("error %q does not mention illegal transition", err.Error())
	}
}

// TestLoadScope_IgnoresUnrelatedCommits: commits in the history slice
// that aren't transitions for this scope (work commits under the
// scope, scope-ends pointing at a different auth SHA) are silently
// skipped — they're noise, not malformed input.
func TestLoadScope_IgnoresUnrelatedCommits(t *testing.T) {
	t.Parallel()
	auth := "4b13a0f"
	other := "9999999"
	history := []Commit{
		{SHA: auth, Trailers: trailerSet("authorize", "E-0003", "human/peter", "opened", "", "ai/claude")},
		// Work commit under the scope.
		{SHA: "work111", Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "promote"},
			{Key: gitops.TrailerEntity, Value: "M-0007"},
			{Key: gitops.TrailerActor, Value: "ai/claude"},
		}},
		// scope-ends for an entirely different scope.
		{SHA: "other11", Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "promote"},
			{Key: gitops.TrailerEntity, Value: "E-0009"},
			{Key: gitops.TrailerScopeEnds, Value: other},
		}},
		// Now actually pause.
		{SHA: "pause11", Trailers: trailerSet("authorize", "E-0003", "human/peter", "paused", "thinking", "")},
	}
	s, err := LoadScope(auth, history)
	if err != nil {
		t.Fatalf("LoadScope: %v", err)
	}
	if s.State != StatePaused {
		t.Errorf("State = %s, want paused", s.State)
	}
	if len(s.Events) != 2 {
		t.Errorf("Events len = %d, want 2 (opener + pause; work commit and other-scope-end are noise)", len(s.Events))
	}
}

// TestLoadScope_AutoEndDuringPaused: a paused scope can still be
// ended by a terminal-promote. Final state is ended, not paused.
func TestLoadScope_AutoEndDuringPaused(t *testing.T) {
	t.Parallel()
	auth := "4b13a0f"
	history := []Commit{
		{SHA: auth, Trailers: trailerSet("authorize", "E-0003", "human/peter", "opened", "", "ai/claude")},
		{SHA: "pause11", Trailers: trailerSet("authorize", "E-0003", "human/peter", "paused", "blocked", "")},
		// Human cancels the scope-entity while the scope is paused.
		{SHA: "endcom1", Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "promote"},
			{Key: gitops.TrailerEntity, Value: "E-0003"},
			{Key: gitops.TrailerActor, Value: "human/peter"},
			{Key: gitops.TrailerTo, Value: "cancelled"},
			{Key: gitops.TrailerScopeEnds, Value: auth},
		}},
	}
	s, err := LoadScope(auth, history)
	if err != nil {
		t.Fatalf("LoadScope: %v", err)
	}
	if s.State != StateEnded {
		t.Errorf("State = %s, want ended", s.State)
	}
	if len(s.Events) != 3 {
		t.Errorf("Events len = %d, want 3 (opener + pause + end)", len(s.Events))
	}
}

// TestLoadScope_MultipleScopeEndsOnSameCommit: when a single
// terminal-promote ends multiple scopes (rare but possible), the
// scope being loaded is ended only when the trailer matches its
// auth SHA. The other scope-ends targets are noise to this scope.
func TestLoadScope_MultipleScopeEndsOnSameCommit(t *testing.T) {
	t.Parallel()
	auth := "4b13a0f"
	history := []Commit{
		{SHA: auth, Trailers: trailerSet("authorize", "E-0003", "human/peter", "opened", "", "ai/claude")},
		// Terminal-promote ending two scopes at once.
		{SHA: "endcom1", Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "promote"},
			{Key: gitops.TrailerEntity, Value: "E-0003"},
			{Key: gitops.TrailerActor, Value: "human/peter"},
			{Key: gitops.TrailerTo, Value: "done"},
			{Key: gitops.TrailerScopeEnds, Value: "9999999"}, // someone else's scope
			{Key: gitops.TrailerScopeEnds, Value: auth},      // ours
		}},
	}
	s, err := LoadScope(auth, history)
	if err != nil {
		t.Fatalf("LoadScope: %v", err)
	}
	if s.State != StateEnded {
		t.Errorf("State = %s, want ended (auth SHA appears among multiple scope-ends)", s.State)
	}
}
