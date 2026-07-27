package verb_test

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestCancel_AlreadyAtCancelTerminal_ReturnsNoOp pins M-0281/AC-2 for the
// idempotent-re-run case: cancelling an entity that is already at the terminal
// status cancel produces (`cancelled` for an epic) converges to a NoOp instead
// of the "already at terminal status" error.
func TestCancel_AlreadyAtCancelTerminal_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Cancel(r.ctx, r.tree(), "E-0001", testActor, "", false)) // proposed -> cancelled

	res, err := verb.Cancel(r.ctx, r.tree(), "E-0001", testActor, "", false)
	if err != nil {
		t.Fatalf("cancel of an already-cancelled epic returned a Go error, want a NoOp: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true (re-cancelling a terminal entity is a no-op)")
	}
	if res.Plan != nil {
		t.Errorf("res.Plan = %+v, want nil (a NoOp produces no commit)", res.Plan)
	}
}

// TestCancel_AlreadyAtSuccessTerminal_ReturnsNoOp pins the Option-A breadth of
// M-0281/AC-2 (ADR-0036): cancelling an entity at a *success* terminal — one
// cancel never produces, here a gap forced to `addressed` — is also a NoOp, not
// an error. A terminal entity is already disposed, so cancel has nothing to do
// regardless of which terminal it reached.
func TestCancel_AlreadyAtSuccessTerminal_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Stray gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	// --force reaches `addressed` (a success terminal, not cancel's `wontfix`)
	// without a resolver.
	r.must(verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "fixture", true, verb.PromoteOptions{}))

	res, err := verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "", false)
	if err != nil {
		t.Fatalf("cancel of an addressed (success-terminal) gap returned a Go error, want a NoOp: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true (cancel of any terminal entity is a no-op)")
	}
	if res.Plan != nil {
		t.Errorf("res.Plan = %+v, want nil (a NoOp produces no commit)", res.Plan)
	}
}
