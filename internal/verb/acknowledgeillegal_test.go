package verb_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// TestAcknowledgeIllegal_CommitShape pins M-0136/AC-1: the verb
// produces an empty commit carrying the four required trailers
// (aiwf-verb, aiwf-force-for, aiwf-actor, aiwf-reason). RED today
// because the stub returns "not implemented"; GREEN once the verb's
// commit-shape logic lands.
func TestAcknowledgeIllegal_CommitShape(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	historicalSHA := commitOne(t, r.root, "alpha.md", "alpha v1\n", "historical illegal flip")

	const reason = "squash-merge from pre-AC-2 era; intermediate FSM steps lost to the squash"
	res, err := verb.AcknowledgeIllegal(r.ctx, r.root, historicalSHA, testActor, reason)
	if err != nil {
		t.Fatalf("AcknowledgeIllegal: %v", err)
	}
	if res == nil || res.Plan == nil {
		t.Fatalf("nil result or plan: %+v", res)
	}
	if !res.Plan.AllowEmpty {
		t.Errorf("AllowEmpty = false, want true (acknowledge-illegal commits are empty)")
	}
	if len(res.Plan.Ops) != 0 {
		t.Errorf("Ops len = %d, want 0", len(res.Plan.Ops))
	}

	mustHaveTrailerInPlanList(t, res.Plan.Trailers, gitops.TrailerVerb, "acknowledge-illegal")
	mustHaveTrailerInPlanList(t, res.Plan.Trailers, gitops.TrailerForceFor, historicalSHA)
	mustHaveTrailerInPlanList(t, res.Plan.Trailers, gitops.TrailerActor, testActor)
	mustHaveTrailerInPlanList(t, res.Plan.Trailers, gitops.TrailerReason, reason)
}

// TestAcknowledgeIllegal_RequiresReason pins M-0136/AC-1's mandatory-
// reason gate: an empty or whitespace-only reason is rejected with a
// typed error mentioning the flag.
func TestAcknowledgeIllegal_RequiresReason(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	historicalSHA := commitOne(t, r.root, "alpha.md", "alpha v1\n", "historical illegal flip")

	cases := []struct{ name, reason string }{
		{"empty", ""},
		{"whitespace only", "   "},
		{"tabs and newlines", "\t \n"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := verb.AcknowledgeIllegal(r.ctx, r.root, historicalSHA, testActor, c.reason)
			if err == nil || !strings.Contains(err.Error(), "reason") {
				t.Errorf("expected error mentioning --reason for reason=%q; got %v", c.reason, err)
			}
		})
	}
}

// TestAcknowledgeIllegal_RequiresHumanActor pins M-0136/AC-1's
// sovereign-actor gate: only `human/...` actors can acknowledge
// historical illegal commits. Non-human actors (ai/, bot/) are
// rejected.
func TestAcknowledgeIllegal_RequiresHumanActor(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	historicalSHA := commitOne(t, r.root, "alpha.md", "alpha v1\n", "historical illegal flip")

	cases := []struct{ name, actor string }{
		{"ai actor", "ai/claude"},
		{"bot actor", "bot/dependabot"},
		{"empty actor", ""},
		{"malformed actor", "claude"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := verb.AcknowledgeIllegal(r.ctx, r.root, historicalSHA, c.actor, "squash-merge fixup")
			if err == nil || !strings.Contains(err.Error(), "human/") {
				t.Errorf("expected error mentioning human/ requirement for actor=%q; got %v", c.actor, err)
			}
		})
	}
}

// commitOne writes a file and creates one commit, returning the new
// commit's SHA. Used by acknowledge-illegal tests that need a
// historical commit to point aiwf-force-for at.
func commitOne(t *testing.T, root, path, content, subject string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, path), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := gitops.Add(context.Background(), root, path); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := gitops.Commit(context.Background(), root, subject, "", nil); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	sha, err := gitops.HeadSubject(context.Background(), root) // sanity; not strictly needed
	if err != nil || sha == "" {
		t.Fatalf("HEAD subject: %v", err)
	}
	// Fetch the SHA via rev-parse HEAD.
	return resolveHeadSHA(t, root)
}

func resolveHeadSHA(t *testing.T, root string) string {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", "rev-parse", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func mustHaveTrailerInPlanList(t *testing.T, trailers []gitops.Trailer, key, value string) {
	t.Helper()
	for _, tr := range trailers {
		if tr.Key == key && tr.Value == value {
			return
		}
	}
	t.Errorf("trailer %s=%q not present in plan; got %+v", key, value, trailers)
}

