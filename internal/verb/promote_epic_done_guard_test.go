package verb_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestPromote_EpicDoneWithNonTerminalChildMilestone_Refuses (G-0394,
// Direction A): promoting an epic to done while it still owns a
// non-terminal (draft) child milestone must refuse with the
// structured CodeEpicPromoteNonTerminalChildren code, listing the
// offending milestone id, and produce no Result.
func TestPromote_EpicDoneWithNonTerminalChildMilestone_Refuses(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Doomed", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Child", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))

	res, err := verb.Promote(r.ctx, r.tree(), "E-0001", "done", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatalf("Promote(E-0001, done) succeeded (res=%+v); want refusal because M-0001 is non-terminal (draft)", res)
	}
	code, ok := entity.Code(err)
	if !ok || code != verb.CodeEpicPromoteNonTerminalChildren.ID {
		t.Fatalf("Code(err) = (%q, %v); want (%q, true)", code, ok, verb.CodeEpicPromoteNonTerminalChildren.ID)
	}
	if !strings.Contains(err.Error(), "M-0001") {
		t.Errorf("error message %q does not list offending milestone M-0001", err.Error())
	}
}

// TestPromote_EpicDoneWithAllTerminalChildren_Succeeds (G-0394):
// characterization against a "refuse everything" regression — an epic
// whose only child milestone has already reached done promotes to
// done cleanly.
func TestPromote_EpicDoneWithAllTerminalChildren_Succeeds(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Child", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "done", testActor, "", false, verb.PromoteOptions{}))

	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "done", testActor, "", false, verb.PromoteOptions{}))
	if e := r.tree().ByID("E-0001"); e == nil || e.Status != "done" {
		t.Errorf("E-0001 = %+v; want status done", e)
	}
}

// TestPromote_EpicDoneNoChildren_Succeeds (G-0394): an epic with zero
// milestones (nothing to enumerate) promotes to done cleanly — the
// guard's empty-slice case must not misfire.
func TestPromote_EpicDoneNoChildren_Succeeds(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Solo", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))

	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "done", testActor, "", false, verb.PromoteOptions{}))
	if e := r.tree().ByID("E-0001"); e == nil || e.Status != "done" {
		t.Errorf("E-0001 = %+v; want status done", e)
	}
}

// TestPromote_EpicDoneForce_BypassesNonTerminalChildrenGuard (G-0394):
// force bypasses the promote-time guard exactly like it bypasses the
// other checks in Promote's `if !force` block — this is the specific
// case Archive's independent subtree-terminality guard exists to
// catch (Direction B).
func TestPromote_EpicDoneForce_BypassesNonTerminalChildrenGuard(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Doomed", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Child", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))

	res, err := verb.Promote(r.ctx, r.tree(), "E-0001", "done", testActor, "forcing through for the test", true, verb.PromoteOptions{})
	if err != nil {
		t.Fatalf("Promote(E-0001, done, force=true) refused: %v; want force to bypass the guard", err)
	}
	if res.Plan == nil {
		t.Fatal("force=true produced no plan")
	}
}
