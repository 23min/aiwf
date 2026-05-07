package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/gitops"
	"github.com/23min/ai-workflow-v2/internal/scope"
)

// TestProvenance_AuthorizedAgentPromote: a human authorizes an agent,
// the agent runs `aiwf promote M-001 in_progress --actor ai/claude
// --principal human/peter`. The commit's trailer set carries the
// full provenance: aiwf-actor=ai/claude, aiwf-principal=human/peter,
// aiwf-on-behalf-of=human/peter, aiwf-authorized-by=<auth-sha>.
func TestProvenance_AuthorizedAgentPromote(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

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
	if out, err := runBin(t, root, binDir, nil, "promote", "E-01", "active"); err != nil {
		t.Fatalf("aiwf promote E-01 active: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--title", "Cache warmup", "--epic", "E-01"); err != nil {
		t.Fatalf("aiwf add milestone: %v\n%s", err, out)
	}

	// Open a scope on E-01 for ai/claude.
	if out, err := runBin(t, root, binDir, nil,
		"authorize", "E-01", "--to", "ai/claude", "--reason", "implement E-01 end-to-end"); err != nil {
		t.Fatalf("aiwf authorize: %v\n%s", err, out)
	}

	// Capture the auth SHA so we can match it against the agent's
	// promote trailers.
	scopes, err := loadEntityScopes(context.Background(), root, "E-01")
	if err != nil {
		t.Fatalf("loadEntityScopes: %v", err)
	}
	if len(scopes) != 1 || scopes[0].State != scope.StateActive {
		t.Fatalf("expected one active scope on E-01; got %+v", scopes)
	}
	authSHA := scopes[0].AuthSHA

	// Agent runs the verb.
	out, runErr := runBin(t, root, binDir, nil,
		"promote", "M-001", "in_progress",
		"--actor", "ai/claude",
		"--principal", "human/peter")
	if runErr != nil {
		t.Fatalf("aiwf promote (agent): %v\n%s", runErr, out)
	}

	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	hasTrailer(t, tr, "aiwf-verb", "promote")
	hasTrailer(t, tr, "aiwf-entity", "M-001")
	hasTrailer(t, tr, "aiwf-actor", "ai/claude")
	hasTrailer(t, tr, "aiwf-principal", "human/peter")
	hasTrailer(t, tr, "aiwf-on-behalf-of", "human/peter")
	// authSHA from loadEntityScopes is the full hash; the trailer
	// likewise carries the full SHA. Compare directly.
	hasTrailer(t, tr, "aiwf-authorized-by", authSHA)
}

// TestProvenance_AgentRefusedOutOfScope: the agent has an active
// scope on E-01, but tries to act on a milestone under E-09 (which
// doesn't reach E-01). The verb is refused before any disk state
// changes.
func TestProvenance_AgentRefusedOutOfScope(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

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
		t.Fatalf("aiwf add epic E-01: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "promote", "E-01", "active"); err != nil {
		t.Fatalf("aiwf promote E-01: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Unrelated"); err != nil {
		t.Fatalf("aiwf add epic E-02: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "promote", "E-02", "active"); err != nil {
		t.Fatalf("aiwf promote E-02: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--title", "Out-of-scope", "--epic", "E-02"); err != nil {
		t.Fatalf("aiwf add milestone under E-02: %v\n%s", err, out)
	}
	// Authorize the agent on E-01 only.
	if out, err := runBin(t, root, binDir, nil,
		"authorize", "E-01", "--to", "ai/claude", "--reason", "scoped to E-01"); err != nil {
		t.Fatalf("aiwf authorize: %v\n%s", err, out)
	}

	// Agent attempts to promote a milestone under E-02 — out of
	// scope. Refusal expected.
	out, err := runBin(t, root, binDir, nil,
		"promote", "M-001", "in_progress",
		"--actor", "ai/claude",
		"--principal", "human/peter")
	if err == nil {
		t.Fatalf("expected refusal; got success:\n%s", out)
	}
	if !strings.Contains(out, "no active scope") {
		t.Errorf("expected no-active-scope message; got:\n%s", out)
	}
}

// TestProvenance_ScopeEntityFollowsPriorEntityChain: scenario #8
// from docs/pocv3/plans/provenance-model-plan.md §"Test scenarios".
// Open a scope on a milestone; reallocate that milestone (renumber
// it). The historical authorize commit's aiwf-entity still points
// at the old id, but the scope-entity resolver walks forward through
// the aiwf-prior-entity chain so the agent can keep operating under
// the same authorization. The authorize commit's SHA stays valid.
func TestProvenance_ScopeEntityFollowsPriorEntityChain(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

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
	if out, err := runBin(t, root, binDir, nil, "promote", "E-01", "active"); err != nil {
		t.Fatalf("aiwf promote E-01 active: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "milestone", "--tdd", "none", "--title", "First", "--epic", "E-01"); err != nil {
		t.Fatalf("aiwf add M-001: %v\n%s", err, out)
	}
	// Authorize the agent on M-001.
	if out, err := runBin(t, root, binDir, nil,
		"authorize", "M-001", "--to", "ai/claude", "--reason", "scoped to M-001"); err != nil {
		t.Fatalf("aiwf authorize: %v\n%s", err, out)
	}
	// Reallocate M-001 → M-002 (or whatever the renumber target is).
	if out, err := runBin(t, root, binDir, nil, "reallocate", "M-001"); err != nil {
		t.Fatalf("aiwf reallocate: %v\n%s", err, out)
	}

	// The new id is whatever reallocate picked. Read it from the
	// most-recent commit's aiwf-entity trailer.
	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	var newID string
	for _, x := range tr {
		if x.Key == "aiwf-entity" {
			newID = x.Value
			break
		}
	}
	if newID == "" || newID == "M-001" {
		t.Fatalf("reallocate did not produce a new id; head trailers: %+v", tr)
	}

	// Verify resolveCurrentEntityID walks the chain.
	resolved, err := resolveCurrentEntityID(context.Background(), root, "M-001")
	if err != nil {
		t.Fatalf("resolveCurrentEntityID: %v", err)
	}
	if resolved != newID {
		t.Errorf("resolveCurrentEntityID(M-001) = %q, want %q", resolved, newID)
	}

	// Agent runs a verb on the NEW id under the original
	// authorization. Should be allowed (chain resolves).
	out, runErr := runBin(t, root, binDir, nil,
		"promote", newID, "in_progress",
		"--actor", "ai/claude",
		"--principal", "human/peter")
	if runErr != nil {
		t.Fatalf("aiwf promote (post-reallocate): %v\n%s", runErr, out)
	}
}

// TestProvenance_AgentAddMilestoneInScope: an agent under an active
// scope on E-01 can `aiwf add milestone --epic E-01` (creation refs
// = [E-01], which trivially reach the scope-entity). The same verb
// against E-02 (out of scope) is refused.
func TestProvenance_AgentAddMilestoneInScope(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

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
		t.Fatalf("aiwf add E-01: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Unrelated"); err != nil {
		t.Fatalf("aiwf add E-02: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "promote", "E-01", "active"); err != nil {
		t.Fatalf("aiwf promote E-01: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil,
		"authorize", "E-01", "--to", "ai/claude", "--reason", "scoped to E-01"); err != nil {
		t.Fatalf("aiwf authorize: %v\n%s", err, out)
	}

	// In scope: agent adds a milestone under E-01.
	if out, err := runBin(t, root, binDir, nil,
		"add", "milestone", "--tdd", "none", "--title", "In-scope", "--epic", "E-01",
		"--actor", "ai/claude", "--principal", "human/peter"); err != nil {
		t.Fatalf("agent add milestone in scope: %v\n%s", err, out)
	}

	// Out of scope: same agent under same scope, but milestone is
	// being added under E-02. Refused.
	out, runErr := runBin(t, root, binDir, nil,
		"add", "milestone", "--tdd", "none", "--title", "Out-of-scope", "--epic", "E-02",
		"--actor", "ai/claude", "--principal", "human/peter")
	if runErr == nil {
		t.Fatalf("expected refusal for out-of-scope add; got success:\n%s", out)
	}
	if !strings.Contains(out, "no active scope") {
		t.Errorf("expected no-active-scope message; got:\n%s", out)
	}
}

// TestProvenance_TerminalPromoteEmitsScopeEnds: when the scope-
// entity itself is promoted to a terminal state, the commit carries
// `aiwf-scope-ends: <auth-sha>` per active scope on that entity.
// Subsequent loadEntityScopes calls report the scope as ended.
func TestProvenance_TerminalPromoteEmitsScopeEnds(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

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
	if out, err := runBin(t, root, binDir, nil, "promote", "E-01", "active"); err != nil {
		t.Fatalf("aiwf promote E-01 active: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil,
		"authorize", "E-01", "--to", "ai/claude", "--reason", "implement E-01"); err != nil {
		t.Fatalf("aiwf authorize: %v\n%s", err, out)
	}
	scopes, err := loadEntityScopes(context.Background(), root, "E-01")
	if err != nil {
		t.Fatalf("loadEntityScopes (pre): %v", err)
	}
	if len(scopes) != 1 {
		t.Fatalf("expected 1 scope; got %d", len(scopes))
	}
	authSHA := scopes[0].AuthSHA

	// Human terminal-promotes E-01 directly.
	out, runErr := runBin(t, root, binDir, nil, "promote", "E-01", "done")
	if runErr != nil {
		t.Fatalf("aiwf promote E-01 done: %v\n%s", runErr, out)
	}

	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	hasTrailer(t, tr, "aiwf-verb", "promote")
	hasTrailer(t, tr, "aiwf-entity", "E-01")
	hasTrailer(t, tr, "aiwf-to", "done")
	hasTrailer(t, tr, "aiwf-scope-ends", authSHA)

	// Scope is now ended.
	scopes, err = loadEntityScopes(context.Background(), root, "E-01")
	if err != nil {
		t.Fatalf("loadEntityScopes (post): %v", err)
	}
	if len(scopes) != 1 || scopes[0].State != scope.StateEnded {
		t.Errorf("expected one ended scope; got %+v", scopes)
	}
}
