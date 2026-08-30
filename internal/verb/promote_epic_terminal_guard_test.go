package verb_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestPromote_EpicDoneWithNonTerminalChildMilestone_Refuses (G-0393 /
// G-0394): promoting an epic to done while it still owns a
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

// TestPromote_EpicCancelledWithNonTerminalChildMilestone_Refuses pins
// the same guard against the OTHER terminal status a bare `aiwf
// promote` can reach directly (the epic FSM legally allows active ->
// cancelled via Promote, not just via the dedicated Cancel verb) —
// without this, `aiwf promote <epic> cancelled` would bypass
// EpicCancelNonTerminalChildrenError's own refusal entirely.
func TestPromote_EpicCancelledWithNonTerminalChildMilestone_Refuses(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Doomed", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Child", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

	res, err := verb.Promote(r.ctx, r.tree(), "E-0001", "cancelled", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatalf("Promote(E-0001, cancelled) succeeded (res=%+v); want refusal because M-0001 is non-terminal", res)
	}
	code, ok := entity.Code(err)
	if !ok || code != verb.CodeEpicPromoteNonTerminalChildren.ID {
		t.Fatalf("Code(err) = (%q, %v); want (%q, true)", code, ok, verb.CodeEpicPromoteNonTerminalChildren.ID)
	}
}

// TestPromote_EpicDoneWithAllTerminalChildren_Succeeds:
// characterization against a "refuse everything" regression — an epic
// whose only child milestone has already reached done promotes to
// done cleanly.
func TestPromote_EpicDoneWithAllTerminalChildren_Succeeds(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Child", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	// G-0269's activating-promote branch guard is out of scope for
	// this fixture (pure FSM scaffolding, not a branch-discipline
	// test) — force past it.
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "fixture-setup", true, verb.PromoteOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "done", testActor, "", false, verb.PromoteOptions{}))

	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "done", testActor, "", false, verb.PromoteOptions{}))
	if e := r.tree().ByID("E-0001"); e == nil || e.Status != "done" {
		t.Errorf("E-0001 = %+v; want status done", e)
	}
}

// TestPromote_EpicDoneNoChildren_Succeeds: an epic with zero
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

// TestPromote_EpicToActiveWithNonTerminalChildMilestone_Succeeds pins
// the guard's own terminal-target scope directly: "active" is the
// only non-terminal status a bare Promote can ever reach for an epic
// (proposed -> active), so a non-terminal child must NOT block this
// transition — only a promote reaching a terminal status (done or
// cancelled) is in scope for the guard. Without this test, a mutation
// that dropped the guard's terminal-status condition entirely survived
// every other test in this package — only caught, indirectly, by
// unrelated cross-verb smoke fixtures elsewhere in the repo that
// happen to promote an epic with a milestone to active.
func TestPromote_EpicToActiveWithNonTerminalChildMilestone_Succeeds(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Doomed", testActor, verb.AddOptions{}))
	// A milestone may be added under a still-proposed epic.
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Child", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	if e := r.tree().ByID("E-0001"); e == nil || e.Status != "active" {
		t.Errorf("E-0001 = %+v; want status active", e)
	}
}

// TestPromote_EpicDoneForce_DoesNotBypassNonTerminalChildrenGuard:
// unlike the other checks in Promote's `if !force` block, this guard
// runs unconditionally — matching Cancel's own EpicCancelNonTerminal-
// ChildrenError (D-0003), which has no force-bypass either. force
// relaxes FSM-transition legality and sovereign-act requirements, not
// this structural children precondition; Archive's independent
// subtree-terminality guard is the defense-in-depth backstop for the
// state a raw frontmatter hand-edit (bypassing this verb entirely)
// can still produce.
func TestPromote_EpicDoneForce_DoesNotBypassNonTerminalChildrenGuard(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Doomed", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Child", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))

	res, err := verb.Promote(r.ctx, r.tree(), "E-0001", "done", testActor, "forcing through for the test", true, verb.PromoteOptions{})
	if err == nil {
		t.Fatalf("Promote(E-0001, done, force=true) succeeded (res=%+v); want the guard to still refuse under force", res)
	}
	code, ok := entity.Code(err)
	if !ok || code != verb.CodeEpicPromoteNonTerminalChildren.ID {
		t.Fatalf("Code(err) = (%q, %v); want (%q, true)", code, ok, verb.CodeEpicPromoteNonTerminalChildren.ID)
	}
}