// TestAcknowledgeIllegal_AC4_RejectsOutOfHistorySHA pins M-0136/AC-4:
// a SHA that doesn't resolve to a commit reachable from HEAD is
// rejected with a typed error mentioning "not reachable". Prevents
// silent accumulation of no-op acknowledgments (typos, copy-paste
// errors, SHAs from orphaned branches).
//
// RED today: the verb only validates the SHA's hex shape, not its
// reachability. Any valid-shape SHA passes; the test asserts an
// error mentioning "reachable" and FAILS because the verb succeeds
// in producing a plan.
func TestAcknowledgeIllegal_AC4_RejectsOutOfHistorySHA(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	// Need at least one commit so HEAD exists for the reachability
	// check (otherwise "no commits" is the failure mode, not
	// "not reachable").
	commitOne(t, r.root, "alpha.md", "alpha v1\n", "real commit")

	// 40-hex SHA with the right shape but not in HEAD's history.
	const bogusSHA = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

	_, err := verb.AcknowledgeIllegal(r.ctx, r.root, bogusSHA, testActor, "typo in the SHA")
	if err == nil {
		t.Fatal("expected error for out-of-history SHA; got nil")
	}
	// G-0236: the error now mentions both checks (not reachable AND
	// not in object database) since the fallback path is documented
	// alongside the primary refusal. Preserves the "reachable"
	// substring expectation from the M-0136/AC-4 era.
	if !strings.Contains(err.Error(), "reachable") {
		t.Errorf("expected error mentioning reachability for SHA %s; got %v", bogusSHA, err)
	}
	if !strings.Contains(err.Error(), "object database") {
		t.Errorf("G-0236: error should reference the object-database fallback so the operator sees both refused paths; got %v", err)
	}
}

// TestAcknowledgeIllegal_G0236_AcceptsOrphanSHA pins the G-0236
// reflog-fallback acceptance path: a SHA that is NOT reachable from
// HEAD but IS present in the local object database (the canonical
// orphan shape — a commit force-pushed away from its ref but still
// in the object DB via reflog reference) is accepted.
//
// Fixture shape: create commit A, create commit B, `git reset --hard A`
// so HEAD = A and B is orphan-in-object-DB. Ack against B's SHA must
// succeed and produce a plan; the M-0136 docstring contract (the four
// trailers + AllowEmpty) is preserved.
//
// Closes G-0236.
func TestAcknowledgeIllegal_G0236_AcceptsOrphanSHA(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	// Commit A — stays reachable from HEAD.
	commitOne(t, r.root, "alpha.md", "alpha v1\n", "alpha")
	headA := resolveHeadSHA(t, r.root)
	// Commit B — will be orphaned.
	commitOne(t, r.root, "beta.md", "beta v1\n", "beta")
	headB := resolveHeadSHA(t, r.root)
	if headA == headB {
		t.Fatalf("setup error: A and B resolve to the same SHA (%s)", headA)
	}
	// `git reset --hard A` rewinds HEAD; B's SHA stays in the object
	// DB (reachable via reflog) but is no longer HEAD-reachable.
	gitReset := exec.CommandContext(r.ctx, "git", "reset", "--hard", headA)
	gitReset.Dir = r.root
	if out, err := gitReset.CombinedOutput(); err != nil {
		t.Fatalf("git reset --hard A: %v\n%s", err, out)
	}
	// Sanity: HEAD is now A; B is in object DB.
	if got := resolveHeadSHA(t, r.root); got != headA {
		t.Fatalf("post-reset HEAD = %s, want %s (A)", got, headA)
	}
	revparse := exec.CommandContext(r.ctx, "git", "rev-parse", "--verify", headB+"^{commit}")
	revparse.Dir = r.root
	if err := revparse.Run(); err != nil {
		t.Fatalf("setup: B should still be in object DB post-reset; rev-parse: %v", err)
	}

	const reason = "rebase cleanup of an AI-actor commit; the content re-landed cleanly under a different SHA"
	res, err := verb.AcknowledgeIllegal(r.ctx, r.root, headB, testActor, reason)
	if err != nil {
		t.Fatalf("G-0236 fallback: ack against orphan SHA must succeed; got %v", err)
	}
	if res == nil || res.Plan == nil {
		t.Fatalf("nil result or plan: %+v", res)
	}
	// Same four-trailer + empty-commit contract M-0136/AC-1 pins for
	// the reachable case — the fallback path must not weaken the
	// shape contract.
	mustHaveTrailerInPlanList(t, res.Plan.Trailers, gitops.TrailerVerb, "acknowledge-illegal")
	mustHaveTrailerInPlanList(t, res.Plan.Trailers, gitops.TrailerForceFor, headB)
	mustHaveTrailerInPlanList(t, res.Plan.Trailers, gitops.TrailerActor, testActor)
	mustHaveTrailerInPlanList(t, res.Plan.Trailers, gitops.TrailerReason, reason)
	if !res.Plan.AllowEmpty {
		t.Errorf("AllowEmpty = false, want true (ack commits are always empty regardless of fallback path)")
	}
}
