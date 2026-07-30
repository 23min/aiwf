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

// ackFixtureWithAC builds an epic + milestone + AC and returns the root and the
// SHA of the add-ac commit. That commit's diff touches the milestone file, so a
// composite `--for-entity M-0001/AC-1` clears the verb's touches-the-entity
// verification (which compares at the rolled-up parent id).
func ackFixtureWithAC(t *testing.T) (r *runner, acSHA string) {
	t.Helper()
	r = newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "cache warms on boot", testActor))
	return r, resolveHeadSHA(t, r.root)
}

// TestAcknowledgeIllegal_CompositeForEntity_ConvergesOnRepeat closes the
// composite half of M-0281/AC-4. The verb emits `aiwf-entity` at full composite
// width, and check's walker stores that value as given, so a duplicate guard
// that looks up only the rolled-up parent id never matches its own writes —
// every re-run of an AC-scoped ack appended another indistinguishable audit
// commit, the exact history growth AC-4 exists to stop. The bare-id tests above
// cannot catch this: for them the composite and rolled-up spellings coincide.
func TestAcknowledgeIllegal_CompositeForEntity_ConvergesOnRepeat(t *testing.T) {
	t.Parallel()
	r, acSHA := ackFixtureWithAC(t)

	first, err := verb.AcknowledgeIllegal(r.ctx, r.root, acSHA, "M-0001/AC-1", testActor, ackReason)
	if err != nil {
		t.Fatalf("first composite ack: %v", err)
	}
	if first.Plan == nil {
		t.Fatal("first composite ack produced no plan")
	}
	if _, applyErr := verb.Apply(r.ctx, r.root, first.Plan); applyErr != nil {
		t.Fatalf("applying the first composite ack: %v", applyErr)
	}
	countAfterFirst := countCommits(t, r.root)

	again, err := verb.AcknowledgeIllegal(r.ctx, r.root, acSHA, "M-0001/AC-1", testActor, ackReason)
	if err != nil {
		t.Fatalf("re-running the composite ack returned a Go error, want a NoOp: %v", err)
	}
	if !again.NoOp {
		t.Errorf("again.NoOp = false, want true — this (SHA, composite entity) pair is already acknowledged")
	}
	if again.Plan != nil {
		t.Errorf("again.Plan = %+v, want nil — the repeat must not append a duplicate audit commit", again.Plan)
	}
	if !strings.Contains(again.NoOpMessage, "M-0001/AC-1") {
		t.Errorf("NoOpMessage = %q, want it to name the composite binding that was found", again.NoOpMessage)
	}
	if got := countCommits(t, r.root); got != countAfterFirst {
		t.Errorf("commit count = %s, want %s (the NoOp must append no commit)", got, countAfterFirst)
	}
}

// TestAcknowledgeIllegal_CompositeAfterParentAck_NamesTheBindingItFound pins
// the message against overclaiming. A parent-scoped ack already suppresses
// every finding an AC-scoped one would, because the consuming rule rolls a
// touched id up to its parent before looking the ack up — so a composite
// request afterwards has nothing left to record and converges. What it must not
// do is report convergence against the composite binding: no ack names
// `M-0001/AC-1`, and an operator told otherwise would believe a per-AC record
// exists in the history when only the parent one does.
func TestAcknowledgeIllegal_CompositeAfterParentAck_NamesTheBindingItFound(t *testing.T) {
	t.Parallel()
	r, acSHA := ackFixtureWithAC(t)

	parent, err := verb.AcknowledgeIllegal(r.ctx, r.root, acSHA, "M-0001", testActor, ackReason)
	if err != nil {
		t.Fatalf("parent-scoped ack: %v", err)
	}
	if _, applyErr := verb.Apply(r.ctx, r.root, parent.Plan); applyErr != nil {
		t.Fatalf("applying the parent-scoped ack: %v", applyErr)
	}

	composite, err := verb.AcknowledgeIllegal(r.ctx, r.root, acSHA, "M-0001/AC-1", testActor, ackReason)
	if err != nil {
		t.Fatalf("composite ack after a parent-scoped ack: %v", err)
	}
	if !composite.NoOp {
		t.Fatalf("composite.NoOp = false, want true — the parent ack already covers what this would record")
	}
	if strings.Contains(composite.NoOpMessage, "AC-1") {
		t.Errorf("NoOpMessage = %q, must not claim an acknowledgment for M-0001/AC-1 — the recorded ack names M-0001", composite.NoOpMessage)
	}
	if !strings.Contains(composite.NoOpMessage, "M-0001") {
		t.Errorf("NoOpMessage = %q, want it to name M-0001, the binding actually found", composite.NoOpMessage)
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

// TestAcknowledgeIllegal_AbbreviatedSHA_ReturnsNoOp pins the arm that makes
// AC-4's convergence usable in practice. `aiwf history` prints the 7-char form,
// so the abbreviated spelling is the one an operator copies back in — the same
// argument promote's resolver comparison makes for itself.
//
// The guard resolves both sides through gitops.ResolveCommitSHA before
// comparing, so a short spelling and the full SHA it names are one referent.
// Comparing the raw strings instead leaves this arm appending a duplicate empty
// audit commit on every repeat, which is the defect AC-4 exists to close — and
// resolving is invisible to a test that only ever spells the SHA in full.
func TestAcknowledgeIllegal_AbbreviatedSHA_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	fullSHA := commitOne(t, r.root, "beta.md", "beta v1\n", "historical illegal flip")

	res, err := verb.AcknowledgeIllegal(r.ctx, r.root, fullSHA, "", testActor, ackReason)
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

	// The abbreviated form names the very commit just acknowledged.
	short := fullSHA[:7]
	again, err := verb.AcknowledgeIllegal(r.ctx, r.root, short, "", testActor, ackReason)
	if err != nil {
		t.Fatalf("re-acking at the abbreviated spelling returned a Go error, want a NoOp: %v", err)
	}
	if !again.NoOp {
		t.Errorf("again.NoOp = false, want true — %s is %s", short, fullSHA)
	}
	if again.Plan != nil {
		t.Errorf("again.Plan = %+v, want nil — an abbreviated re-ack must not append a duplicate", again.Plan)
	}
	if got := countCommits(t, r.root); got != countAfterFirst {
		t.Errorf("commit count = %s, want %s (the NoOp must append no commit)", got, countAfterFirst)
	}
}
