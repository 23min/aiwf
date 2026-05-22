package cellcoverage

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/workflows/spec"
)

// TestNewCellFixture_Bootstraps pins that NewCellFixture produces a
// usable fresh repo: workdir exists, aiwf.yaml present, tree loads
// clean with zero entities. The shape every per-cell test starts
// from.
func TestNewCellFixture_Bootstraps(t *testing.T) {
	t.Parallel()
	f := NewCellFixture(t)
	if f.Root == "" {
		t.Fatal("Root is empty")
	}
	tr := f.Tree()
	if len(tr.Entities) != 0 {
		t.Errorf("fresh fixture should have zero entities; got %d", len(tr.Entities))
	}
	// aiwf.yaml lands at the repo root after init.
	if _, err := f.openRel("aiwf.yaml"); err != nil {
		t.Errorf("aiwf.yaml missing after NewCellFixture: %v", err)
	}
}

// TestBringEntityToState covers a spread of (Kind, FromState) targets
// to pin the verb-sequence dispatch logic. The full enumeration runs
// in AC-3's per-cell driver; this set spans the dispatch arms.
func TestBringEntityToState(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		kind      entity.Kind
		fromState string
		wantID    string
		wantPath  string // optional: assert on-disk presence
	}{
		// Top-level kind variants
		{"epic-proposed", entity.KindEpic, "proposed", "E-0001", ""},
		{"epic-active", entity.KindEpic, "active", "E-0001", ""},
		{"epic-done", entity.KindEpic, "done", "E-0001", ""},
		{"epic-cancelled", entity.KindEpic, "cancelled", "E-0001", ""},
		{"milestone-draft", entity.KindMilestone, "draft", "M-0001", ""},
		{"milestone-in_progress", entity.KindMilestone, "in_progress", "M-0001", ""},
		// The principled-design case the user flagged: zero-AC
		// promote-to-done. Spec table says Legal; AntiRule says
		// "milestone is NOT required to have >= 1 AC"; FSM has
		// in_progress -> done edge. Vacuous satisfaction of
		// "all-children-acs.status != open" is mathematically clean.
		{"milestone-done-zero-acs", entity.KindMilestone, "done", "M-0001", ""},
		{"milestone-cancelled", entity.KindMilestone, "cancelled", "M-0001", ""},
		{"adr-proposed", entity.KindADR, "proposed", "ADR-0001", ""},
		{"adr-accepted", entity.KindADR, "accepted", "ADR-0001", ""},
		{"adr-rejected", entity.KindADR, "rejected", "ADR-0001", ""},
		{"gap-open", entity.KindGap, "open", "G-0001", ""},
		{"gap-addressed", entity.KindGap, "addressed", "G-0001", ""},
		{"gap-wontfix", entity.KindGap, "wontfix", "G-0001", ""},
		{"decision-proposed", entity.KindDecision, "proposed", "D-0001", ""},
		{"decision-accepted", entity.KindDecision, "accepted", "D-0001", ""},
		{"decision-rejected", entity.KindDecision, "rejected", "D-0001", ""},
		{"contract-proposed", entity.KindContract, "proposed", "C-0001", ""},
		{"contract-accepted", entity.KindContract, "accepted", "C-0001", ""},
		{"contract-deprecated", entity.KindContract, "deprecated", "C-0001", ""},
		{"contract-retired", entity.KindContract, "retired", "C-0001", ""},
		{"contract-rejected", entity.KindContract, "rejected", "C-0001", ""},
		// Sub-kinds (spec.KindAC; tracked via composite id)
		{"ac-open", spec.KindAC, "open", "M-0001/AC-1", ""},
		{"ac-met", spec.KindAC, "met", "M-0001/AC-1", ""},
		{"ac-deferred", spec.KindAC, "deferred", "M-0001/AC-1", ""},
		{"ac-cancelled", spec.KindAC, "cancelled", "M-0001/AC-1", ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := NewCellFixture(t)
			id := f.BringEntityToState(t, tc.kind, tc.fromState, BringOpts{})
			if id != tc.wantID {
				t.Errorf("BringEntityToState returned id %q, want %q", id, tc.wantID)
			}
			// Re-load tree and assert the resulting state matches.
			tr := f.Tree()
			if entity.IsCompositeID(id) {
				// AC: navigate to the slot.
				milestone, ac, err := LookupComposite(tr, id)
				if err != nil {
					t.Fatalf("lookup composite %q: %v", id, err)
				}
				_ = milestone
				if ac.Status != tc.fromState {
					t.Errorf("AC %q status = %q, want %q", id, ac.Status, tc.fromState)
				}
			} else {
				e := tr.ByID(id)
				if e == nil {
					t.Fatalf("entity %q not in tree after BringEntityToState", id)
				}
				if e.Status != tc.fromState {
					t.Errorf("entity %q status = %q, want %q", id, e.Status, tc.fromState)
				}
			}
		})
	}
}

