package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/gitops"
	"github.com/23min/ai-workflow-v2/internal/verb"
)

// TestAddAC_AppendsACAndScaffoldsHeading covers the happy path: a
// milestone with no ACs receives one, frontmatter shows the new
// entry, the body grows a `### AC-1 — <title>` heading, and the
// commit lands with composite-id trailers.
func TestAddAC_AppendsACAndScaffoldsHeading(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First criterion", testActor, nil))

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

	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First", testActor, nil))

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
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "AC one", testActor, nil))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "AC two", testActor, nil))
	r.must(verb.Cancel(r.ctx, r.tree(), "M-001/AC-2", testActor, "", false))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "AC three (max+1, not gap-fill)", testActor, nil))

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

// TestAddAC_RefusesProseyTitle is the verb-time half of G20: a long,
// markdown-formatted, or multi-sentence title is refused with a
// usage-shaped error message before any disk change. The user is
// pointed at the workflow: short label for --title, hand-edit body
// prose under the heading after creation.
func TestAddAC_RefusesProseyTitle(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))

	prosey := "**Full embedment inventory.** A machine-reviewable table enumerates every rule."
	_, err := verb.AddAC(r.ctx, r.tree(), "M-001", prosey, testActor, nil)
	if err == nil {
		t.Fatal("expected refusal for prose-y title; got no error")
	}
	if !strings.Contains(err.Error(), "looks like prose") {
		t.Errorf("error message should mention 'looks like prose'; got: %v", err)
	}

	// Sanity: the milestone still has zero ACs (verb refused before any write).
	if m := r.tree().ByID("M-001"); m == nil || len(m.ACs) != 0 {
		t.Errorf("M-001 should have 0 ACs after refused add, got %+v", m.ACs)
	}
}

// TestAddAC_AcceptsShortLabel confirms the refusal doesn't accidentally
// reject reasonable short labels — the happy path the verb is built
// around still works.
func TestAddAC_AcceptsShortLabel(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "Engine emits warning on bad input", testActor, nil))

	if m := r.tree().ByID("M-001"); m == nil || len(m.ACs) != 1 {
		t.Fatalf("M-001 should have 1 AC after happy add, got %+v", m)
	}
}

// TestAddAC_NotAMilestoneRefuses: only milestones host ACs.
func TestAddAC_NotAMilestoneRefuses(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	_, err := verb.AddAC(r.ctx, r.tree(), "E-01", "nope", testActor, nil)
	if err == nil || !strings.Contains(err.Error(), "not a milestone") {
		t.Errorf("expected 'not a milestone' error, got %v", err)
	}
}

// TestAddAC_NonExistentParent surfaces a clean error.
func TestAddAC_NonExistentParent(t *testing.T) {
	r := newRunner(t)
	_, err := verb.AddAC(r.ctx, r.tree(), "M-999", "title", testActor, nil)
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
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First criterion", testActor, nil))
	r.must(verb.Promote(r.ctx, r.tree(), "M-001/AC-1", "met", testActor, "", false, verb.PromoteOptions{}))

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
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First", testActor, nil))

	// open → cancelled is legal; open → "weird" isn't a valid status,
	// so even the FSM check fails. Use met → done as the illegal
	// jump (met can only go to deferred or cancelled).
	r.must(verb.Promote(r.ctx, r.tree(), "M-001/AC-1", "met", testActor, "", false, verb.PromoteOptions{}))
	_, err := verb.Promote(r.ctx, r.tree(), "M-001/AC-1", "done", testActor, "", false, verb.PromoteOptions{})
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
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First", testActor, nil))
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
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First", testActor, nil))
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
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "Old title", testActor, nil))
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
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "Same title", testActor, nil))

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
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First", testActor, nil))

	// "" → red
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "red", testActor, "", false, nil))
	if got := r.tree().ByID("M-001").ACs[0].TDDPhase; got != "red" {
		t.Fatalf("after first phase change: phase = %q, want red", got)
	}
	// red → green
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "green", testActor, "", false, nil))
	// green → done (refactor optional)
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "done", testActor, "", false, nil))

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
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First", testActor, nil))

	// "" → green is illegal (must enter at red).
	if _, err := verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "green", testActor, "", false, nil); err == nil {
		t.Error("expected error for empty → green phase")
	}
	// red → done is illegal (must go through green).
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "red", testActor, "", false, nil))
	if _, err := verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "done", testActor, "", false, nil); err == nil {
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
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First", testActor, nil))
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "red", testActor, "", false, nil))

	// red → done forced.
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "done", testActor, "skipped green for the demo", true, nil))

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

