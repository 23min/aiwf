package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestRenderHistory_PreI2_5BackwardsCompat: a HistoryEvent with no
// I2.5 trailers (the pre-aiwf-I2.5 shape) renders without any chips
// or principal-via-agent rewrite. Guards the load-bearing
// backwards-compat promise from the plan: existing trailered
// commits must keep their original rendering.
func TestRenderHistory_PreI2_5BackwardsCompat(t *testing.T) {
	e := HistoryEvent{
		Date:   "2026-04-30T12:00:00+00:00",
		Actor:  "human/peter",
		Verb:   "promote",
		Detail: "feat: promote E-01 active",
		Commit: "1a2b3c4",
		To:     "active",
	}
	if got := renderActor(e); got != "human/peter" {
		t.Errorf("renderActor on pre-I2.5 event = %q, want %q", got, "human/peter")
	}
	chips := renderScopeChips(e, map[string]string{}, false)
	if chips != "" {
		t.Errorf("renderScopeChips on pre-I2.5 event = %q, want empty", chips)
	}
}

// TestBuildScopeEntityMap_GitFailureFallback: pointing at a non-
// repo directory makes the underlying git invocation fail. The
// helper swallows the error and returns an empty map so chip
// rendering falls back to "?" without blocking the verb.
func TestBuildScopeEntityMap_GitFailureFallback(t *testing.T) {
	tmp := t.TempDir()
	got := buildScopeEntityMap(context.Background(), tmp, nil)
	if len(got) != 0 {
		t.Errorf("buildScopeEntityMap on non-repo = %v, want empty map", got)
	}
}

// TestShow_CompositeIdWithScopes: querying `aiwf show M-001/AC-1`
// when an active scope authorized work on the parent milestone
// surfaces the scope under `scopes`. Exercises
// buildCompositeShowView's I2.5 wiring.
func TestShow_CompositeIdWithScopes(t *testing.T) {
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
	if out, err := runBin(t, root, binDir, nil, "add", "ac", "M-0001", "--title", "warmup works"); err != nil {
		t.Fatalf("aiwf add ac: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-0001", "--to", "ai/claude"); err != nil {
		t.Fatalf("authorize: %v\n%s", err, out)
	}
	// Agent acts on the AC inside the scope (status transition).
	if out, err := runBin(t, root, binDir, nil,
		"promote", "M-0001/AC-1", "met",
		"--actor", "ai/claude", "--principal", "human/peter"); err != nil {
		t.Fatalf("promote AC: %v\n%s", err, out)
	}

	out, err := runBin(t, root, binDir, nil, "show", "--format=json", "M-0001/AC-1")
	if err != nil {
		t.Fatalf("show composite: %v\n%s", err, out)
	}
	var env struct {
		Result ShowView `json:"result"`
	}
	if jErr := json.Unmarshal([]byte(out), &env); jErr != nil {
		t.Fatalf("parse JSON: %v\n%s", jErr, out)
	}
	if env.Result.AC == nil {
		t.Fatalf("AC nil; raw:\n%s", out)
	}
	if len(env.Result.Scopes) != 1 {
		t.Fatalf("composite scopes len = %d, want 1; raw:\n%s", len(env.Result.Scopes), out)
	}
	if env.Result.Scopes[0].Agent != "ai/claude" {
		t.Errorf("scope.agent = %q, want ai/claude", env.Result.Scopes[0].Agent)
	}
}

// TestShow_AncestorScopeNotInheritedWithoutAct documents the
// current loadEntityScopeViews scoping rule: a scope opened on a
// parent entity does NOT surface in a child's `aiwf show` until at
// least one commit on the child references the auth-SHA via
// aiwf-authorized-by. The function returns scopes that were either
// (a) opened on this entity directly, or (b) referenced by this
// entity's history.
//
// This is the conservative reading of "every scope that ever
// applied to this entity": "applied" = "was used in a commit that
// touched this entity." A scope opened on E-01 that has authorized
// no work yet is not yet "applied" to its child M-001.
//
// If we want descendants to surface ancestor scopes proactively
// (even before any agent commit references them), this test must
// flip — it pins the behavior so a design change is explicit, not
// silent.
func TestShow_AncestorScopeNotInheritedWithoutAct(t *testing.T) {
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
	// Open scope on E-01 but NEVER act on M-001 under it.
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-0001", "--to", "ai/claude"); err != nil {
		t.Fatalf("authorize: %v\n%s", err, out)
	}

	// E-01 surfaces the scope — opened on it directly.
	out, err := runBin(t, root, binDir, nil, "show", "--format=json", "E-0001")
	if err != nil {
		t.Fatalf("show E-01: %v\n%s", err, out)
	}
	var envE struct {
		Result ShowView `json:"result"`
	}
	if jErr := json.Unmarshal([]byte(out), &envE); jErr != nil {
		t.Fatalf("parse E-01 JSON: %v\n%s", jErr, out)
	}
	if len(envE.Result.Scopes) != 1 {
		t.Fatalf("E-01 scopes len = %d, want 1; raw:\n%s", len(envE.Result.Scopes), out)
	}

	// M-001 does NOT surface the ancestor scope — no commit on
	// M-001 references it yet.
	mout, mErr := runBin(t, root, binDir, nil, "show", "--format=json", "M-0001")
	if mErr != nil {
		t.Fatalf("show M-001: %v\n%s", mErr, mout)
	}
	var envM struct {
		Result ShowView `json:"result"`
	}
	if jErr := json.Unmarshal([]byte(mout), &envM); jErr != nil {
		t.Fatalf("parse M-001 JSON: %v\n%s", jErr, mout)
	}
	if len(envM.Result.Scopes) != 0 {
		t.Errorf("M-001 scopes len = %d, want 0 (ancestor scope not yet applied); raw:\n%s",
			len(envM.Result.Scopes), mout)
	}
}

