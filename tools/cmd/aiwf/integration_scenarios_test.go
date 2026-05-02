package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
)

// initRepoFor sets up a temp git repo with the given email and runs
// `aiwf init` so verbs can run. Returns (root, binDir).
func initRepoFor(t *testing.T, email string) (root, binDir string) {
	t.Helper()
	bin := aiwfBinary(t)
	binDir = strings.TrimSuffix(bin, "/aiwf")
	root = t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", email},
		{"config", "user.name", "Test User"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	return root, binDir
}

// TestScenario_TerminalPromoteEndsMultipleParallelScopes covers
// plan §4 #3 with the multi-scope wrinkle. Two scopes opened on the
// same epic; terminal-promote of the epic must end BOTH atomically
// (one aiwf-scope-ends per active scope on the entity).
func TestScenario_TerminalPromoteEndsMultipleParallelScopes(t *testing.T) {
	root, binDir := initRepoFor(t, "peter@example.com")
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("aiwf add epic: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-01", "--to", "ai/claude"); err != nil {
		t.Fatalf("authorize 1: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-01", "--to", "bot/ci"); err != nil {
		t.Fatalf("authorize 2: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "promote", "E-01", "active"); err != nil {
		t.Fatalf("promote active: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "promote", "E-01", "done"); err != nil {
		t.Fatalf("promote done: %v\n%s", err, out)
	}
	tr, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	endsCount := 0
	for _, t := range tr {
		if t.Key == gitops.TrailerScopeEnds {
			endsCount++
		}
	}
	if endsCount != 2 {
		t.Errorf("got %d aiwf-scope-ends trailers, want 2 (one per active scope on E-01); trailers=%+v", endsCount, tr)
	}

	// `aiwf show E-01` reflects both scopes ended.
	out, err := runBin(t, root, binDir, nil, "show", "--format=json", "E-01")
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
		t.Fatalf("scopes len = %d, want 2", len(env.Result.Scopes))
	}
	for i, s := range env.Result.Scopes {
		if s.State != "ended" {
			t.Errorf("scope[%d].state = %q, want ended", i, s.State)
		}
		if s.EndedAt == "" {
			t.Errorf("scope[%d].ended_at empty", i)
		}
	}
}

// TestScenario_PivotMidFlight covers plan §4 #4: pause E-01, open
// E-02, work on E-02, pause E-02, resume E-01. Each scoped commit
// references the right authorize SHA.
func TestScenario_PivotMidFlight(t *testing.T) {
	root, binDir := initRepoFor(t, "peter@example.com")
	for _, args := range [][]string{
		{"add", "epic", "--title", "Engine"},
		{"add", "epic", "--title", "Pipeline"},
		{"add", "milestone", "--epic", "E-01", "--title", "Cache"},
		{"add", "milestone", "--epic", "E-02", "--title", "Sink"},
	} {
		if out, err := runBin(t, root, binDir, nil, args...); err != nil {
			t.Fatalf("setup %v: %v\n%s", args, err, out)
		}
	}
	// Open scope on E-01.
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-01", "--to", "ai/claude"); err != nil {
		t.Fatalf("authorize E-01: %v\n%s", err, out)
	}
	// Agent acts under E-01 scope.
	if out, err := runBin(t, root, binDir, nil,
		"promote", "M-001", "in_progress",
		"--actor", "ai/claude", "--principal", "human/peter"); err != nil {
		t.Fatalf("agent on M-001 (under E-01): %v\n%s", err, out)
	}
	// Pause E-01, open E-02.
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-01", "--pause", "switching focus"); err != nil {
		t.Fatalf("pause E-01: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-02", "--to", "ai/claude"); err != nil {
		t.Fatalf("authorize E-02: %v\n%s", err, out)
	}
	// Agent acts under E-02 scope.
	if out, err := runBin(t, root, binDir, nil,
		"promote", "M-002", "in_progress",
		"--actor", "ai/claude", "--principal", "human/peter"); err != nil {
		t.Fatalf("agent on M-002 (under E-02): %v\n%s", err, out)
	}
	// Pause E-02, resume E-01. Agent acts under E-01 again.
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-02", "--pause", "back to E-01"); err != nil {
		t.Fatalf("pause E-02: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-01", "--resume", "continuing E-01 work"); err != nil {
		t.Fatalf("resume E-01: %v\n%s", err, out)
	}
	// Cancel M-001 under the resumed E-01 scope (any verb that touches a child works).
	if out, err := runBin(t, root, binDir, nil,
		"cancel", "M-001",
		"--actor", "ai/claude", "--principal", "human/peter"); err != nil {
		t.Fatalf("cancel M-001 under resumed E-01: %v\n%s", err, out)
	}

	// Inspect the M-001 cancel commit's authorized-by trailer; it must
	// be the E-01 scope's auth SHA.
	cancelTrailers, err := gitops.HeadTrailers(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	authBy := ""
	for _, tr := range cancelTrailers {
		if tr.Key == gitops.TrailerAuthorizedBy {
			authBy = tr.Value
		}
	}
	if authBy == "" {
		t.Fatalf("cancel M-001 commit carries no aiwf-authorized-by; trailers=%+v", cancelTrailers)
	}
	// Resolve the E-01 scope's auth SHA (the first authorize-opened
	// commit on E-01) and verify it matches.
	logOut, err := runGit(root, "log", "--reverse", "-E",
		"--grep", "^aiwf-verb: authorize$",
		"--grep", "^aiwf-scope: opened$",
		"--grep", "^aiwf-entity: E-01$",
		"--all-match",
		"--pretty=tformat:%H")
	if err != nil {
		t.Fatalf("git log: %v\n%s", err, logOut)
	}
	expectedAuthSHA := strings.Fields(strings.TrimSpace(logOut))[0]
	if authBy != expectedAuthSHA {
		t.Errorf("cancel M-001 aiwf-authorized-by = %s, want E-01 opener SHA %s", authBy, expectedAuthSHA)
	}
}

// TestScenario_ReallocatePreservesAuthorization covers plan §4 #8:
// open scope on E-01, reallocate E-01 → E-02, agent acts on a child
// under what's now E-02. The standing rules must NOT fire
// out-of-scope (the rename chain resolves) and the verb succeeds.
func TestScenario_ReallocatePreservesAuthorization(t *testing.T) {
	root, binDir := initRepoFor(t, "peter@example.com")
	for _, args := range [][]string{
		{"add", "epic", "--title", "Decoy"},  // burn E-01 so reallocate has a target
		{"add", "epic", "--title", "Engine"}, // E-02 — the one we'll reallocate
		{"add", "milestone", "--epic", "E-02", "--title", "Cache"},
	} {
		if out, err := runBin(t, root, binDir, nil, args...); err != nil {
			t.Fatalf("setup %v: %v\n%s", args, err, out)
		}
	}
	// Open scope on E-02 (the soon-to-be-reallocated epic).
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-02", "--to", "ai/claude"); err != nil {
		t.Fatalf("authorize: %v\n%s", err, out)
	}
	// Cancel E-01 (the decoy) so reallocate lands E-02 → some lower id.
	// Actually reallocate just bumps to next free; we don't need to do that.
	// Reallocate E-02 — the kernel picks the next free id.
	if out, err := runBin(t, root, binDir, nil, "reallocate", "E-02"); err != nil {
		t.Fatalf("reallocate: %v\n%s", err, out)
	}
	// Find what E-02 became. List epic dirs.
	entries, err := os.ReadDir(filepath.Join(root, "work", "epics"))
	if err != nil {
		t.Fatal(err)
	}
	var newEpicID string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, "-engine") && !strings.HasPrefix(name, "E-02-") {
			parts := strings.SplitN(name, "-", 2)
			newEpicID = parts[0]
			break
		}
	}
	if newEpicID == "" {
		t.Fatalf("could not find renamed Engine epic in %v", entries)
	}
	if newEpicID == "E-02" {
		t.Fatalf("reallocate produced same id %s", newEpicID)
	}

	// Agent acts on M-001 (still its child). The chain E-02 → newEpicID
	// must resolve so the act is allowed.
	if out, err := runBin(t, root, binDir, nil,
		"promote", "M-001", "in_progress",
		"--actor", "ai/claude", "--principal", "human/peter"); err != nil {
		t.Fatalf("agent verb after reallocate failed: %v\n%s", err, out)
	}

	// `aiwf check` must NOT fire out-of-scope.
	out, _ := runBin(t, root, binDir, nil, "check")
	if strings.Contains(out, "provenance-authorization-out-of-scope") {
		t.Errorf("out-of-scope fired despite rename-chain resolution:\n%s", out)
	}
}