// TestNextTDDPhaseTowards covers the TDD-phase FSM walker's branches
// per CLAUDE.md "every reachable conditional branch in the diff has
// an explicit test." The walker only walks toward done in practice
// (the SatisfyPredicate use site), but the function table covers all
// FSM source-target pairs because future predicates (or direct
// callers) may exercise them.
func TestNextTDDPhaseTowards(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		current string
		target  string
		want    string
	}{
		// Per https://go.dev/ — covering each branch of the tddPhase FSM:
		// `` → red, red → green, green → {refactor, done}, refactor → done.
		{"empty-to-red", "", "red", "red"},
		{"red-to-green", "red", "green", "green"},
		{"green-to-refactor", "green", "refactor", "refactor"},
		{"green-to-done", "green", "done", "done"},
		{"refactor-to-done", "refactor", "done", "done"},
		// Unknown source returns "" (no progress).
		{"unknown-source", "done", "anywhere", ""},
		{"junk-source", "junk", "done", ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := nextTDDPhaseTowards(tc.current, tc.target)
			if got != tc.want {
				t.Errorf("nextTDDPhaseTowards(%q, %q) = %q, want %q", tc.current, tc.target, got, tc.want)
			}
		})
	}
}

// TestBringEntityToState_MilestoneDoneWithACs covers the populated-
// fixture variant for the Milestone.done cell: BringOpts.ACs = N
// seeds N met-status ACs (phase done so acs-tdd-audit doesn't fire)
// before the milestone hops in_progress -> done. The default path
// (BringOpts.ACs = 0) takes the zero-AC vacuous-satisfaction route
// per the spec AntiRule; this case pins that the populated path
// also lands at done cleanly.
func TestBringEntityToState_MilestoneDoneWithACs(t *testing.T) {
	t.Parallel()
	f := NewCellFixture(t)
	id := f.BringEntityToState(t, entity.KindMilestone, entity.StatusDone, BringOpts{ACs: 2})
	if id != "M-0001" {
		t.Fatalf("BringEntityToState returned id %q, want M-0001", id)
	}
	tr := f.Tree()
	m := tr.ByID("M-0001")
	if m == nil {
		t.Fatal("M-0001 not found")
	}
	if m.Status != entity.StatusDone {
		t.Errorf("milestone status = %q, want %q", m.Status, entity.StatusDone)
	}
	if len(m.ACs) != 2 {
		t.Errorf("ACs len = %d, want 2", len(m.ACs))
	}
	for i, ac := range m.ACs {
		if ac.Status != entity.StatusMet {
			t.Errorf("AC %d status = %q, want %q", i, ac.Status, entity.StatusMet)
		}
		if ac.TDDPhase != entity.TDDPhaseDone {
			t.Errorf("AC %d tdd_phase = %q, want %q", i, ac.TDDPhase, entity.TDDPhaseDone)
		}
	}
}

