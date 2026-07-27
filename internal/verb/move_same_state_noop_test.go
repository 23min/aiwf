package verb_test

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestMove_ToCurrentParent_ReturnsNoOp pins M-0281/AC-3: moving a milestone to
// the epic it is already under converges to a NoOp instead of the "already
// under epic" error, so a re-run (interactive or scripted) is a clean exit 0.
func TestMove_ToCurrentParent_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

	res, err := verb.Move(r.ctx, r.tree(), "M-0001", "E-0001", testActor)
	if err != nil {
		t.Fatalf("move to the current parent returned a Go error, want a NoOp: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true (move to the current parent is a no-op)")
	}
	if res.Plan != nil {
		t.Errorf("res.Plan = %+v, want nil (a NoOp produces no commit)", res.Plan)
	}
}
