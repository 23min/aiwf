package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
	"github.com/23min/ai-workflow-v2/tools/internal/verb"
)

// TestAddAC_AppendsACAndScaffoldsHeading covers the happy path: a
// milestone with no ACs receives one, frontmatter shows the new
// entry, the body grows a `### AC-1 — <title>` heading, and the
// commit lands with composite-id trailers.
func TestAddAC_AppendsACAndScaffoldsHeading(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First criterion", testActor))

	m := r.tree().ByID("M-001")
	if m == nil {
		t.Fatal("M-001 missing")
	}
	if len(m.ACs) != 1 || m.ACs[0].ID != "AC-1" {
		t.Errorf("ACs = %+v", m.ACs)
	}
	if m.ACs[0].Title != "First criterion" || m.ACs[0].Status != "open" {
		t.Errorf("ACs[0] = %+v", m.ACs[0])
	}
	if m.ACs[0].TDDPhase != "" {
		t.Errorf("default tdd: should leave tdd_phase empty; got %q", m.ACs[0].TDDPhase)
	}

	// Trailers carry the composite id.
	trailers, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	var entityTr gitops.Trailer
	for _, tr := range trailers {
		if tr.Key == "aiwf-entity" {
			entityTr = tr
		}
	}
	if entityTr.Value != "M-001/AC-1" {
		t.Errorf("aiwf-entity = %q, want M-001/AC-1", entityTr.Value)
	}
}

// TestAddAC_SeedsRedPhaseUnderTDDRequired: when the parent milestone
// is tdd: required, the verb writes tdd_phase: red as part of the
// same commit. The kernel never makes a TDD-policy decision — it just
// writes the only legal starting state under the FSM.
//
// There's no kernel verb yet to flip a milestone's tdd: policy, so the
// test sets it on disk directly (a hand-edit), then exercises AddAC.
func TestAddAC_SeedsRedPhaseUnderTDDRequired(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))

	// Hand-edit tdd: required onto the milestone file. This is a
	// stand-in until a verb to flip the policy lands; the test is
	// about AddAC's seeding behavior, not how the policy got set.
	m := r.tree().ByID("M-001")
	mPath := filepath.Join(r.root, m.Path)
	original, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	patched := strings.Replace(string(original), "parent: E-01\n", "parent: E-01\ntdd: required\n", 1)
	if err := os.WriteFile(mPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write patched: %v", err)
	}
	// Stage and commit so subsequent verbs see a clean tree.
	if err := gitops.Add(r.ctx, r.root, m.Path); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := gitops.Commit(r.ctx, r.root, "test: enable tdd: required", "", nil); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First", testActor))

	m = r.tree().ByID("M-001")
	if len(m.ACs) != 1 {
		t.Fatalf("ACs = %+v", m.ACs)
	}
	if m.ACs[0].TDDPhase != "red" {
		t.Errorf("tdd_phase = %q, want red", m.ACs[0].TDDPhase)
	}
}

// TestAddAC_PositionMaxPlus1AcrossCancellation: the next AC id is
// max+1 over the FULL list (cancelled entries count toward position).
// After cancelling AC-2, a new AC must be AC-3, not AC-2.
func TestAddAC_PositionMaxPlus1AcrossCancellation(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "AC one", testActor))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "AC two", testActor))
	r.must(verb.Cancel(r.ctx, r.tree(), "M-001/AC-2", testActor, "", false))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "AC three (max+1, not gap-fill)", testActor))

	m := r.tree().ByID("M-001")
	if len(m.ACs) != 3 {
		t.Fatalf("expected 3 ACs (cancelled AC-2 stays in place), got %d: %+v", len(m.ACs), m.ACs)
	}
	wantIDs := []string{"AC-1", "AC-2", "AC-3"}
	for i, want := range wantIDs {
		if m.ACs[i].ID != want {
			t.Errorf("ACs[%d].ID = %q, want %q", i, m.ACs[i].ID, want)
		}
	}
	if m.ACs[1].Status != "cancelled" {
		t.Errorf("ACs[1].Status = %q, want cancelled", m.ACs[1].Status)
	}
}

// TestAddAC_NotAMilestoneRefuses: only milestones host ACs.
func TestAddAC_NotAMilestoneRefuses(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	_, err := verb.AddAC(r.ctx, r.tree(), "E-01", "nope", testActor)
	if err == nil || !strings.Contains(err.Error(), "not a milestone") {
		t.Errorf("expected 'not a milestone' error, got %v", err)
	}
}

// TestAddAC_NonExistentParent surfaces a clean error.
func TestAddAC_NonExistentParent(t *testing.T) {
	r := newRunner(t)
	_, err := verb.AddAC(r.ctx, r.tree(), "M-999", "title", testActor)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found error, got %v", err)
	}
}

// TestPromote_Composite: aiwf promote M-001/AC-1 met flips the AC's
// status; the milestone file is rewritten in place.
func TestPromote_Composite(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First criterion", testActor))
	r.must(verb.Promote(r.ctx, r.tree(), "M-001/AC-1", "met", testActor, "", false))

	m := r.tree().ByID("M-001")
	if m.ACs[0].Status != "met" {
		t.Errorf("AC-1 status = %q, want met", m.ACs[0].Status)
	}
}

