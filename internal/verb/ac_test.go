package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// TestAddAC_AppendsACAndScaffoldsHeading covers the happy path: a
// milestone with no ACs receives one, frontmatter shows the new
// entry, the body grows a `### AC-1 — <title>` heading, and the
// commit lands with composite-id trailers.
func TestAddAC_AppendsACAndScaffoldsHeading(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First criterion", testActor, nil))

	m := r.tree().ByID("M-0001")
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
	if entityTr.Value != "M-0001/AC-1" {
		t.Errorf("aiwf-entity = %q, want M-001/AC-1", entityTr.Value)
	}
}

// TestAddAC_RewritesPlaceholderHeadingInPlace is the G-0247 verb-side
// guard: when the milestone body already carries a `### AC-N` heading
// for the id being allocated — the placeholder the ritual milestone
// template ships — `aiwf add ac` rewrites it in place instead of
// appending a second `### AC-1` heading that aiwf check's set-collapse
// used to hide.
func TestAddAC_RewritesPlaceholderHeadingInPlace(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

	// Simulate the ritual template: placeholder AC-1 and AC-2 headings
	// already anchor the body before any AC exists in frontmatter.
	m := r.tree().ByID("M-0001")
	abs := filepath.Join(r.root, m.Path)
	raw, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	withPlaceholder := string(raw) +
		"\n### AC-1 — <observable behavior>\n\n<Prose: examples.>\n" +
		"\n### AC-2 — <observable behavior>\n\n<Prose…>\n"
	if err = os.WriteFile(abs, []byte(withPlaceholder), 0o644); err != nil {
		t.Fatal(err)
	}

	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "Real criterion", testActor, nil))

	body, err := readMilestoneBody(r.root, r.tree().ByID("M-0001").Path)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	// Anchor on the `—` separator so the count can't be fooled by an
	// `### AC-10`-style prefix collision (CLAUDE.md §"Substring
	// assertions are not structural assertions").
	if n := strings.Count(body, "### AC-1 —"); n != 1 {
		t.Errorf("expected exactly one `### AC-1 —` heading, got %d:\n%s", n, body)
	}
	if !strings.Contains(body, "### AC-1 — Real criterion") {
		t.Errorf("placeholder heading not rewritten to the real title:\n%s", body)
	}
	// The unclaimed AC-2 placeholder is left untouched (the closure's
	// non-matching branch).
	if !strings.Contains(body, "### AC-2 — <observable behavior>") {
		t.Errorf("unrelated AC-2 placeholder should be untouched:\n%s", body)
	}
}

// TestAddAC_AppendsBodyContentWhenNoPlaceholder: with no `### AC-N`
// placeholder present, the heading and supplied body content are
// appended at the end of the body — the historical path, preserved.
func TestAddAC_AppendsBodyContentWhenNoPlaceholder(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddACBatch(r.ctx, r.tree(), "M-0001",
		[]string{"Real criterion"}, [][]byte{[]byte("The contract prose.")}, testActor, nil))

	body, err := readMilestoneBody(r.root, r.tree().ByID("M-0001").Path)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	hi := strings.Index(body, "### AC-1 — Real criterion")
	ci := strings.Index(body, "The contract prose.")
	if hi < 0 || ci < 0 || ci < hi {
		t.Errorf("body content should follow the appended heading; heading@%d prose@%d:\n%s", hi, ci, body)
	}
}

// TestRename_CompositeLeavesSiblingHeadings: renaming one AC's heading
// in a multi-AC body rewrites only the targeted `### AC-N` and leaves
// sibling headings untouched (the rewrite closure's non-matching
// branch).
func TestRename_CompositeLeavesSiblingHeadings(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First criterion", testActor, nil))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "Second criterion", testActor, nil))
	r.must(verb.Rename(r.ctx, r.tree(), "M-0001/AC-2", "Renamed second", testActor, 0))

	body, err := readMilestoneBody(r.root, r.tree().ByID("M-0001").Path)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	if !strings.Contains(body, "### AC-1 — First criterion") {
		t.Errorf("sibling AC-1 heading should be untouched:\n%s", body)
	}
	if !strings.Contains(body, "### AC-2 — Renamed second") {
		t.Errorf("AC-2 heading should be rewritten:\n%s", body)
	}
}