// TestShow_MultipleScopesSorted: two scopes opened in sequence on
// the same entity, the second after the first is paused. Both
// surface in `scopes`, ordered by Opened ascending (oldest first).
func TestShow_MultipleScopesSorted(t *testing.T) {
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
	// Two scopes on the same entity, distinct agents, opened in
	// sequence. Pause the first so the second can open without
	// surprise (the kernel allows multiple parallel scopes on one
	// entity, but pausing makes the test's observed behavior
	// unambiguous).
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-0001", "--to", "ai/claude"); err != nil {
		t.Fatalf("authorize 1: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-0001", "--pause", "switching agent"); err != nil {
		t.Fatalf("authorize pause: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-0001", "--to", "bot/ci"); err != nil {
		t.Fatalf("authorize 2: %v\n%s", err, out)
	}

	out, err := runBin(t, root, binDir, nil, "show", "--format=json", "E-0001")
	if err != nil {
		t.Fatalf("show E-01: %v\n%s", err, out)
	}
	var env struct {
		Result ShowView `json:"result"`
	}
	if jErr := json.Unmarshal([]byte(out), &env); jErr != nil {
		t.Fatalf("parse JSON: %v\n%s", jErr, out)
	}
	if len(env.Result.Scopes) != 2 {
		t.Fatalf("scopes len = %d, want 2; raw:\n%s", len(env.Result.Scopes), out)
	}
	// Sorted by Opened ascending — the ai/claude scope opened first.
	if env.Result.Scopes[0].Agent != "ai/claude" {
		t.Errorf("[0].agent = %q, want ai/claude (older)", env.Result.Scopes[0].Agent)
	}
	if env.Result.Scopes[1].Agent != "bot/ci" {
		t.Errorf("[1].agent = %q, want bot/ci (newer)", env.Result.Scopes[1].Agent)
	}
	// Opened timestamps must be non-empty and ascending.
	if env.Result.Scopes[0].Opened == "" || env.Result.Scopes[1].Opened == "" {
		t.Errorf("Opened empty: %+v", env.Result.Scopes)
	}
	if env.Result.Scopes[0].Opened > env.Result.Scopes[1].Opened {
		t.Errorf("scopes not sorted ascending by Opened: [0]=%s [1]=%s",
			env.Result.Scopes[0].Opened, env.Result.Scopes[1].Opened)
	}
	// State sanity: the first is paused, the second active.
	if env.Result.Scopes[0].State != "paused" {
		t.Errorf("[0].state = %q, want paused", env.Result.Scopes[0].State)
	}
	if env.Result.Scopes[1].State != "active" {
		t.Errorf("[1].state = %q, want active", env.Result.Scopes[1].State)
	}
}
