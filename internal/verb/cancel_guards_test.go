package verb_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestCancel_EpicWithNonTerminalChildMilestone_Refuses (M-0139/AC-1):
// cancelling an epic that still owns a non-terminal (draft) child
// milestone must refuse with the structured
// CodeEpicCancelNonTerminalChildren code (D-0003), listing the
// offending milestone id, and produce no Result.
func TestCancel_EpicWithNonTerminalChildMilestone_Refuses(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Doomed", testActor, verb.AddOptions{}))
	// A fresh milestone enters at `draft` — non-terminal.
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Child", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

	res, err := verb.Cancel(r.ctx, r.tree(), "E-0001", testActor, "", false)
	if err == nil {
		t.Fatalf("Cancel(E-0001) succeeded (res=%+v); want refusal because M-0001 is non-terminal", res)
	}
	code, ok := entity.Code(err)
	if !ok || code != verb.CodeEpicCancelNonTerminalChildren.ID {
		t.Fatalf("Code(err) = (%q, %v); want (%q, true)", code, ok, verb.CodeEpicCancelNonTerminalChildren.ID)
	}
	if !strings.Contains(err.Error(), "M-0001") {
		t.Errorf("error message %q does not list offending milestone M-0001", err.Error())
	}
}

// TestCancel_MilestoneWithOpenAC_Refuses (M-0139/AC-2): cancelling a
// milestone with an `open` AC must refuse with the structured
// CodeMilestoneCancelNonTerminalACs code (D-0004), listing the
// composite AC id, and produce no Result.
func TestCancel_MilestoneWithOpenAC_Refuses(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Work", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	// A freshly-added AC enters at `open`.
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First criterion", testActor, nil))

	res, err := verb.Cancel(r.ctx, r.tree(), "M-0001", testActor, "", false)
	if err == nil {
		t.Fatalf("Cancel(M-0001) succeeded (res=%+v); want refusal because AC-1 is open", res)
	}
	code, ok := entity.Code(err)
	if !ok || code != verb.CodeMilestoneCancelNonTerminalACs.ID {
		t.Fatalf("Code(err) = (%q, %v); want (%q, true)", code, ok, verb.CodeMilestoneCancelNonTerminalACs.ID)
	}
	if !strings.Contains(err.Error(), "M-0001/AC-1") {
		t.Errorf("error message %q does not list offending composite id M-0001/AC-1", err.Error())
	}
}

// TestCancel_AllChildrenTerminal_Succeeds (M-0139/AC-3): the guards
// must not refuse a legal cancel. An epic whose only child milestone is
// cancelled (terminal) cancels cleanly; a milestone with no ACs cancels
// cleanly. Characterization against a "refuse everything" regression.
func TestCancel_AllChildrenTerminal_Succeeds(t *testing.T) {
	t.Parallel()

	t.Run("epic-with-terminal-child", func(t *testing.T) {
		t.Parallel()
		r := newRunner(t)
		r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Doomed", testActor, verb.AddOptions{}))
		r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Child", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
		// Cancel the child first so the epic's only child is terminal.
		r.must(verb.Cancel(r.ctx, r.tree(), "M-0001", testActor, "", false))

		r.must(verb.Cancel(r.ctx, r.tree(), "E-0001", testActor, "", false))
		if e := r.tree().ByID("E-0001"); e == nil || e.Status != "cancelled" {
			t.Errorf("E-0001 = %+v; want status cancelled", e)
		}
	})

	t.Run("milestone-with-no-acs", func(t *testing.T) {
		t.Parallel()
		r := newRunner(t)
		r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
		r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Work", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

		r.must(verb.Cancel(r.ctx, r.tree(), "M-0001", testActor, "", false))
		if e := r.tree().ByID("M-0001"); e == nil || e.Status != "cancelled" {
			t.Errorf("M-0001 = %+v; want status cancelled", e)
		}
	})
}
