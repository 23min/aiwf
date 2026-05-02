package main

import (
	"strings"
	"testing"
)

// TestRenderActor: the actor column shows `principal via agent`
// when the principal differs from the actor (the agent-acts-for-
// human pattern from I2.5). Direct human acts (no principal) render
// the actor verbatim.
func TestRenderActor(t *testing.T) {
	tests := []struct {
		name string
		e    HistoryEvent
		want string
	}{
		{"direct human", HistoryEvent{Actor: "human/peter"}, "human/peter"},
		{"agent for principal", HistoryEvent{Actor: "ai/claude", Principal: "human/peter"}, "human/peter via ai/claude"},
		{"principal == actor (defensive)", HistoryEvent{Actor: "human/peter", Principal: "human/peter"}, "human/peter"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderActor(tt.e); got != tt.want {
				t.Errorf("renderActor = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRenderScopeChips covers the three chip variants: scope
// lifecycle on authorize commits, the [<scope-entity> <sha>] chip on
// scope-authorized rows, and per-end chips on terminal-promote rows.
func TestRenderScopeChips(t *testing.T) {
	scopeEntities := map[string]string{
		"4b13a0fdeadbeef": "E-03",
		"abc1234deadbeef": "E-09",
	}

	tests := []struct {
		name     string
		e        HistoryEvent
		showAuth bool
		want     string
	}{
		{
			name: "no chips",
			e:    HistoryEvent{Verb: "promote"},
			want: "",
		},
		{
			name: "authorize opened",
			e:    HistoryEvent{Verb: "authorize", Scope: "opened"},
			want: "  [scope: opened]",
		},
		{
			name: "authorize paused",
			e:    HistoryEvent{Verb: "authorize", Scope: "paused"},
			want: "  [scope: paused]",
		},
		{
			name: "scope-authorized agent verb",
			e:    HistoryEvent{Verb: "promote", AuthorizedBy: "4b13a0fdeadbeef"},
			want: "  [E-03 4b13a0f]",
		},
		{
			name:     "scope-authorized with --show-authorization",
			e:        HistoryEvent{Verb: "promote", AuthorizedBy: "4b13a0fdeadbeef"},
			showAuth: true,
			want:     "  [E-03 4b13a0fdeadbeef]",
		},
		{
			name: "terminal-promote ends one scope",
			e:    HistoryEvent{Verb: "promote", AuthorizedBy: "4b13a0fdeadbeef", ScopeEnds: []string{"4b13a0fdeadbeef"}},
			want: "  [E-03 4b13a0f] [E-03 ended]",
		},
		{
			name: "terminal-promote ends two scopes",
			e:    HistoryEvent{Verb: "promote", ScopeEnds: []string{"4b13a0fdeadbeef", "abc1234deadbeef"}},
			want: "  [E-03 ended] [E-09 ended]",
		},
		{
			name: "unknown auth-sha falls back to ?",
			e:    HistoryEvent{AuthorizedBy: "ffffffffeeeeeee"},
			want: "  [? fffffff]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderScopeChips(tt.e, scopeEntities, tt.showAuth); got != tt.want {
				t.Errorf("renderScopeChips = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRenderHistory_AuthorizationFlow walks an end-to-end story:
// human authorizes ai/claude on E-01 → agent promotes M-001 inside
// scope → human terminal-promotes E-01 (which ends the scope). The
// resulting history rendering carries each chip.
func TestRenderHistory_AuthorizationFlow(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := strings.TrimSuffix(bin, "/aiwf")
	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add epic: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--epic", "E-01", "--title", "Cache"); err != nil {
		t.Fatalf("aiwf add milestone: %v\n%s", err, out)
	}
	// Open scope on E-01.
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-01", "--to", "ai/claude"); err != nil {
		t.Fatalf("authorize: %v\n%s", err, out)
	}
	// Agent promotes M-001 inside the scope.
	if out, err := runBin(t, root, binDir, nil,
		"promote", "M-001", "in_progress",
		"--actor", "ai/claude", "--principal", "human/peter"); err != nil {
		t.Fatalf("promote M-001: %v\n%s", err, out)
	}

	out, err := runBin(t, root, binDir, nil, "history", "E-01")
	if err != nil {
		t.Fatalf("history: %v\n%s", err, out)
	}
	if !strings.Contains(out, "[scope: opened]") {
		t.Errorf("expected [scope: opened] chip on the authorize event:\n%s", out)
	}

	mout, err := runBin(t, root, binDir, nil, "history", "M-001")
	if err != nil {
		t.Fatalf("history M-001: %v\n%s", err, mout)
	}
	if !strings.Contains(mout, "[E-01 ") {
		t.Errorf("expected [E-01 <sha>] chip on the agent's promote:\n%s", mout)
	}
	if !strings.Contains(mout, "human/peter via ai/claude") {
		t.Errorf("expected `human/peter via ai/claude` actor rendering:\n%s", mout)
	}

	// --show-authorization expands the SHA inline.
	mout2, err := runBin(t, root, binDir, nil, "history", "--show-authorization", "M-001")
	if err != nil {
		t.Fatalf("history --show-authorization: %v\n%s", err, mout2)
	}
	// Look for a chip whose SHA portion is longer than 7 chars.
	if !strings.Contains(mout2, "[E-01 ") {
		t.Errorf("expected [E-01 ...] chip:\n%s", mout2)
	}
}