// TestScenario_MultiCloneIdentity covers plan §4 #9: two repos with
// different `git config user.email` produce trailers naming each
// committer respectively. The kernel reads identity at runtime;
// no aiwf.yaml field carries it.
func TestScenario_MultiCloneIdentity(t *testing.T) {
	rootA, binDirA := initRepoFor(t, "alice@example.com")
	rootB, binDirB := initRepoFor(t, "bob@example.com")
	if out, err := runBin(t, rootA, binDirA, nil, "add", "epic", "--title", "Alice's epic"); err != nil {
		t.Fatalf("alice add: %v\n%s", err, out)
	}
	if out, err := runBin(t, rootB, binDirB, nil, "add", "epic", "--title", "Bob's epic"); err != nil {
		t.Fatalf("bob add: %v\n%s", err, out)
	}
	trA, err := gitops.HeadTrailers(context.Background(), rootA)
	if err != nil {
		t.Fatal(err)
	}
	trB, err := gitops.HeadTrailers(context.Background(), rootB)
	if err != nil {
		t.Fatal(err)
	}
	actorA, actorB := "", ""
	for _, t := range trA {
		if t.Key == gitops.TrailerActor {
			actorA = t.Value
		}
	}
	for _, t := range trB {
		if t.Key == gitops.TrailerActor {
			actorB = t.Value
		}
	}
	if actorA != "human/alice" {
		t.Errorf("rootA actor = %q, want human/alice", actorA)
	}
	if actorB != "human/bob" {
		t.Errorf("rootB actor = %q, want human/bob", actorB)
	}
}