// TestAddAC_PlaceholderHeadingCoLocatesBodyContent: when a placeholder
// heading is rewritten in place AND the operator supplies AC body
// content (--body-file), the content lands beneath the rewritten
// heading rather than orphaned at the end of the document.
func TestAddAC_PlaceholderHeadingCoLocatesBodyContent(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

	m := r.tree().ByID("M-0001")
	abs := filepath.Join(r.root, m.Path)
	raw, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	withPlaceholder := string(raw) + "\n### AC-1 — <observable behavior>\n\n<Prose.>\n"
	if err = os.WriteFile(abs, []byte(withPlaceholder), 0o644); err != nil {
		t.Fatal(err)
	}

	r.must(verb.AddACBatch(r.ctx, r.tree(), "M-0001",
		[]string{"Real criterion"}, [][]byte{[]byte("The contract prose.")}, testActor, nil))

	body, err := readMilestoneBody(r.root, r.tree().ByID("M-0001").Path)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	// Anchor on the `—` separator so the count can't be fooled by an
	// `### AC-10`-style prefix collision (CLAUDE.md §"Substring
	// assertions are not structural assertions").
	if n := strings.Count(body, "### AC-1 —"); n != 1 {
		t.Errorf("expected exactly one `### AC-1 —` heading, got %d:\n%s", n, body)
	}
	hi := strings.Index(body, "### AC-1 — Real criterion")
	ci := strings.Index(body, "The contract prose.")
	if hi < 0 || ci < 0 || ci < hi {
		t.Errorf("contract prose should appear beneath the rewritten heading; heading@%d prose@%d:\n%s", hi, ci, body)
	}
}

// TestAddAC_SeedsRedPhaseUnderTDDRequired: when the parent milestone
// is tdd: required, the verb writes tdd_phase: red as part of the
// same commit. The kernel never makes a TDD-policy decision — it just
// writes the only legal starting state under the FSM.
func TestAddAC_SeedsRedPhaseUnderTDDRequired(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "required"}))

	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First", testActor, nil))

	m := r.tree().ByID("M-0001")
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
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "AC one", testActor, nil))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "AC two", testActor, nil))
	r.must(verb.Cancel(r.ctx, r.tree(), "M-0001/AC-2", testActor, "", false))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "AC three (max+1, not gap-fill)", testActor, nil))

	m := r.tree().ByID("M-0001")
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
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

	prosey := "**Full embedment inventory.** A machine-reviewable table enumerates every rule."
	_, err := verb.AddAC(r.ctx, r.tree(), "M-0001", prosey, testActor, nil)
	if err == nil {
		t.Fatal("expected refusal for prose-y title; got no error")
	}
	if !strings.Contains(err.Error(), "looks like prose") {
		t.Errorf("error message should mention 'looks like prose'; got: %v", err)
	}

	// Sanity: the milestone still has zero ACs (verb refused before any write).
	if m := r.tree().ByID("M-0001"); m == nil || len(m.ACs) != 0 {
		t.Errorf("M-001 should have 0 ACs after refused add, got %+v", m.ACs)
	}
}

// TestAddAC_AcceptsShortLabel confirms the refusal doesn't accidentally
// reject reasonable short labels — the happy path the verb is built
// around still works.
func TestAddAC_AcceptsShortLabel(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "Engine emits warning on bad input", testActor, nil))

	if m := r.tree().ByID("M-0001"); m == nil || len(m.ACs) != 1 {
		t.Fatalf("M-001 should have 1 AC after happy add, got %+v", m)
	}
}

// TestAddAC_NotAMilestoneRefuses: only milestones host ACs.
func TestAddAC_NotAMilestoneRefuses(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	_, err := verb.AddAC(r.ctx, r.tree(), "E-0001", "nope", testActor, nil)
	if err == nil || !strings.Contains(err.Error(), "not a milestone") {
		t.Errorf("expected 'not a milestone' error, got %v", err)
	}
}

// TestAddAC_NonExistentParent surfaces a clean error.
func TestAddAC_NonExistentParent(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	_, err := verb.AddAC(r.ctx, r.tree(), "M-0999", "title", testActor, nil)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found error, got %v", err)
	}
}

// TestPromote_Composite: aiwf promote M-001/AC-1 met flips the AC's
// status; the milestone file is rewritten in place.
func TestPromote_Composite(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First criterion", testActor, nil))
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", "met", testActor, "", false, verb.PromoteOptions{}))

	m := r.tree().ByID("M-0001")
	if m.ACs[0].Status != "met" {
		t.Errorf("AC-1 status = %q, want met", m.ACs[0].Status)
	}
}

