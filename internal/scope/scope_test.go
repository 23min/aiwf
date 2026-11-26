package scope

import (
	"slices"
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

		// Illegal: a state outside the closed set moves nowhere.
		{"unknown → active", State("unknown"), StateActive, false},
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

// commit builds a Commit from a SHA and alternating trailer key/value
// pairs. Repeated keys keep their order, as git log returns them.
func commit(sha string, kv ...string) Commit {
	c := Commit{SHA: sha}
	for i := 0; i+1 < len(kv); i += 2 {
		c.Trailers = append(c.Trailers, gitops.Trailer{Key: kv[i], Value: kv[i+1]})
	}
	return c
}

// The commits `aiwf authorize` and a closing promote write on E-0003,
// reduced to the trailers Replay reads.
func opened(sha string) Commit {
	return commit(sha, gitops.TrailerVerb, "authorize", gitops.TrailerEntity, "E-0003",
		gitops.TrailerActor, "human/peter", gitops.TrailerTo, "ai/claude",
		gitops.TrailerScope, "opened", gitops.TrailerReason, "implement the engine")
}

func paused(sha string) Commit {
	return commit(sha, gitops.TrailerVerb, "authorize", gitops.TrailerEntity, "E-0003",
		gitops.TrailerActor, "human/peter", gitops.TrailerScope, "paused", gitops.TrailerReason, "blocked")
}

func resumed(sha string) Commit {
	return commit(sha, gitops.TrailerVerb, "authorize", gitops.TrailerEntity, "E-0003",
		gitops.TrailerActor, "human/peter", gitops.TrailerScope, "resumed", gitops.TrailerReason, "unblocked")
}

// closed is the promote that takes E-0003 to done, ending each named scope.
func closed(sha string, auths ...string) Commit {
	c := commit(sha, gitops.TrailerVerb, "promote", gitops.TrailerEntity, "E-0003",
		gitops.TrailerActor, "human/peter", gitops.TrailerTo, "done")
	for _, a := range auths {
		c.Trailers = append(c.Trailers, gitops.Trailer{Key: gitops.TrailerScopeEnds, Value: a})
	}
	return c
}

// endedByAuthorize is `aiwf authorize E-0003 --end`: an authorize commit
// that carries no aiwf-scope trailer.
func endedByAuthorize(sha, auth string) Commit {
	return commit(sha, gitops.TrailerVerb, "authorize", gitops.TrailerEntity, "E-0003",
		gitops.TrailerActor, "human/peter", gitops.TrailerReason, "withdrawn",
		gitops.TrailerScopeEnds, auth)
}

// TestReplay pins the rules by which Replay derives each scope's state
// from its entity's commits. A case's want lists every scope Replay
// returns, in open order, with the commits that moved it: the event list
// shows which scope a pause or resume landed on, which the final state
// alone does not.
func TestReplay(t *testing.T) {
	t.Parallel()
	type want struct {
		auth   string
		state  State
		events []string
	}
	tests := []struct {
		name    string
		history []Commit
		want    []want
	}{
		{
			name:    "an authorize commit carrying aiwf-scope: opened opens an active scope",
			history: []Commit{opened("A")},
			want:    []want{{"A", StateActive, []string{"A"}}},
		},
		{
			name: "no other commit opens a scope",
			history: []Commit{
				commit("X", gitops.TrailerVerb, "promote", gitops.TrailerEntity, "E-0003", gitops.TrailerScope, "opened"),
			},
			want: nil,
		},
		{
			// r2 makes A the scope most recently made active; p3 must still
			// land on B, the one opened later.
			name: "a pause moves the most recently opened active scope and no other",
			history: []Commit{
				opened("A"), opened("B"), paused("p1"), paused("p2"),
				resumed("r1"), resumed("r2"), paused("p3"),
			},
			want: []want{
				{"A", StateActive, []string{"A", "p2", "r2"}},
				{"B", StatePaused, []string{"B", "p1", "r1", "p3"}},
			},
		},
		{
			name:    "a pause or resume passes over a scope that has ended",
			history: []Commit{opened("A"), opened("B"), endedByAuthorize("e1", "B"), paused("p1"), resumed("r1")},
			want: []want{
				{"A", StateActive, []string{"A", "p1", "r1"}},
				{"B", StateEnded, []string{"B", "e1"}},
			},
		},
		{
			name:    "a resume moves the most recently opened paused scope and no other",
			history: []Commit{opened("A"), opened("B"), paused("p1"), paused("p2"), resumed("r1")},
			want: []want{
				{"A", StatePaused, []string{"A", "p2"}},
				{"B", StateActive, []string{"B", "p1", "r1"}},
			},
		},
		{
			name:    "a pause or resume that no scope can make changes nothing",
			history: []Commit{paused("p0"), opened("A"), resumed("r1"), paused("p1"), paused("p2")},
			want:    []want{{"A", StatePaused, []string{"A", "p1"}}},
		},
		{
			name:    "a commit ends every scope its aiwf-scope-ends trailers name, active or paused",
			history: []Commit{opened("A"), opened("B"), paused("p1"), closed("c1", "A", "B")},
			want: []want{
				{"A", StateEnded, []string{"A", "c1"}},
				{"B", StateEnded, []string{"B", "p1", "c1"}},
			},
		},
		{
			name:    "an end naming no scope on the entity moves none",
			history: []Commit{opened("A"), closed("c1", "9999999")},
			want:    []want{{"A", StateActive, []string{"A"}}},
		},
		{
			name: "an ended scope stays ended",
			history: []Commit{
				opened("A"), endedByAuthorize("e1", "A"),
				paused("p1"), resumed("r1"), closed("c1", "A"),
			},
			want: []want{{"A", StateEnded, []string{"A", "e1"}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got []want
			for _, s := range Replay(tt.history) {
				w := want{auth: s.AuthSHA, state: s.State}
				for _, e := range s.Events {
					w.events = append(w.events, e.SHA)
				}
				got = append(got, w)
			}
			if !slices.EqualFunc(got, tt.want, func(a, b want) bool {
				return a.auth == b.auth && a.state == b.state && slices.Equal(a.events, b.events)
			}) {
				t.Errorf("Replay =\n  %+v\nwant\n  %+v", got, tt.want)
			}
		})
	}
}

// TestReplay_RecordsTheScopeAndEachEvent pins what a scope carries
// besides its state: the entity, agent and principal from its opening
// commit, and each event's state and aiwf-reason.
func TestReplay_RecordsTheScopeAndEachEvent(t *testing.T) {
	t.Parallel()
	scopes := Replay([]Commit{opened("A"), paused("p1"), resumed("r1"), endedByAuthorize("e1", "A")})
	if len(scopes) != 1 {
		t.Fatalf("Replay returned %d scopes, want 1", len(scopes))
	}
	s := scopes[0]
	if s.Entity != "E-0003" || s.Agent != "ai/claude" || s.Principal != "human/peter" {
		t.Errorf("scope entity, agent, principal = %q, %q, %q; want E-0003, ai/claude, human/peter",
			s.Entity, s.Agent, s.Principal)
	}
	wantEvents := []Event{
		{SHA: "A", State: StateActive, Reason: "implement the engine"},
		{SHA: "p1", State: StatePaused, Reason: "blocked"},
		{SHA: "r1", State: StateActive, Reason: "unblocked"},
		{SHA: "e1", State: StateEnded, Reason: "withdrawn"},
	}
	if !slices.Equal(s.Events, wantEvents) {
		t.Errorf("events =\n  %+v\nwant\n  %+v", s.Events, wantEvents)
	}
}