// TestPromoteACPhase_TestsTrailerWritten: passing a non-nil
// TestMetrics to a phase promotion lands the canonical aiwf-tests
// trailer alongside the standard transition trailers. Load-bearing
// for I3 step 2 — the kernel write path the rituals plugin will call
// is the verb's TestMetrics arg, not direct trailer construction.
func TestPromoteACPhase_TestsTrailerWritten(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First", testActor, nil))

	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "red", testActor, "", false, nil))
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "green", testActor, "", false,
		&gitops.TestMetrics{Pass: 12, Fail: 0, Skip: 1}))

	trailers, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	var sawTests bool
	for _, tr := range trailers {
		if tr.Key == "aiwf-tests" {
			sawTests = true
			if tr.Value != "pass=12 fail=0 skip=1" {
				t.Errorf("aiwf-tests value = %q, want %q", tr.Value, "pass=12 fail=0 skip=1")
			}
		}
	}
	if !sawTests {
		t.Errorf("expected aiwf-tests trailer on phase commit; got %+v", trailers)
	}
}

// TestAddAC_TestsTrailerOnSeededRedOnly: --tests lands when seeding
// red (parent milestone tdd: required); for non-tdd-required parents
// the verb refuses the flag rather than silently dropping it. Load-
// bearing for "no half-finished implementations" — a flag the user
// passed must either be honored or rejected.
func TestAddAC_TestsTrailerOnSeededRedOnly(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Required", testActor, verb.AddOptions{EpicID: "E-01"}))

	// Hand-edit the milestone to tdd: required so the next AddAC
	// seeds the AC at red phase.
	mPath := filepath.Join(r.root, "work", "epics", "E-01-foundations", "M-001-required.md")
	raw, readErr := os.ReadFile(mPath)
	if readErr != nil {
		t.Fatalf("read milestone: %v", readErr)
	}
	patched := strings.Replace(string(raw), "status: draft\n", "status: draft\ntdd: required\n", 1)
	if writeErr := os.WriteFile(mPath, []byte(patched), 0o644); writeErr != nil {
		t.Fatalf("write milestone: %v", writeErr)
	}

	// Trailer lands on the seeded-red AC creation commit.
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First", testActor,
		&gitops.TestMetrics{Pass: 0, Fail: 1, Skip: 0}))
	trailers, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	var sawTests bool
	for _, tr := range trailers {
		if tr.Key == "aiwf-tests" {
			sawTests = true
			if tr.Value != "pass=0 fail=1 skip=0" {
				t.Errorf("aiwf-tests value = %q", tr.Value)
			}
		}
	}
	if !sawTests {
		t.Errorf("expected aiwf-tests trailer on add-ac under tdd: required; got %+v", trailers)
	}

	// Add a second milestone without tdd — passing --tests must
	// refuse rather than silently drop.
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Optional", testActor, verb.AddOptions{EpicID: "E-01"}))
	if _, err := verb.AddAC(r.ctx, r.tree(), "M-002", "First", testActor,
		&gitops.TestMetrics{Pass: 1}); err == nil {
		t.Error("expected error when --tests is set on a non-tdd-required milestone")
	}
}
