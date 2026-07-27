package verb_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

const ackReason = "squash-merge from a pre-audit era; intermediate FSM steps lost to the squash"

// countCommits returns the number of commits reachable from HEAD.
func countCommits(t *testing.T, root string) string {
	t.Helper()
	out, err := runGit(t.Context(), root, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("rev-list --count HEAD: %v", err)
	}
	return strings.TrimSpace(out)
}

// TestAcknowledgeIllegal_AlreadyAcknowledged_ReturnsNoOp pins M-0281/AC-4 —
// the correctness half of this milestone. Re-running `acknowledge illegal`
// against a SHA that HEAD's history already acknowledges appended a *duplicate*
// empty audit commit every time; it must now return a NoOp and write nothing.
// The commit count is asserted before and after, so a regression that re-lands
// the duplicate commit fails even if the Result shape looks right.
func TestAcknowledgeIllegal_AlreadyAcknowledged_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	historicalSHA := commitOne(t, r.root, "alpha.md", "alpha v1\n", "historical illegal flip")

	// First ack: lands a real empty audit commit.
	res, err := verb.AcknowledgeIllegal(r.ctx, r.root, historicalSHA, "", testActor, ackReason)
	if err != nil {
		t.Fatalf("first AcknowledgeIllegal: %v", err)
	}
	if res.Plan == nil {
		t.Fatal("first ack produced no plan")
	}
	if _, applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("applying the first ack: %v", applyErr)
	}

	countAfterFirst := countCommits(t, r.root)

	// Second ack against the same SHA: nothing left to record.
	again, err := verb.AcknowledgeIllegal(r.ctx, r.root, historicalSHA, "", testActor, ackReason)
	if err != nil {
		t.Fatalf("re-acking an already-acknowledged SHA returned a Go error, want a NoOp: %v", err)
	}
	if !again.NoOp {
		t.Errorf("again.NoOp = false, want true (the SHA is already acknowledged)")
	}
	if again.Plan != nil {
		t.Errorf("again.Plan = %+v, want nil — a second ack must not append a duplicate empty audit commit", again.Plan)
	}
	if got := countCommits(t, r.root); got != countAfterFirst {
		t.Errorf("commit count = %s, want %s (the NoOp must append no commit)", got, countAfterFirst)
	}
}

// TestAcknowledgeIllegal_OrphanSHAWithUnbornHEAD_StillRecords pins the
// fail-open edge of M-0281/AC-4's duplicate guard: when HEAD carries no
// commits at all, no acknowledgment can have been recorded, so the verb must
// still produce a plan. The fixture is the extreme of the reflog-only orphan
// case the verb deliberately supports — a commit that exists in the object
// database while HEAD is unborn (its ref deleted). Failing open here degrades
// to always-record; failing closed would silently skip a needed ack.
func TestAcknowledgeIllegal_OrphanSHAWithUnbornHEAD_StillRecords(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	orphanSHA := commitOne(t, r.root, "alpha.md", "alpha v1\n", "soon-to-be-orphaned commit")
	// Delete HEAD's ref: the commit object survives, HEAD becomes unborn.
	if _, err := runGit(r.ctx, r.root, "update-ref", "-d", "HEAD"); err != nil {
		t.Fatalf("update-ref -d HEAD: %v", err)
	}

	res, err := verb.AcknowledgeIllegal(r.ctx, r.root, orphanSHA, "", testActor, ackReason)
	if err != nil {
		t.Fatalf("acking an orphan SHA against an unborn HEAD: %v", err)
	}
	if res.NoOp {
		t.Errorf("res.NoOp = true, want false — an unborn HEAD carries no acknowledgments, so there is one to record")
	}
	if res.Plan == nil {
		t.Error("res.Plan = nil, want a plan")
	}
}

// TestAcknowledgeIllegal_EntityBoundAckIsIndependentOfBlanketAck guards the
// per-(SHA, entity) shape: a blanket per-SHA ack does not suppress the
// provenance rule's per-(commit, entity) findings, so an entity-bound ack for
// the same SHA is NOT a duplicate — it must still produce a plan. Only a
// matching (SHA, entity) pair converges to a NoOp.
func TestAcknowledgeIllegal_EntityBoundAckIsIndependentOfBlanketAck(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	// A real entity commit, so the SHA's diff touches the entity the
	// entity-bound ack names (the verb verifies that binding).
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Stray gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	entitySHA := resolveHeadSHA(t, r.root)

	// Blanket per-SHA ack lands first.
	blanket, err := verb.AcknowledgeIllegal(r.ctx, r.root, entitySHA, "", testActor, ackReason)
	if err != nil {
		t.Fatalf("blanket ack: %v", err)
	}
	if _, applyErr := verb.Apply(r.ctx, r.root, blanket.Plan); applyErr != nil {
		t.Fatalf("applying the blanket ack: %v", applyErr)
	}

	// An entity-bound ack for the same SHA is a different ack shape — it must
	// still produce a plan, not converge to a NoOp.
	bound, err := verb.AcknowledgeIllegal(r.ctx, r.root, entitySHA, "G-0001", testActor, ackReason)
	if err != nil {
		t.Fatalf("entity-bound ack after a blanket ack: %v", err)
	}
	if bound.NoOp {
		t.Errorf("bound.NoOp = true, want false (a blanket ack does not cover the per-(SHA, entity) shape)")
	}
	if bound.Plan == nil {
		t.Fatal("bound.Plan = nil, want a plan (the entity-bound ack still has something to record)")
	}
	if _, applyErr := verb.Apply(r.ctx, r.root, bound.Plan); applyErr != nil {
		t.Fatalf("applying the entity-bound ack: %v", applyErr)
	}

	// Re-running the SAME entity-bound ack is now the duplicate case.
	countBefore := countCommits(t, r.root)
	dup, err := verb.AcknowledgeIllegal(r.ctx, r.root, entitySHA, "G-0001", testActor, ackReason)
	if err != nil {
		t.Fatalf("re-running the entity-bound ack: %v", err)
	}
	if !dup.NoOp {
		t.Errorf("dup.NoOp = false, want true (this (SHA, entity) pair is already acknowledged)")
	}
	if got := countCommits(t, r.root); got != countBefore {
		t.Errorf("commit count = %s, want %s (the NoOp must append no commit)", got, countBefore)
	}
}
