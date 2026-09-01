package verb_test

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// ac_phase_reset_test.go pins G-0569: an AC returning to `open` starts its TDD
// cycle over.
//
// The phase FSM is linear and terminal, so a carried-over `done` satisfies
// `acs-tdd-audit`'s "met requires done" in advance. The second `met` then rides
// the first cycle's evidence — measured firing twice on M-0171, on a
// `tdd: required` milestone, with no force on the re-promote.

// reworkFixture drives one AC through a full cycle to met/done.
func reworkFixture(t *testing.T) *runner {
	t.Helper()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "required"}))
	r.must(verb.AddACBatch(r.ctx, r.tree(), "M-0001", []string{"Does the thing"},
		[][]byte{[]byte("Real prose describing the criterion.")}, testActor))
	gitCheckoutNewBranch(t, r.root, "epic/E-0001-platform")
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "", false, verb.PromoteOptions{}))
	for _, p := range []string{"red", "green", "done"} {
		r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", p, testActor, "", false, nil))
	}
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", "met", testActor, "", false, verb.PromoteOptions{}))
	return r
}

func acPhase(t *testing.T, r *runner) (status entity.Status, phase string) {
	t.Helper()
	m := r.tree().ByID("M-0001")
	if m == nil || len(m.ACs) == 0 {
		t.Fatal("fixture lost the milestone or its AC")
	}
	return m.ACs[0].Status, m.ACs[0].TDDPhase
}

// TestPromoteAC_ReturnToOpen_ResetsTheTDDPhase is the defect itself. Reset is to
// the pre-cycle empty phase, not to `red`: `red` claims a failing test exists,
// which arriving at `open` does not establish — the same reasoning AddACBatch
// applies to a freshly-created AC.
func TestPromoteAC_ReturnToOpen_ResetsTheTDDPhase(t *testing.T) {
	t.Parallel()
	r := reworkFixture(t)
	if s, p := acPhase(t, r); s != "met" || p != "done" {
		t.Fatalf("fixture did not reach met/done; got %s/%s", s, p)
	}
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", "open", testActor, "review refuted the evidence", true, verb.PromoteOptions{}))
	s, p := acPhase(t, r)
	if s != "open" {
		t.Fatalf("status = %q, want open", s)
	}
	if p != "" {
		t.Errorf("tdd_phase = %q, want empty — the second cycle must start from the pre-cycle state, not inherit the first cycle's evidence", p)
	}
}

// TestPromoteAC_ReturnToOpen_FromDeferred_ResetsToo pins the second door. The
// reset keys on arrival at `open`, not on the met -> open edge, so a forced
// demote from `deferred` cannot carry a stale phase across either.
func TestPromoteAC_ReturnToOpen_FromDeferred_ResetsToo(t *testing.T) {
	t.Parallel()
	r := reworkFixture(t)
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", "deferred", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", "open", testActor, "reopened", true, verb.PromoteOptions{}))
	if s, p := acPhase(t, r); s != "open" || p != "" {
		t.Errorf("got %s/%q, want open with an empty phase", s, p)
	}
}

// TestPromoteAC_RedundantOpenPromote_ConvergesAndKeepsThePhase pins the state
// the reset must not touch. An AC at `open` carrying a finished phase is what
// wf-tdd-cycle produces in the window between the cycle ending and the `met`
// promote — every tdd: required AC passes through it. Nothing distinguishes it
// from one a pre-reset demote left behind, so a redundant promote converges and
// leaves the evidence alone, forced or not.
func TestPromoteAC_RedundantOpenPromote_ConvergesAndKeepsThePhase(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "required"}))
	r.must(verb.AddACBatch(r.ctx, r.tree(), "M-0001", []string{"Does the thing"},
		[][]byte{[]byte("Real prose describing the criterion.")}, testActor))
	gitCheckoutNewBranch(t, r.root, "epic/E-0001-platform")
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "", false, verb.PromoteOptions{}))
	for _, ph := range []string{"red", "green", "done"} {
		r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", ph, testActor, "", false, nil))
	}
	if s, ph := acPhase(t, r); s != "open" || ph != "done" {
		t.Fatalf("fixture = %s/%q, want the ordinary pre-met state open/done", s, ph)
	}

	res, err := verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", "open", testActor, "", false, verb.PromoteOptions{})
	if err != nil {
		t.Fatalf("a redundant open promote must converge, not refuse: %v", err)
	}
	if !res.NoOp {
		t.Errorf("NoOp = false; nothing arrived, so there is nothing to write")
	}
	if _, ph := acPhase(t, r); ph != "done" {
		t.Fatalf("tdd_phase = %q, want done — a finished cycle's evidence was destroyed", ph)
	}

	// Force does not change the answer: a sovereign override has no transition
	// to re-apply either.
	if _, err := verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", "open", testActor, "idempotent re-run", true, verb.PromoteOptions{}); err != nil {
		t.Fatalf("forced redundant promote: %v", err)
	}
	if _, ph := acPhase(t, r); ph != "done" {
		t.Errorf("tdd_phase = %q under --force, want done", ph)
	}
}
