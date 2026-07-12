package verb_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestPromote_MilestoneCancelledWithOpenAC_Refuses (G-0335): promoting
// a milestone straight to `cancelled` while it still carries an open
// AC must refuse with the structured CodeMilestonePromoteNonTerminalACs
// code, listing the offending composite AC id, and produce no Result —
// closing the gap where `aiwf promote <M> cancelled` bypassed the same
// guard `aiwf cancel <M>` already enforced.
func TestPromote_MilestoneCancelledWithOpenAC_Refuses(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Work", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First criterion", testActor, nil))

	res, err := verb.Promote(r.ctx, r.tree(), "M-0001", "cancelled", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatalf("Promote(M-0001, cancelled) succeeded (res=%+v); want refusal because AC-1 is open", res)
	}
	code, ok := entity.Code(err)
	if !ok || code != verb.CodeMilestonePromoteNonTerminalACs.ID {
		t.Fatalf("Code(err) = (%q, %v); want (%q, true)", code, ok, verb.CodeMilestonePromoteNonTerminalACs.ID)
	}
	if !strings.Contains(err.Error(), "M-0001/AC-1") {
		t.Errorf("error message %q does not list offending composite id M-0001/AC-1", err.Error())
	}
}

// TestPromote_MilestoneCancelledWithMultipleOpenACs_ListsAll pins the
// list-rendering branch TestPromote_MilestoneCancelledWithOpenAC_Refuses's
// single-AC fixture never exercises: with two open ACs (one cancelled
// in between, to confirm only the still-open ones are listed), the
// error message names both offending composite ids and the count.
func TestPromote_MilestoneCancelledWithMultipleOpenACs_ListsAll(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Work", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First criterion", testActor, nil))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "Second criterion", testActor, nil))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "Third criterion", testActor, nil))
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001/AC-2", "cancelled", testActor, "", false, verb.PromoteOptions{}))

	res, err := verb.Promote(r.ctx, r.tree(), "M-0001", "cancelled", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatalf("Promote(M-0001, cancelled) succeeded (res=%+v); want refusal because AC-1 and AC-3 are open", res)
	}
	if !strings.Contains(err.Error(), "2 open acceptance criterion(s)") {
		t.Errorf("error message %q does not report count 2", err.Error())
	}
	if !strings.Contains(err.Error(), "M-0001/AC-1") || !strings.Contains(err.Error(), "M-0001/AC-3") {
		t.Errorf("error message %q does not list both offending composite ids M-0001/AC-1 and M-0001/AC-3", err.Error())
	}
	if strings.Contains(err.Error(), "M-0001/AC-2") {
		t.Errorf("error message %q lists AC-2, which is already cancelled and should not appear", err.Error())
	}
}

// TestPromote_MilestoneCancelledWithNoOpenACs_Succeeds:
// characterization against a "refuse everything" regression — a
// milestone with no ACs (and one with only a cancelled AC) promotes to
// cancelled cleanly.
func TestPromote_MilestoneCancelledWithNoOpenACs_Succeeds(t *testing.T) {
	t.Parallel()

	t.Run("no-acs", func(t *testing.T) {
		t.Parallel()
		r := newRunner(t)
		r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
		r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Work", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

		r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "cancelled", testActor, "", false, verb.PromoteOptions{}))
		if e := r.tree().ByID("M-0001"); e == nil || e.Status != "cancelled" {
			t.Errorf("M-0001 = %+v; want status cancelled", e)
		}
	})

	t.Run("ac-already-cancelled", func(t *testing.T) {
		t.Parallel()
		r := newRunner(t)
		r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
		r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Work", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
		r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First criterion", testActor, nil))
		r.must(verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", "cancelled", testActor, "", false, verb.PromoteOptions{}))

		r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "cancelled", testActor, "", false, verb.PromoteOptions{}))
		if e := r.tree().ByID("M-0001"); e == nil || e.Status != "cancelled" {
			t.Errorf("M-0001 = %+v; want status cancelled", e)
		}
	})
}

// TestPromote_MilestoneCancelledForce_DoesNotBypassOpenACGuard: like
// the epic guard (G-0393 / G-0394), this guard runs unconditionally —
// matching Cancel's own MilestoneCancelNonTerminalACsError (D-0004),
// which has no force-bypass either. force relaxes FSM-transition
// legality and sovereign-act requirements, not this structural AC
// precondition.
func TestPromote_MilestoneCancelledForce_DoesNotBypassOpenACGuard(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Work", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First criterion", testActor, nil))

	res, err := verb.Promote(r.ctx, r.tree(), "M-0001", "cancelled", testActor, "forcing through for the test", true, verb.PromoteOptions{})
	if err == nil {
		t.Fatalf("Promote(M-0001, cancelled, force=true) succeeded (res=%+v); want the guard to still refuse under force", res)
	}
	code, ok := entity.Code(err)
	if !ok || code != verb.CodeMilestonePromoteNonTerminalACs.ID {
		t.Fatalf("Code(err) = (%q, %v); want (%q, true)", code, ok, verb.CodeMilestonePromoteNonTerminalACs.ID)
	}
}

// TestPromote_MilestoneToInProgressWithOpenAC_Succeeds pins the guard's
// own target scope directly: `in_progress` is not `cancelled`, so an
// open AC must NOT block this transition — only a promote reaching
// `cancelled` is in scope for the guard.
func TestPromote_MilestoneToInProgressWithOpenAC_Succeeds(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Work", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First criterion", testActor, nil))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))

	// G-0269's activating-promote branch guard is out of scope for this
	// fixture (pure FSM scaffolding, not a branch-discipline test) —
	// force past it.
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "fixture-setup", true, verb.PromoteOptions{}))
	if e := r.tree().ByID("M-0001"); e == nil || e.Status != "in_progress" {
		t.Errorf("M-0001 = %+v; want status in_progress", e)
	}
}