// TestSatisfyPredicate exercises each non-trivial atom (the 7 atoms
// that need fixture mutation; verb-arg atoms — self.target-state,
// self.evidence — are no-ops since the driver supplies them at
// verb-time). After mutation, the helper self-verifies via
// spec.EvaluatePredicate; the test re-confirms by an independent
// EvaluatePredicate call.
func TestSatisfyPredicate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		setupKind  entity.Kind
		setupState string
		pred       spec.Predicate
		evalCtx    spec.EvalContext // ctx passed to EvaluatePredicate (typically empty for entity-side atoms)
	}{
		// self.addressed_by non-empty / == ""
		{"addressed_by-non-empty", entity.KindGap, "open", spec.Predicate{Subject: "self.addressed_by", Op: "non-empty"}, spec.EvalContext{}},
		{"addressed_by-empty", entity.KindGap, "open", spec.Predicate{Subject: "self.addressed_by", Op: "==", Value: ""}, spec.EvalContext{}},
		// parent.tdd == required
		{"parent-tdd-required", spec.KindAC, "open", spec.Predicate{Subject: "parent.tdd", Op: "==", Value: "required"}, spec.EvalContext{}},
		// any-child.status not in milestone-terminal-set
		{"any-child-non-terminal", entity.KindEpic, "active", spec.Predicate{Subject: "any-child.status", Op: "∉", Value: "milestone-terminal-set"}, spec.EvalContext{}},
		// any-child-ac.status == open
		{"any-child-ac-open", entity.KindMilestone, "in_progress", spec.Predicate{Subject: "any-child-ac.status", Op: "==", Value: "open"}, spec.EvalContext{}},
		// all-children-acs.status != open
		{"all-children-acs-non-open", entity.KindMilestone, "in_progress", spec.Predicate{Subject: "all-children-acs.status", Op: "!=", Value: "open"}, spec.EvalContext{}},
		// self.tdd_phase != done (AC at red/green; ctx.AC populated)
		{"tdd_phase-not-done", spec.KindAC, "open", spec.Predicate{Subject: "self.tdd_phase", Op: "!=", Value: "done"}, spec.EvalContext{}},
		// self.tdd_phase == done — walkACToPhase walks the TDD FSM
		// from red → green → done. Exercises walkACToPhase + nextTDDPhaseTowards.
		{"tdd_phase-eq-done", spec.KindAC, "open", spec.Predicate{Subject: "self.tdd_phase", Op: "==", Value: "done"}, spec.EvalContext{}},
		// self.superseded_by non-empty — satisfyADRSuperseded
		// builds a sibling ADR and runs the supersession verb,
		// populating SupersededBy atomically. Exercises
		// satisfyADRSuperseded + trailerValue.
		{"superseded_by-non-empty", entity.KindADR, "accepted", spec.Predicate{Subject: "self.superseded_by", Op: "non-empty"}, spec.EvalContext{}},
		// self.superseded_by == "" — no mutation needed; the ADR
		// is at accepted with SupersededBy empty by default.
		{"superseded_by-empty", entity.KindADR, "accepted", spec.Predicate{Subject: "self.superseded_by", Op: "==", Value: ""}, spec.EvalContext{}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := NewCellFixture(t)
			id := f.BringEntityToState(t, tc.setupKind, tc.setupState, BringOpts{})
			f.SatisfyPredicate(t, tc.pred, id, &tc.evalCtx)

			// Re-load and verify independently via spec.EvaluatePredicate.
			tr := f.Tree()
			e, evalCtx, err := resolveForEval(tr, id, tc.evalCtx)
			if err != nil {
				t.Fatalf("resolveForEval %q: %v", id, err)
			}
			ok, err := spec.EvaluatePredicate(tc.pred, e, tr, evalCtx)
			if err != nil {
				t.Fatalf("EvaluatePredicate %+v: %v", tc.pred, err)
			}
			if !ok {
				t.Errorf("predicate %+v does not hold against entity %q after SatisfyPredicate", tc.pred, id)
			}
		})
	}
}