// TestScenario_LockContentionThenAuditOnlyRecovery covers G24
// end-to-end: a process holds .git/index.lock; `aiwf cancel` fails
// with the lock-contention diagnostic; the user releases the lock
// and finishes the work via a manual commit; `aiwf cancel
// --audit-only --reason "..."` backfills the audit trail.
func TestScenario_LockContentionThenAuditOnlyRecovery(t *testing.T) {
	root, binDir := initRepoFor(t, "peter@example.com")
	if out, err := runBin(t, root, binDir, nil, "add", "gap", "--title", "Validators leak"); err != nil {
		t.Fatalf("aiwf add gap: %v\n%s", err, out)
	}
	// Create the lock file synthetically — git refuses to start a
	// commit while .git/index.lock exists, deterministically firing
	// the lock-contention path.
	lockPath := filepath.Join(root, ".git", "index.lock")
	if err := os.WriteFile(lockPath, []byte("synthetic"), 0o644); err != nil {
		t.Fatalf("create lock: %v", err)
	}
	out, lockErr := runBin(t, root, binDir, nil, "cancel", "G-001", "--reason", "test under lock")
	if lockErr == nil {
		t.Fatalf("expected aiwf cancel to fail under lock contention; got success:\n%s", out)
	}
	if !strings.Contains(out, "index.lock") {
		t.Errorf("expected lock-contention diagnostic to mention index.lock; got:\n%s", out)
	}
	// Release the lock. Note: the verb's rollback ALSO ran under the
	// lock, so it failed too — the file is already at `wontfix` on
	// disk (the verb's intended write was made; only the commit
	// failed). The "manual recovery" path is to commit that pending
	// change, then run --audit-only to backfill the trailers.
	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove lock: %v", err)
	}
	gapRel := mustFindFile(t, root, "G-001-")
	if out, err := runGit(root, "add", gapRel); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	if out, err := runGit(root, "commit", "-m", "manually mark G-001 wontfix"); err != nil {
		t.Fatalf("manual commit: %v\n%s", err, out)
	}
	// Audit-only recovery records the audit trail.
	if out, err := runBin(t, root, binDir, nil,
		"cancel", "G-001", "--audit-only", "--reason", "lock contention recovery"); err != nil {
		t.Fatalf("aiwf cancel --audit-only: %v\n%s", err, out)
	}
	historyOut, err := runBin(t, root, binDir, nil, "history", "G-001")
	if err != nil {
		t.Fatalf("history: %v\n%s", err, historyOut)
	}
	if !strings.Contains(historyOut, "[audit-only: lock contention recovery]") {
		t.Errorf("expected [audit-only: ...] chip; got:\n%s", historyOut)
	}
}

