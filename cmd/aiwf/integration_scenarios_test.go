package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/gitops"
)

// initRepoFor sets up a temp git repo with the given email, wires
// it to a bare-repo upstream (so `git rev-parse @{u}` resolves
// during scenario tests), and runs `aiwf init`. Returns (root,
// binDir).
//
// The upstream is needed because `aiwf check`'s untrailered-audit
// pass now skips the scan with a `provenance-untrailered-scope-
// undefined` advisory when no upstream is configured (issue #5
// sub-item 2). Real consumer repos have upstreams; the test setup
// mirrors that.
func initRepoFor(t *testing.T, email string) (root, binDir string) {
	t.Helper()
	bin := aiwfBinary(t)
	binDir = strings.TrimSuffix(bin, "/aiwf")
	root = setupGitRepoWithUpstream(t, email)
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	return root, binDir
}

// setupGitRepoWithUpstream creates a temp git repo with an initial
// empty commit, a bare-repo remote at `origin`, and an upstream-
// tracked branch so @{u} resolves to a real ref. The empty commit
// is pushed so HEAD..@{u} starts as an empty range; subsequent
// commits are unpushed and visible to the audit pass.
//
// Returns the working repo's root path. The bare upstream lives in
// a sibling tempdir owned by the same test.
func setupGitRepoWithUpstream(t *testing.T, email string) string {
	t.Helper()
	upstream := t.TempDir()
	if out, err := runGit(upstream, "init", "--bare", "-q"); err != nil {
		t.Fatalf("git init bare: %v\n%s", err, out)
	}
	root := t.TempDir()
	if out, err := runGit(root, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", email},
		{"config", "user.name", "Test User"},
		{"remote", "add", "origin", upstream},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runGit(root, "commit", "--allow-empty", "-m", "seed"); err != nil {
		t.Fatalf("git commit seed: %v\n%s", err, out)
	}
	if out, err := runGit(root, "push", "-u", "origin", "HEAD:main"); err != nil {
		t.Fatalf("git push -u: %v\n%s", err, out)
	}
	return root
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
	// Resolve the two opener SHAs by walking git log.
	openersOut, err := runGit(root, "log", "--reverse", "-E",
		"--grep", "^aiwf-verb: authorize$",
		"--grep", "^aiwf-scope: opened$",
		"--grep", "^aiwf-entity: E-01$",
		"--all-match",
		"--pretty=tformat:%H")
	if err != nil {
		t.Fatalf("git log openers: %v\n%s", err, openersOut)
	}
	openers := strings.Fields(strings.TrimSpace(openersOut))
	if len(openers) != 2 {
		t.Fatalf("expected 2 openers; got %d (%v)", len(openers), openers)
	}
	endsSeen := map[string]bool{}
	for _, tr := range tr {
		if tr.Key == gitops.TrailerScopeEnds {
			endsSeen[tr.Value] = true
		}
	}
	if len(endsSeen) != 2 {
		t.Errorf("got %d distinct aiwf-scope-ends values, want 2; saw %v", len(endsSeen), endsSeen)
	}
	for _, sha := range openers {
		if !endsSeen[sha] {
			t.Errorf("scope-ends does not name opener %s; saw %v", sha, endsSeen)
		}
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
		{"add", "milestone", "--tdd", "none", "--epic", "E-01", "--title", "Cache"},
		{"add", "milestone", "--tdd", "none", "--epic", "E-02", "--title", "Sink"},
	} {
		if out, err := runBin(t, root, binDir, nil, args...); err != nil {
			t.Fatalf("setup %v: %v\n%s", args, err, out)
		}
	}
	// Open scope on E-01 and capture its SHA before any pivot.
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-01", "--to", "ai/claude"); err != nil {
		t.Fatalf("authorize E-01: %v\n%s", err, out)
	}
	e01AuthSHA := mustHeadSHA(t, root)
	// Agent acts under E-01 scope.
	if out, err := runBin(t, root, binDir, nil,
		"promote", "M-001", "in_progress",
		"--actor", "ai/claude", "--principal", "human/peter"); err != nil {
		t.Fatalf("agent on M-001 (under E-01): %v\n%s", err, out)
	}
	if got := authorizedByOf(t, root, "HEAD"); got != e01AuthSHA {
		t.Errorf("M-001 promote (under E-01) aiwf-authorized-by = %s, want %s", got, e01AuthSHA)
	}
	// Pause E-01, open E-02 (capture E-02's auth SHA before agent acts).
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-01", "--pause", "switching focus"); err != nil {
		t.Fatalf("pause E-01: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-02", "--to", "ai/claude"); err != nil {
		t.Fatalf("authorize E-02: %v\n%s", err, out)
	}
	e02AuthSHA := mustHeadSHA(t, root)
	// Agent acts under E-02 scope; the commit must reference E-02's SHA.
	if out, err := runBin(t, root, binDir, nil,
		"promote", "M-002", "in_progress",
		"--actor", "ai/claude", "--principal", "human/peter"); err != nil {
		t.Fatalf("agent on M-002 (under E-02): %v\n%s", err, out)
	}
	if got := authorizedByOf(t, root, "HEAD"); got != e02AuthSHA {
		t.Errorf("M-002 promote (under E-02) aiwf-authorized-by = %s, want %s", got, e02AuthSHA)
	}
	// Pause E-02, resume E-01.
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-02", "--pause", "back to E-01"); err != nil {
		t.Fatalf("pause E-02: %v\n%s", err, out)
	}
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-01", "--resume", "continuing E-01 work"); err != nil {
		t.Fatalf("resume E-01: %v\n%s", err, out)
	}
	// Cancel M-001 under the resumed E-01 scope; commit must reference E-01's SHA again.
	if out, err := runBin(t, root, binDir, nil,
		"cancel", "M-001",
		"--actor", "ai/claude", "--principal", "human/peter"); err != nil {
		t.Fatalf("cancel M-001 under resumed E-01: %v\n%s", err, out)
	}
	if got := authorizedByOf(t, root, "HEAD"); got != e01AuthSHA {
		t.Errorf("cancel M-001 (resumed E-01) aiwf-authorized-by = %s, want %s", got, e01AuthSHA)
	}
}