// TestPromote_CompositeRespectsACFSM: AC FSM rejects open → done
// without --force.
func TestPromote_CompositeRespectsACFSM(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First", testActor, nil))

	// open → cancelled is legal; open → "weird" isn't a valid status,
	// so even the FSM check fails. Use met → done as the illegal
	// jump (met can only go to deferred or cancelled).
	r.must(verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", "met", testActor, "", false, verb.PromoteOptions{}))
	_, err := verb.Promote(r.ctx, r.tree(), "M-0001/AC-1", "done", testActor, "", false, verb.PromoteOptions{})
	if err == nil || !strings.Contains(err.Error(), "cannot transition") {
		t.Errorf("expected illegal-transition error for met → done, got %v", err)
	}
}

// TestCancel_Composite cancels an AC; the entry stays in acs[] at
// its original position.
func TestCancel_Composite(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First", testActor, nil))
	r.must(verb.Cancel(r.ctx, r.tree(), "M-0001/AC-1", testActor, "", false))

	m := r.tree().ByID("M-0001")
	if len(m.ACs) != 1 || m.ACs[0].ID != "AC-1" || m.ACs[0].Status != "cancelled" {
		t.Errorf("ACs = %+v", m.ACs)
	}
}

// TestCancel_CompositeAlreadyCancelled refuses re-cancelling — same
// guard as for top-level entities. No diff to write.
func TestCancel_CompositeAlreadyCancelled(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First", testActor, nil))
	r.must(verb.Cancel(r.ctx, r.tree(), "M-0001/AC-1", testActor, "", false))

	_, err := verb.Cancel(r.ctx, r.tree(), "M-0001/AC-1", testActor, "", false)
	if err == nil || !strings.Contains(err.Error(), "already cancelled") {
		t.Errorf("expected 'already cancelled' error, got %v", err)
	}
}

// TestRename_CompositeUpdatesTitleAndHeading: the AC's frontmatter
// title is updated AND the body heading is rewritten in place.
func TestRename_CompositeUpdatesTitleAndHeading(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "Old title", testActor, nil))
	r.must(verb.Rename(r.ctx, r.tree(), "M-0001/AC-1", "New title", testActor, 0))

	m := r.tree().ByID("M-0001")
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
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "Same title", testActor, nil))

	_, err := verb.Rename(r.ctx, r.tree(), "M-0001/AC-1", "Same title", testActor, 0)
	if err == nil || !strings.Contains(err.Error(), "already") {
		t.Errorf("expected no-op error, got %v", err)
	}
}

// TestPromoteACPhase_RoundTrip walks the full TDD cycle on a freshly-
// created AC: "" → red → green → done. The "" → red transition is
// the load-bearing pre-cycle entry case for ACs that didn't get an
// auto-seed.
func TestPromoteACPhase_RoundTrip(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First", testActor, nil))

	// "" → red
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", "red", testActor, "", false, nil))
	if got := r.tree().ByID("M-0001").ACs[0].TDDPhase; got != "red" {
		t.Fatalf("after first phase change: phase = %q, want red", got)
	}
	// red → green
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", "green", testActor, "", false, nil))
	// green → done (refactor optional)
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", "done", testActor, "", false, nil))

	if got := r.tree().ByID("M-0001").ACs[0].TDDPhase; got != "done" {
		t.Errorf("final phase = %q, want done", got)
	}
}

// TestPromoteACPhase_RejectsIllegalSkipAhead: the FSM rules out
// red → done. "" → green is also rejected — only "" → red is the
// pre-cycle entry transition.
func TestPromoteACPhase_RejectsIllegalSkipAhead(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First", testActor, nil))

	// "" → green is illegal (must enter at red).
	if _, err := verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", "green", testActor, "", false, nil); err == nil {
		t.Error("expected error for empty → green phase")
	}
	// red → done is illegal (must go through green).
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", "red", testActor, "", false, nil))
	if _, err := verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", "done", testActor, "", false, nil); err == nil {
		t.Error("expected error for red → done phase")
	}
}

// TestPromoteACPhase_ForceRelaxesFSM: --force lets red → done land,
// and the trailers carry both aiwf-to: <newPhase> and aiwf-force:
// <reason> as expected.
func TestPromoteACPhase_ForceRelaxesFSM(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First", testActor, nil))
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", "red", testActor, "", false, nil))

	// red → done forced.
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", "done", testActor, "skipped green for the demo", true, nil))

	if got := r.tree().ByID("M-0001").ACs[0].TDDPhase; got != "done" {
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
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First", testActor, nil))

	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", "red", testActor, "", false, nil))
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", "green", testActor, "", false,
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
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Required", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "required"}))

	// Trailer lands on the seeded-red AC creation commit.
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", "First", testActor,
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
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Optional", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	if _, err := verb.AddAC(r.ctx, r.tree(), "M-0002", "First", testActor,
		&gitops.TestMetrics{Pass: 1}); err == nil {
		t.Error("expected error when --tests is set on a non-tdd-required milestone")
	}
}
