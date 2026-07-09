package verb_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestPromote_EpicToDoneWithNonTerminalChildMilestone_Refuses pins
// G-0393: `aiwf promote <epic> done` must refuse the same way `aiwf
// cancel <epic>` already does when the epic still owns a non-terminal
// child milestone — otherwise a subsequent `aiwf archive --apply`
// sweeps the still-in_progress milestone alongside its now-terminal
// parent, a state `aiwf check`'s archived-entity-not-terminal rule
// only catches after the fact.
func TestPromote_EpicToDoneWithNonTerminalChildMilestone_Refuses(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Doomed", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	// A fresh milestone enters at `draft` — non-terminal.
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Child", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

	res, err := verb.Promote(r.ctx, r.tree(), "E-0001", "done", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatalf("Promote(E-0001, done) succeeded (res=%+v); want refusal because M-0001 is non-terminal", res)
	}
	code, ok := entity.Code(err)
	if !ok || code != verb.CodeEpicPromoteNonTerminalChildren.ID {
		t.Fatalf("Code(err) = (%q, %v); want (%q, true)", code, ok, verb.CodeEpicPromoteNonTerminalChildren.ID)
	}
	if !strings.Contains(err.Error(), "M-0001") {
		t.Errorf("error message %q does not list offending milestone M-0001", err.Error())
	}
}

// TestPromote_EpicToCancelledWithNonTerminalChildMilestone_Refuses
// pins the same guard against the OTHER terminal status a bare
// `aiwf promote` can reach directly (the epic FSM legally allows
// active -> cancelled via Promote, not just via the dedicated Cancel
// verb) — without this, `aiwf promote <epic> cancelled` would bypass
// EpicCancelNonTerminalChildrenError's own refusal entirely.
func TestPromote_EpicToCancelledWithNonTerminalChildMilestone_Refuses(t *testing.T) {
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

// TestPromote_EpicToDoneWithAllChildrenTerminal_Succeeds is the
// characterization control: the new guard must not refuse a legal
// promote when every child milestone is already terminal.
func TestPromote_EpicToDoneWithAllChildrenTerminal_Succeeds(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Doomed", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Child", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Cancel(r.ctx, r.tree(), "M-0001", testActor, "", false))

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
// cancelled) is in scope for G-0393's guard. Without this test, a
// mutation that dropped the guard's terminal-status condition
// entirely survived every other test in this package — only caught,
// indirectly, by unrelated cross-verb smoke fixtures elsewhere in the
// repo that happen to promote an epic with a milestone to active.
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

// TestPromote_EpicWithNoChildMilestones_Succeeds is the second
// characterization control: an epic with no milestones at all (the
// common case) must still promote to done cleanly.
func TestPromote_EpicWithNoChildMilestones_Succeeds(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Solo", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))

	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "done", testActor, "", false, verb.PromoteOptions{}))
	if e := r.tree().ByID("E-0001"); e == nil || e.Status != "done" {
		t.Errorf("E-0001 = %+v; want status done", e)
	}
}