// TestScenario_RepeatedPauseResumeCycle covers plan §4 (informally
// implied by FSM cycle correctness): pause → resume → pause →
// resume on the same scope, twice, lands the scope in `active` and
// LoadScope walks every transition without raising.
func TestScenario_RepeatedPauseResumeCycle(t *testing.T) {
	root, binDir := initRepoFor(t, "peter@example.com")
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("add epic: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-01", "--to", "ai/claude"); err != nil {
		t.Fatalf("authorize: %v\n%s", err, out)
	}
	for i, args := range [][]string{
		{"authorize", "E-01", "--pause", "first pause"},
		{"authorize", "E-01", "--resume", "first resume"},
		{"authorize", "E-01", "--pause", "second pause"},
		{"authorize", "E-01", "--resume", "second resume"},
	} {
		if out, err := runBin(t, root, binDir, nil, args...); err != nil {
			t.Fatalf("step %d %v: %v\n%s", i, args, err, out)
		}
	}
	out, err := runBin(t, root, binDir, nil, "show", "--format=json", "E-01")
	if err != nil {
		t.Fatalf("show: %v\n%s", err, out)
	}
	var env struct {
		Result ShowView `json:"result"`
	}
	if jErr := json.Unmarshal([]byte(out), &env); jErr != nil {
		t.Fatalf("parse JSON: %v\n%s", jErr, out)
	}
	if len(env.Result.Scopes) != 1 {
		t.Fatalf("scopes len = %d, want 1", len(env.Result.Scopes))
	}
	if env.Result.Scopes[0].State != "active" {
		t.Errorf("scope.state = %q, want active (after final --resume)", env.Result.Scopes[0].State)
	}
	if env.Result.Scopes[0].EventCount < 5 {
		t.Errorf("event_count = %d, want >= 5 (open + 4 transitions)", env.Result.Scopes[0].EventCount)
	}
	// `aiwf check` must not fire any provenance findings on this
	// well-formed history.
	checkOut, _ := runBin(t, root, binDir, nil, "check")
	if strings.Contains(checkOut, "provenance-") {
		t.Errorf("clean pause/resume cycle produced provenance findings:\n%s", checkOut)
	}
}

// TestScenario_AuthorizeWithOnBehalfOfNeutralAtCheck covers the
// G22-deferred sub-agent-delegation question: an authorize commit
// that itself carries on-behalf-of/authorized-by is not refused by
// the standing rules. The kernel reserves the policy decision; the
// check pass stays neutral.
func TestScenario_AuthorizeWithOnBehalfOfNeutralAtCheck(t *testing.T) {
	root, binDir := initRepoFor(t, "peter@example.com")
	if out, err := runBin(t, root, binDir, nil, "add", "epic", "--title", "Engine"); err != nil {
		t.Fatalf("add epic: %v\n%s", err, out)
	}
	// Open a real authorize commit so its SHA can be referenced.
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-01", "--to", "ai/claude"); err != nil {
		t.Fatalf("authorize: %v\n%s", err, out)
	}
	authSHAout, err := runGit(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse: %v\n%s", err, authSHAout)
	}
	authSHA := strings.TrimSpace(authSHAout)

	// Hand-craft a sub-authorize commit that names the existing scope
	// in its provenance trailers. This is the G22-deferred shape; the
	// verb refuses to write it (non-human authorizer), so we synthesize
	// it directly with `git commit`.
	msg := "chore: sub-authorize\n\n" +
		"aiwf-verb: authorize\n" +
		"aiwf-entity: E-01\n" +
		"aiwf-actor: ai/claude\n" +
		"aiwf-principal: human/peter\n" +
		"aiwf-to: ai/sub-agent\n" +
		"aiwf-scope: opened\n" +
		"aiwf-on-behalf-of: human/peter\n" +
		"aiwf-authorized-by: " + authSHA + "\n"
	if out, err := runGit(root, "commit", "--allow-empty", "-m", msg); err != nil {
		t.Fatalf("hand-crafted authorize commit: %v\n%s", err, out)
	}
	// `aiwf check` must NOT fire `provenance-trailer-incoherent` for
	// this combination (G22 deferred). Other rules may still fire on
	// this commit's other trailers, so we look specifically for the
	// authorize+on-behalf-of-incoherent rule absence.
	out, _ := runBin(t, root, binDir, nil, "check")
	// Check that no message mentions our hand-crafted sub-authorize SHA
	// in a trailer-incoherent finding. The find we want absent is
	// "authorize + on-behalf-of refused" — i.e., a rule we did NOT
	// implement, which proves the policy is deferred.
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "provenance-trailer-incoherent") &&
			strings.Contains(line, "authorize") &&
			strings.Contains(line, "on-behalf-of") {
			t.Errorf("kernel fired a trailer-incoherent rule for the G22-deferred authorize+on-behalf-of combo:\n%s", line)
		}
	}
}