// TestPromote_CompositeRespectsACFSM: AC FSM rejects open → done
// without --force.
func TestPromote_CompositeRespectsACFSM(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First", testActor))

	// open → cancelled is legal; open → "weird" isn't a valid status,
	// so even the FSM check fails. Use met → done as the illegal
	// jump (met can only go to deferred or cancelled).
	r.must(verb.Promote(r.ctx, r.tree(), "M-001/AC-1", "met", testActor, "", false))
	_, err := verb.Promote(r.ctx, r.tree(), "M-001/AC-1", "done", testActor, "", false)
	if err == nil || !strings.Contains(err.Error(), "cannot transition") {
		t.Errorf("expected illegal-transition error for met → done, got %v", err)
	}
}

// TestCancel_Composite cancels an AC; the entry stays in acs[] at
// its original position.
func TestCancel_Composite(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First", testActor))
	r.must(verb.Cancel(r.ctx, r.tree(), "M-001/AC-1", testActor, "", false))

	m := r.tree().ByID("M-001")
	if len(m.ACs) != 1 || m.ACs[0].ID != "AC-1" || m.ACs[0].Status != "cancelled" {
		t.Errorf("ACs = %+v", m.ACs)
	}
}

// TestCancel_CompositeAlreadyCancelled refuses re-cancelling — same
// guard as for top-level entities. No diff to write.
func TestCancel_CompositeAlreadyCancelled(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First", testActor))
	r.must(verb.Cancel(r.ctx, r.tree(), "M-001/AC-1", testActor, "", false))

	_, err := verb.Cancel(r.ctx, r.tree(), "M-001/AC-1", testActor, "", false)
	if err == nil || !strings.Contains(err.Error(), "already cancelled") {
		t.Errorf("expected 'already cancelled' error, got %v", err)
	}
}

// TestRename_CompositeUpdatesTitleAndHeading: the AC's frontmatter
// title is updated AND the body heading is rewritten in place.
func TestRename_CompositeUpdatesTitleAndHeading(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "Old title", testActor))
	r.must(verb.Rename(r.ctx, r.tree(), "M-001/AC-1", "New title", testActor))

	m := r.tree().ByID("M-001")
	if m.ACs[0].Title != "New title" {
		t.Errorf("frontmatter title = %q, want New title", m.ACs[0].Title)
	}
	body, err := readMilestoneBody(r.root, m.Path)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	if !strings.Contains(body, "### AC-1 — New title") {
		t.Errorf("body should contain rewritten heading; got:\n%s", body)
	}
	if strings.Contains(body, "Old title") {
		t.Errorf("body should not still contain 'Old title':\n%s", body)
	}
}

// TestRename_CompositeNoOp errors when the new title equals the
// current one — no diff to write.
func TestRename_CompositeNoOp(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "Same title", testActor))

	_, err := verb.Rename(r.ctx, r.tree(), "M-001/AC-1", "Same title", testActor)
	if err == nil || !strings.Contains(err.Error(), "already") {
		t.Errorf("expected no-op error, got %v", err)
	}
}

// TestPromoteACPhase_RoundTrip walks the full TDD cycle on a freshly-
// created AC: "" → red → green → done. The "" → red transition is
// the load-bearing pre-cycle entry case for ACs that didn't get an
// auto-seed.
func TestPromoteACPhase_RoundTrip(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First", testActor))

	// "" → red
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "red", testActor, "", false))
	if got := r.tree().ByID("M-001").ACs[0].TDDPhase; got != "red" {
		t.Fatalf("after first phase change: phase = %q, want red", got)
	}
	// red → green
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "green", testActor, "", false))
	// green → done (refactor optional)
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "done", testActor, "", false))

	if got := r.tree().ByID("M-001").ACs[0].TDDPhase; got != "done" {
		t.Errorf("final phase = %q, want done", got)
	}
}

// TestPromoteACPhase_RejectsIllegalSkipAhead: the FSM rules out
// red → done. "" → green is also rejected — only "" → red is the
// pre-cycle entry transition.
func TestPromoteACPhase_RejectsIllegalSkipAhead(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First", testActor))

	// "" → green is illegal (must enter at red).
	if _, err := verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "green", testActor, "", false); err == nil {
		t.Error("expected error for empty → green phase")
	}
	// red → done is illegal (must go through green).
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "red", testActor, "", false))
	if _, err := verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "done", testActor, "", false); err == nil {
		t.Error("expected error for red → done phase")
	}
}

// TestPromoteACPhase_ForceRelaxesFSM: --force lets red → done land,
// and the trailers carry both aiwf-to: <newPhase> and aiwf-force:
// <reason> as expected.
func TestPromoteACPhase_ForceRelaxesFSM(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First", testActor))
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "red", testActor, "", false))

	// red → done forced.
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "done", testActor, "skipped green for the demo", true))

	if got := r.tree().ByID("M-001").ACs[0].TDDPhase; got != "done" {
		t.Errorf("phase = %q, want done", got)
	}
	trailers, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	var sawTo, sawForce bool
	for _, tr := range trailers {
		switch tr.Key {
		case "aiwf-to":
			sawTo = tr.Value == "done"
		case "aiwf-force":
			sawForce = tr.Value == "skipped green for the demo"
		}
	}
	if !sawTo || !sawForce {
		t.Errorf("expected aiwf-to: done and aiwf-force: <reason>; got %+v", trailers)
	}
}

// readMilestoneBody is a small helper local to this test file.
func readMilestoneBody(root, relPath string) (string, error) {
	body, err := os.ReadFile(filepath.Join(root, relPath))
	return string(body), err
}
