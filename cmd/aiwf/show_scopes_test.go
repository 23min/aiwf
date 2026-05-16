package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestShow_ScopesView_AuthorizationFlow walks the load-bearing
// scenario: human authorizes ai/claude on E-01, agent promotes a
// child milestone, terminal-promote of E-01 ends the scope. After
// the flow `aiwf show` for both E-01 and the milestone surfaces the
// scope under `scopes`. The agent column reflects the scope's
// agent, and the state moves through active → ended.
func TestShow_ScopesView_AuthorizationFlow(t *testing.T) {
	t.Parallel()
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
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Cache"); err != nil {
		t.Fatalf("aiwf add milestone: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-0001", "--to", "ai/claude"); err != nil {
		t.Fatalf("authorize: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil,
		"promote", "M-0001", "in_progress",
		"--actor", "ai/claude", "--principal", "human/peter"); err != nil {
		t.Fatalf("promote M-001: %v\n%s", err, out)
	}

	// JSON show on E-01: scopes block has one entry, state=active.
	out, err := runBin(t, root, binDir, nil, "show", "--format=json", "E-0001")
	if err != nil {
		t.Fatalf("show E-01 json: %v\n%s", err, out)
	}
	var env struct {
		Result ShowView `json:"result"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, out)
	}
	if len(env.Result.Scopes) != 1 {
		t.Fatalf("E-01 scopes len = %d, want 1; raw:\n%s", len(env.Result.Scopes), out)
	}
	s := env.Result.Scopes[0]
	if s.Agent != "ai/claude" {
		t.Errorf("scope.agent = %q, want ai/claude", s.Agent)
	}
	if s.Principal != "human/peter" {
		t.Errorf("scope.principal = %q, want human/peter", s.Principal)
	}
	if s.Entity != "E-0001" {
		t.Errorf("scope.entity = %q, want E-01", s.Entity)
	}
	if s.State != "active" {
		t.Errorf("scope.state = %q, want active", s.State)
	}
	if s.EndedAt != "" {
		t.Errorf("scope.ended_at = %q, want empty", s.EndedAt)
	}
	if s.AuthSHA == "" {
		t.Errorf("scope.auth_sha is empty")
	}

	// JSON show on M-001: same scope surfaces (the agent acted under
	// it). Same auth_sha, same state.
	mout, mErr := runBin(t, root, binDir, nil, "show", "--format=json", "M-0001")
	if mErr != nil {
		t.Fatalf("show M-001 json: %v\n%s", mErr, mout)
	}
	var envM struct {
		Result ShowView `json:"result"`
	}
	if jErr := json.Unmarshal([]byte(mout), &envM); jErr != nil {
		t.Fatalf("parse JSON M-001: %v\n%s", jErr, mout)
	}
	if len(envM.Result.Scopes) != 1 {
		t.Fatalf("M-001 scopes len = %d, want 1; raw:\n%s", len(envM.Result.Scopes), out)
	}
	if envM.Result.Scopes[0].AuthSHA != s.AuthSHA {
		t.Errorf("M-001 scope auth_sha = %q, E-01 scope auth_sha = %q; want equal",
			envM.Result.Scopes[0].AuthSHA, s.AuthSHA)
	}

	// Terminal-promote E-01 → ends the scope. Re-check JSON show.
	if pOut, pErr := runBin(t, root, binDir, nil, "promote", "E-0001", "active"); pErr != nil {
		t.Fatalf("promote E-01 active: %v\n%s", pErr, pOut)
	}
	if pOut, pErr := runBin(t, root, binDir, nil, "promote", "E-0001", "done"); pErr != nil {
		t.Fatalf("promote E-01 done: %v\n%s", pErr, pOut)
	}
	endOut, endErr := runBin(t, root, binDir, nil, "show", "--format=json", "E-0001")
	if endErr != nil {
		t.Fatalf("show E-01 json post-end: %v\n%s", endErr, endOut)
	}
	var envEnd struct {
		Result ShowView `json:"result"`
	}
	if jErr := json.Unmarshal([]byte(endOut), &envEnd); jErr != nil {
		t.Fatalf("parse JSON post-end: %v\n%s", jErr, endOut)
	}
	if len(envEnd.Result.Scopes) != 1 {
		t.Fatalf("post-end scopes len = %d, want 1; raw:\n%s", len(envEnd.Result.Scopes), out)
	}
	if envEnd.Result.Scopes[0].State != "ended" {
		t.Errorf("post-end scope.state = %q, want ended", envEnd.Result.Scopes[0].State)
	}
	if envEnd.Result.Scopes[0].EndedAt == "" {
		t.Errorf("post-end scope.ended_at is empty")
	}
}

// TestShow_ScopesView_NoScopes asserts that an entity with no scope
// involvement omits the scopes field (omitempty in JSON, no
// "Scopes (N):" block in text).
func TestShow_ScopesView_NoScopes(t *testing.T) {
	t.Parallel()
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
		t.Fatalf("aiwf add: %v\n%s", err, out)
	}
	out, err := runBin(t, root, binDir, nil, "show", "E-0001")
	if err != nil {
		t.Fatalf("show: %v\n%s", err, out)
	}
	if strings.Contains(out, "Scopes (") {
		t.Errorf("expected no Scopes block; got:\n%s", out)
	}
}