// mustHeadSHA returns the full SHA of HEAD in root, t.Fatal'ing on
// failure. Helper for tests that need to capture an authorize
// commit's SHA right after `aiwf authorize`.
func mustHeadSHA(t *testing.T, root string) string {
	t.Helper()
	out, err := runGit(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v\n%s", err, out)
	}
	return strings.TrimSpace(out)
}

// authorizedByOf returns the value of the aiwf-authorized-by:
// trailer on the commit named by `rev`, or "" when absent. Used to
// verify scoped commits reference the right authorize SHA.
func authorizedByOf(t *testing.T, root, rev string) string {
	t.Helper()
	out, err := runGit(root, "log", rev, "-1",
		"--pretty=tformat:%(trailers:key=aiwf-authorized-by,valueonly=true,unfold=true)")
	if err != nil {
		t.Fatalf("git log %s: %v\n%s", rev, err, out)
	}
	return strings.TrimSpace(out)
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
		{"add", "milestone", "--tdd", "none", "--epic", "E-02", "--title", "Cache"},
	} {
		if out, err := runBin(t, root, binDir, nil, args...); err != nil {
			t.Fatalf("setup %v: %v\n%s", args, err, out)
		}
	}
	// Open scope on E-02 (the soon-to-be-reallocated epic) and capture
	// the auth SHA so we can later verify the post-reallocate commit
	// continues to reference it.
	if out, err := runBin(t, root, binDir, nil, "authorize", "E-02", "--to", "ai/claude"); err != nil {
		t.Fatalf("authorize: %v\n%s", err, out)
	}
	authSHA := mustHeadSHA(t, root)
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
	// The post-reallocate agent commit's aiwf-authorized-by: must
	// still be the original scope's SHA — a successful verb here
	// could otherwise be a false pass (e.g., scope failed to load
	// and the commit landed without provenance trailers).
	if got := authorizedByOf(t, root, "HEAD"); got != authSHA {
		t.Errorf("post-reallocate agent commit aiwf-authorized-by = %s, want original scope SHA %s", got, authSHA)
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
	// `aiwf check` should now fire the
	// provenance-untrailered-entity-commit warning on the manual
	// commit — the audit-trail hole G24 surfaces.
	preCheck, _ := runBin(t, root, binDir, nil, "check")
	if !strings.Contains(preCheck, "provenance-untrailered-entity-commit") {
		t.Errorf("expected provenance-untrailered-entity-commit warning before audit-only; got:\n%s", preCheck)
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
	// After audit-only, the warning must clear — the plan's promise
	// for step 7b. RunUntrailedAudit suppresses manual commits whose
	// touched entities have a later audit-only commit covering them.
	postCheck, _ := runBin(t, root, binDir, nil, "check")
	if strings.Contains(postCheck, "provenance-untrailered-entity-commit") {
		t.Errorf("warning should clear after audit-only backfill; got:\n%s", postCheck)
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
	if env.Result.Scopes[0].EventCount != 5 {
		t.Errorf("event_count = %d, want exactly 5 (open + 4 transitions)", env.Result.Scopes[0].EventCount)
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
	subAuthSHA := mustHeadSHA(t, root)
	// The hand-crafted commit's trailers all sit on the legal side
	// of the I2.5 standing rules: ai/ actor with human/ principal
	// (required-together OK), human/ on-behalf-of, valid SHA,
	// authorized-by points at a real opener, scope=opened. Per the
	// design (G22 deferred), the kernel does not enforce a rule on
	// (authorize-verb, on-behalf-of); so NO provenance finding may
	// reference this commit. A future rule that fires on this
	// combination would surface here.
	out, _ := runBin(t, root, binDir, nil, "check")
	short := subAuthSHA[:7]
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, short) && strings.Contains(line, "provenance-") {
			t.Errorf("kernel fired a standing rule against the G22-deferred sub-authorize commit %s:\n%s", short, line)
		}
	}
}
