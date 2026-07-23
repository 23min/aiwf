package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// findingByCode returns the first finding with the given code+subcode.
// "" subcode matches any subcode. Helper local to this file.
func findingByCode(fs []Finding, code, subcode string) *Finding {
	for i := range fs {
		if fs[i].Code != code {
			continue
		}
		if subcode != "" && fs[i].Subcode != subcode {
			continue
		}
		return &fs[i]
	}
	return nil
}

func TestAcsShape_CleanMilestoneNoFindings(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo", Status: "in_progress", Parent: "E-0001",
		TDD: "required",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "First", Status: "open", TDDPhase: "red"},
			{ID: "AC-2", Title: "Second", Status: "met", TDDPhase: "done"},
		},
	})
	if got := acsShape(tr); len(got) != 0 {
		t.Errorf("expected zero findings, got: %+v", got)
	}
}

func TestAcsShape_AbsentACsAndTDD(t *testing.T) {
	t.Parallel()
	// A pre-I2 milestone with no acs[] and no tdd: must produce no
	// findings. This is the load-bearing backwards-compat assertion.
	tr := makeTree(&entity.Entity{
		ID: "M-0001", Kind: entity.KindMilestone, Title: "Pre-I2", Status: "in_progress", Parent: "E-0001",
	})
	if got := acsShape(tr); len(got) != 0 {
		t.Errorf("absent acs/tdd should produce no findings, got: %+v", got)
	}
}

func TestAcsShape_IDProblems(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		acs     []entity.AcceptanceCriterion
		wantSub string
	}{
		{
			name: "missing id",
			acs: []entity.AcceptanceCriterion{
				{Title: "no id", Status: "open"},
			},
			wantSub: "id",
		},
		{
			name: "id wrong format",
			acs: []entity.AcceptanceCriterion{
				{ID: "ac-1", Title: "lowercase", Status: "open"},
			},
			wantSub: "id",
		},
		{
			name: "id at wrong position",
			acs: []entity.AcceptanceCriterion{
				{ID: "AC-2", Title: "should be AC-1", Status: "open"},
			},
			wantSub: "id",
		},
		{
			name: "second slot wrong position",
			acs: []entity.AcceptanceCriterion{
				{ID: "AC-1", Title: "ok", Status: "open"},
				{ID: "AC-3", Title: "should be AC-2", Status: "open"},
			},
			wantSub: "id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tr := makeTree(&entity.Entity{
				ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
				Status: "in_progress", Parent: "E-0001", ACs: tt.acs,
			})
			got := acsShape(tr)
			if findingByCode(got, CodeACsShape, tt.wantSub) == nil {
				t.Errorf("expected acs-shape/%s; got: %+v", tt.wantSub, got)
			}
		})
	}
}

func TestAcsShape_TitleStatusTDDPhaseAndPolicy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		ac       entity.AcceptanceCriterion
		tdd      string
		wantCode string
		wantSub  string
	}{
		{
			name:     "title missing",
			ac:       entity.AcceptanceCriterion{ID: "AC-1", Title: "", Status: "open"},
			wantCode: CodeACsShape,
			wantSub:  "title",
		},
		{
			name:     "status missing",
			ac:       entity.AcceptanceCriterion{ID: "AC-1", Title: "x", Status: ""},
			wantCode: CodeACsShape,
			wantSub:  "status",
		},
		{
			name:     "status invalid",
			ac:       entity.AcceptanceCriterion{ID: "AC-1", Title: "x", Status: "frobnicate"},
			wantCode: CodeACsShape,
			wantSub:  "status",
		},
		{
			name:     "tdd_phase invalid",
			ac:       entity.AcceptanceCriterion{ID: "AC-1", Title: "x", Status: "open", TDDPhase: "blue"},
			wantCode: CodeACsShape,
			wantSub:  "tdd-phase",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tr := makeTree(&entity.Entity{
				ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
				Status: "in_progress", Parent: "E-0001", TDD: tt.tdd,
				ACs: []entity.AcceptanceCriterion{tt.ac},
			})
			got := acsShape(tr)
			if findingByCode(got, tt.wantCode, tt.wantSub) == nil {
				t.Errorf("expected %s/%s; got: %+v", tt.wantCode, tt.wantSub, got)
			}
		})
	}
}

// TestAcsShape_AbsentTDDPhaseNeverFires locks G-0286/M-0267 AC-1: under
// tdd: required, an AC with an absent tdd_phase never produces
// acs-shape/tdd-phase, regardless of the AC's own status. Presence is
// no longer required at all — only a *present-but-invalid* value is a
// shape error (see the sibling "tdd_phase invalid" case above). The
// "met requires tdd_phase: done" property is a distinct concern
// enforced by acsTDDAudit, not this rule (see
// TestAcsTDDAudit_MetWithAbsentPhaseFiresAsError below).
func TestAcsShape_AbsentTDDPhaseNeverFires(t *testing.T) {
	t.Parallel()
	for _, status := range []string{"open", "met", "deferred", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			tr := makeTree(&entity.Entity{
				ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
				Status: "in_progress", Parent: "E-0001", TDD: "required",
				ACs: []entity.AcceptanceCriterion{
					{ID: "AC-1", Title: "x", Status: status},
				},
			})
			got := acsShape(tr)
			if f := findingByCode(got, CodeACsShape, "tdd-phase"); f != nil {
				t.Errorf("absent tdd_phase (status %q) should not fire acs-shape/tdd-phase, got: %+v", status, f)
			}
		})
	}
}

func TestAcsShape_TDDPolicyInvalid(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001", TDD: "strict",
	})
	got := acsShape(tr)
	if findingByCode(got, CodeACsShape, "tdd-policy") == nil {
		t.Errorf("expected acs-shape/tdd-policy; got: %+v", got)
	}
}

func TestAcsShape_NonMilestoneSkipped(t *testing.T) {
	t.Parallel()
	// Other kinds shouldn't produce AC findings even if their fields
	// are populated (which the schema disallows but the struct permits).
	tr := makeTree(&entity.Entity{
		ID: "E-0001", Kind: entity.KindEpic, Title: "Foo", Status: "active",
		ACs: []entity.AcceptanceCriterion{{ID: "AC-1", Title: "x", Status: "open"}},
	})
	if got := acsShape(tr); len(got) != 0 {
		t.Errorf("epic with ACs should be skipped by acsShape, got: %+v", got)
	}
}

func TestAcsShape_PositionStableAcrossCancellation(t *testing.T) {
	t.Parallel()
	// AC-2 cancelled stays in position 2; new AC at position 3 must
	// be AC-3 (max+1, not gap-fill). This is the load-bearing
	// position-stability assertion.
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "ok", Status: "met"},
			{ID: "AC-2", Title: "cancelled", Status: "cancelled"},
			{ID: "AC-3", Title: "next", Status: "open"},
		},
	})
	if got := acsShape(tr); len(got) != 0 {
		t.Errorf("position-stable list should produce no findings, got: %+v", got)
	}

	// And: gap-filling (AC-2 cancelled, new entry as AC-2 again) is
	// flagged as a position error because the position rule counts
	// cancelled entries.
	bad := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "ok", Status: "met"},
			{ID: "AC-2", Title: "cancelled", Status: "cancelled"},
			{ID: "AC-2", Title: "gap-filled — wrong", Status: "open"},
		},
	})
	got := acsShape(bad)
	if findingByCode(got, CodeACsShape, "id") == nil {
		t.Errorf("expected an acs-shape/id finding for the position-3 duplicate AC-2; got: %+v", got)
	}
}

// TestAcsTitleProse_FlagsLongTitle covers the standing-check half of
// G20: an AC that landed via hand-edit (or pre-G20 tooling) with a
// prose-y title surfaces as a warning so the human knows to refactor.
func TestAcsTitleProse_FlagsLongTitle(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "**Full embedment inventory.** A machine-reviewable table enumerates every rule.", Status: "open"},
		},
	})
	got := acsTitleProse(tr)
	f := findingByCode(got, CodeACsTitleProse, "")
	if f == nil {
		t.Fatalf("expected acs-title-prose; got: %+v", got)
	}
	if f.Severity != SeverityWarning {
		t.Errorf("severity = %q, want warning", f.Severity)
	}
	if f.EntityID != "M-0007/AC-1" {
		t.Errorf("entityID = %q, want M-007/AC-1", f.EntityID)
	}
}

// TestAcsTitleProse_ShortTitleClean: a short label produces no
// finding, so the existing clean-fixture round-trip continues to
// pass.
func TestAcsTitleProse_ShortTitleClean(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "Engine emits warning on bad input", Status: "open"},
		},
	})
	if got := acsTitleProse(tr); len(got) != 0 {
		t.Errorf("short labels should produce no findings, got: %+v", got)
	}
}

func TestAcsTDDAudit_RequiredFiresAsError(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001", TDD: "required",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "x", Status: "met", TDDPhase: "green"},
		},
	})
	got := acsTDDAudit(tr)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1: %+v", len(got), got)
	}
	if got[0].Code != CodeACsTDDAudit {
		t.Errorf("code = %q, want \"acs-tdd-audit\"", got[0].Code)
	}
	if got[0].Severity != SeverityError {
		t.Errorf("severity = %q, want error", got[0].Severity)
	}
	if got[0].EntityID != "M-0007/AC-1" {
		t.Errorf("entityID = %q, want M-007/AC-1", got[0].EntityID)
	}
}

// TestAcsTDDAudit_MetWithAbsentPhaseFiresAsError locks G-0286/M-0267
// AC-2: now that acsShape no longer requires tdd_phase to be present
// (TestAcsShape_AbsentTDDPhaseNeverFires), a status: met AC with a
// wholly absent phase under tdd: required is reachable without ever
// tripping acs-shape — acsTDDAudit must be the rule that still catches
// it. Prior to the relaxation this combination was effectively
// screened out earlier by acs-shape's now-removed presence check, so
// acsTDDAudit's own coverage of "absent" (as opposed to
// "present-but-wrong") was untested.
func TestAcsTDDAudit_MetWithAbsentPhaseFiresAsError(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001", TDD: "required",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "x", Status: "met"},
		},
	})
	got := acsTDDAudit(tr)
	if len(got) != 1 || got[0].Code != CodeACsTDDAudit || got[0].Severity != SeverityError {
		t.Fatalf("expected one error acs-tdd-audit finding, got: %+v", got)
	}
	if !strings.Contains(got[0].Message, "(absent)") {
		t.Errorf("message should surface the absent phase, got: %q", got[0].Message)
	}
}

func TestAcsTDDAudit_AdvisoryFiresAsWarning(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001", TDD: "advisory",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "x", Status: "met", TDDPhase: "refactor"},
		},
	})
	got := acsTDDAudit(tr)
	if len(got) != 1 || got[0].Severity != SeverityWarning || got[0].Code != CodeACsTDDAudit {
		t.Errorf("expected one warning finding with code \"acs-tdd-audit\", got: %+v", got)
	}
}

// TestAcsTDDAudit_AdvisoryMetWithAbsentPhaseFiresAsWarning locks
// G-0286/M-0267 AC-2's explicit claim for the advisory side: same as
// TestAcsTDDAudit_MetWithAbsentPhaseFiresAsError but under tdd:
// advisory, so the finding is a warning, not an error. Completes the
// severity x absent-vs-wrong-value coverage matrix for acsTDDAudit.
func TestAcsTDDAudit_AdvisoryMetWithAbsentPhaseFiresAsWarning(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001", TDD: "advisory",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "x", Status: "met"},
		},
	})
	got := acsTDDAudit(tr)
	if len(got) != 1 || got[0].Code != CodeACsTDDAudit || got[0].Severity != SeverityWarning {
		t.Fatalf("expected one warning acs-tdd-audit finding, got: %+v", got)
	}
	if !strings.Contains(got[0].Message, "(absent)") {
		t.Errorf("message should surface the absent phase, got: %q", got[0].Message)
	}
}

func TestAcsTDDAudit_NoneSkipped(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001", TDD: "none",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "x", Status: "met", TDDPhase: "green"},
		},
	})
	if got := acsTDDAudit(tr); len(got) != 0 {
		t.Errorf("tdd: none should suppress audit, got: %+v", got)
	}
	// Same when tdd is absent.
	tr2 := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "x", Status: "met", TDDPhase: "red"},
		},
	})
	if got := acsTDDAudit(tr2); len(got) != 0 {
		t.Errorf("absent tdd: should suppress audit, got: %+v", got)
	}
}

func TestAcsTDDAudit_DonePassesUnderRequired(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001", TDD: "required",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "x", Status: "met", TDDPhase: "done"},
		},
	})
	if got := acsTDDAudit(tr); len(got) != 0 {
		t.Errorf("met+done under required should pass, got: %+v", got)
	}
}

func TestAcsTDDAudit_NonMetIgnored(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001", TDD: "required",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "x", Status: "open", TDDPhase: "red"},
			{ID: "AC-2", Title: "x", Status: "deferred", TDDPhase: "green"},
		},
	})
	if got := acsTDDAudit(tr); len(got) != 0 {
		t.Errorf("non-met statuses don't trigger the audit, got: %+v", got)
	}
}

// TestCheckRun_EmptyPhaseACsRaiseNoShapeOrTDDAudit pins M-0274/AC-3: a
// tdd: required milestone whose ACs rest at the pre-cycle empty phase
// raises neither acs-shape nor acs-tdd-audit, in both draft and
// in_progress. An absent phase is legal until an AC is promoted to met
// (G-0286) — the check-layer tolerance the M-0274 seeding fix depends on.
// Asserted at the aggregate seam (Run) rather than on the two rule
// functions in isolation, and across the milestone's whole pre-met
// lifecycle, so a future rule that grew a status branch could not silently
// start flagging the honest empty phase.
func TestCheckRun_EmptyPhaseACsRaiseNoShapeOrTDDAudit(t *testing.T) {
	t.Parallel()
	for _, status := range []string{"draft", "in_progress"} {
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			tr := makeTree(
				&entity.Entity{
					ID: "E-0001", Kind: entity.KindEpic, Title: "Foundations",
					Status: "active", Path: "work/epics/E-0001-foundations/epic.md",
				},
				&entity.Entity{
					ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
					Status: status, Parent: "E-0001", TDD: "required",
					Path: "work/epics/E-0001-foundations/M-0007-foo.md",
					ACs: []entity.AcceptanceCriterion{
						{ID: "AC-1", Title: "First", Status: "open"},  // pre-cycle empty phase
						{ID: "AC-2", Title: "Second", Status: "open"}, // pre-cycle empty phase
					},
				},
			)
			got := Run(tr, nil)
			if f := findingByCode(got, CodeACsShape, ""); f != nil {
				t.Errorf("empty-phase ACs under %s must not fire acs-shape, got: %+v", status, f)
			}
			if f := findingByCode(got, CodeACsTDDAudit, ""); f != nil {
				t.Errorf("empty-phase ACs under %s must not fire acs-tdd-audit, got: %+v", status, f)
			}
		})
	}
}

func TestMilestoneDoneIncompleteACs_FiresOnOpen(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "done", Parent: "E-0001",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "x", Status: "met"},
			{ID: "AC-2", Title: "y", Status: "open"},
			{ID: "AC-3", Title: "z", Status: "open"},
		},
	})
	got := milestoneDoneIncompleteACs(tr)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].EntityID != "M-0007" {
		t.Errorf("entityID = %q, want M-007", got[0].EntityID)
	}
	// Message should list both open AC ids.
	if !contains(got[0].Message, "AC-2") || !contains(got[0].Message, "AC-3") {
		t.Errorf("message should list both open AC ids: %q", got[0].Message)
	}
}

func TestMilestoneDoneIncompleteACs_TerminalACsAccepted(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "done", Parent: "E-0001",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "x", Status: "met"},
			{ID: "AC-2", Title: "y", Status: "deferred"},
			{ID: "AC-3", Title: "z", Status: "cancelled"},
		},
	})
	if got := milestoneDoneIncompleteACs(tr); len(got) != 0 {
		t.Errorf("met/deferred/cancelled are acceptable terminals, got: %+v", got)
	}
}

func TestMilestoneDoneIncompleteACs_NotDoneSkipped(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "x", Status: "open"},
		},
	})
	if got := milestoneDoneIncompleteACs(tr); len(got) != 0 {
		t.Errorf("non-done milestones don't trigger the rule, got: %+v", got)
	}
}

// TestMilestoneDoneZeroACs_FiresWarning pins M-0268/AC-3 (D-0039
// point 2): a non-archived milestone reaching `done` with an empty
// acs[] surfaces a warning-severity finding, extending the
// milestone-done-incomplete-acs pattern rather than replacing it —
// this is check-time only, never a verb-time refusal (D-0039
// explicitly rejects a second hard block at `done`).
func TestMilestoneDoneZeroACs_FiresWarning(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "done", Parent: "E-0001",
	})
	got := milestoneDoneZeroACs(tr)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].Severity != SeverityWarning {
		t.Errorf("severity = %q, want warning", got[0].Severity)
	}
	if got[0].EntityID != "M-0007" {
		t.Errorf("entityID = %q, want M-0007", got[0].EntityID)
	}
	if got[0].Code != CodeMilestoneDoneZeroACs {
		t.Errorf("code = %q, want %q", got[0].Code, CodeMilestoneDoneZeroACs)
	}
}

// TestMilestoneDoneZeroACs_PopulatedACsSilent: a done milestone that
// carries at least one AC (regardless of that AC's own status —
// that's milestoneDoneIncompleteACs's concern) never fires this rule.
func TestMilestoneDoneZeroACs_PopulatedACsSilent(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "done", Parent: "E-0001",
		ACs: []entity.AcceptanceCriterion{{ID: "AC-1", Title: "x", Status: "met"}},
	})
	if got := milestoneDoneZeroACs(tr); len(got) != 0 {
		t.Errorf("populated acs[] should not fire, got: %+v", got)
	}
}

// TestMilestoneDoneZeroACs_NotDoneSkipped: only status: done is in
// scope — a zero-AC milestone at draft/in_progress/cancelled is a
// different rule's concern (or none at all, at draft/in_progress).
func TestMilestoneDoneZeroACs_NotDoneSkipped(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001",
	})
	if got := milestoneDoneZeroACs(tr); len(got) != 0 {
		t.Errorf("non-done milestones don't trigger the rule, got: %+v", got)
	}
}

// TestMilestoneDoneZeroACs_ArchivedSilent mirrors the archive-scoping
// convention shared by every rule in this file (ADR-0004 §"Check shape
// rules"): a zero-AC done milestone that has been archived is
// historical state, not active drift.
func TestMilestoneDoneZeroACs_ArchivedSilent(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "done", Parent: "E-0001",
		Path: "work/epics/archive/E-0001-foo/M-0007-foo.md",
	})
	if got := milestoneDoneZeroACs(tr); len(got) != 0 {
		t.Errorf("archived milestones don't trigger the rule, got: %+v", got)
	}
}

// writeMilestoneFile writes a milestone markdown file at root/relPath
// with the given content, creating parent directories as needed.
// Shared setup for acsEmptyBodyOnStart tests, which — unlike most
// rules in this file — must read the body from disk (the AC body
// prose isn't part of the in-memory *entity.Entity frontmatter).
func writeMilestoneFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestAcsEmptyBodyOnStart_FiresOnInProgress pins M-0268/AC-4 (G-0216):
// a non-archived, in_progress milestone with an AC whose body is a
// title-only stub fires an error-severity finding.
func TestAcsEmptyBodyOnStart_FiresOnInProgress(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mPath := "work/epics/E-0001-foo/M-0007-foo.md"
	writeMilestoneFile(t, root, mPath, "---\n"+
		"id: M-0007\ntitle: Foo\nstatus: in_progress\nparent: E-0001\n"+
		"acs:\n  - id: AC-1\n    title: First\n    status: open\n---\n\n"+
		"## Acceptance criteria\n\n### AC-1 — First\n")

	tr := &tree.Tree{Root: root, Entities: []*entity.Entity{{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001",
		ACs:  []entity.AcceptanceCriterion{{ID: "AC-1", Title: "First", Status: "open"}},
		Path: mPath,
	}}}
	got := acsEmptyBodyOnStart(tr)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].Severity != SeverityError {
		t.Errorf("severity = %q, want error", got[0].Severity)
	}
	if got[0].EntityID != "M-0007/AC-1" {
		t.Errorf("entityID = %q, want M-0007/AC-1", got[0].EntityID)
	}
	if got[0].Code != CodeACsEmptyBodyOnStart {
		t.Errorf("code = %q, want %q", got[0].Code, CodeACsEmptyBodyOnStart)
	}
}

// TestAcsEmptyBodyOnStart_FiresOnDone: the finding is not silenced
// once the milestone reaches done — unlike the pre-existing
// entity-body-empty/ac warning, which the terminal-status lifecycle
// gate silences at done. AC-4 deliberately does NOT use that gate.
func TestAcsEmptyBodyOnStart_FiresOnDone(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mPath := "work/epics/E-0001-foo/M-0007-foo.md"
	writeMilestoneFile(t, root, mPath, "---\n"+
		"id: M-0007\ntitle: Foo\nstatus: done\nparent: E-0001\n"+
		"acs:\n  - id: AC-1\n    title: First\n    status: met\n---\n\n"+
		"## Acceptance criteria\n\n### AC-1 — First\n")

	tr := &tree.Tree{Root: root, Entities: []*entity.Entity{{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "done", Parent: "E-0001",
		ACs:  []entity.AcceptanceCriterion{{ID: "AC-1", Title: "First", Status: "met"}},
		Path: mPath,
	}}}
	if got := acsEmptyBodyOnStart(tr); len(got) != 1 {
		t.Errorf("expected 1 finding on a done milestone, got %d: %+v", len(got), got)
	}
}

// TestAcsEmptyBodyOnStart_DraftSkipped: a draft milestone is out of
// scope — an empty AC body pre-start is expected (aiwfx-plan-
// milestones ships shape first), the same lifecycle stance
// entity-body-empty/ac already takes.
func TestAcsEmptyBodyOnStart_DraftSkipped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mPath := "work/epics/E-0001-foo/M-0007-foo.md"
	writeMilestoneFile(t, root, mPath, "---\n"+
		"id: M-0007\ntitle: Foo\nstatus: draft\nparent: E-0001\n"+
		"acs:\n  - id: AC-1\n    title: First\n    status: open\n---\n\n"+
		"## Acceptance criteria\n\n### AC-1 — First\n")

	tr := &tree.Tree{Root: root, Entities: []*entity.Entity{{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "draft", Parent: "E-0001",
		ACs:  []entity.AcceptanceCriterion{{ID: "AC-1", Title: "First", Status: "open"}},
		Path: mPath,
	}}}
	if got := acsEmptyBodyOnStart(tr); len(got) != 0 {
		t.Errorf("draft milestones don't trigger the rule, got: %+v", got)
	}
}

// TestAcsEmptyBodyOnStart_ArchivedSilent: archived, forward-only per
// D-0039 point 3 — no separate grandfather/timestamp mechanism.
func TestAcsEmptyBodyOnStart_ArchivedSilent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mPath := "work/epics/archive/E-0001-foo/M-0007-foo.md"
	writeMilestoneFile(t, root, mPath, "---\n"+
		"id: M-0007\ntitle: Foo\nstatus: done\nparent: E-0001\n"+
		"acs:\n  - id: AC-1\n    title: First\n    status: met\n---\n\n"+
		"## Acceptance criteria\n\n### AC-1 — First\n")

	tr := &tree.Tree{Root: root, Entities: []*entity.Entity{{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "done", Parent: "E-0001",
		ACs:  []entity.AcceptanceCriterion{{ID: "AC-1", Title: "First", Status: "met"}},
		Path: mPath,
	}}}
	if got := acsEmptyBodyOnStart(tr); len(got) != 0 {
		t.Errorf("archived milestones don't trigger the rule, got: %+v", got)
	}
}

// TestAcsEmptyBodyOnStart_PopulatedBodySilent: an AC with real prose
// under its heading never fires.
func TestAcsEmptyBodyOnStart_PopulatedBodySilent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mPath := "work/epics/E-0001-foo/M-0007-foo.md"
	writeMilestoneFile(t, root, mPath, "---\n"+
		"id: M-0007\ntitle: Foo\nstatus: in_progress\nparent: E-0001\n"+
		"acs:\n  - id: AC-1\n    title: First\n    status: open\n---\n\n"+
		"## Acceptance criteria\n\n### AC-1 — First\n\nReal prose.\n")

	tr := &tree.Tree{Root: root, Entities: []*entity.Entity{{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001",
		ACs:  []entity.AcceptanceCriterion{{ID: "AC-1", Title: "First", Status: "open"}},
		Path: mPath,
	}}}
	if got := acsEmptyBodyOnStart(tr); len(got) != 0 {
		t.Errorf("populated AC body should not fire, got: %+v", got)
	}
}

// TestAcsEmptyBodyOnStart_CancelledACSkipped: a cancelled AC's body
// isn't a live contract anymore — matches entity-body-empty/ac's own
// exclusion.
func TestAcsEmptyBodyOnStart_CancelledACSkipped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mPath := "work/epics/E-0001-foo/M-0007-foo.md"
	writeMilestoneFile(t, root, mPath, "---\n"+
		"id: M-0007\ntitle: Foo\nstatus: in_progress\nparent: E-0001\n"+
		"acs:\n  - id: AC-1\n    title: First\n    status: cancelled\n---\n\n"+
		"## Acceptance criteria\n\n### AC-1 — First\n")

	tr := &tree.Tree{Root: root, Entities: []*entity.Entity{{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001",
		ACs:  []entity.AcceptanceCriterion{{ID: "AC-1", Title: "First", Status: "cancelled"}},
		Path: mPath,
	}}}
	if got := acsEmptyBodyOnStart(tr); len(got) != 0 {
		t.Errorf("cancelled AC should not fire, got: %+v", got)
	}
}

// TestAcsEmptyBodyOnStart_MissingHeadingSkipped: no `### AC-N` heading
// at all is acs-body-coherence/missing-heading's concern, not this
// one — matches AC-2's own verb-time carve-out.
func TestAcsEmptyBodyOnStart_MissingHeadingSkipped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mPath := "work/epics/E-0001-foo/M-0007-foo.md"
	writeMilestoneFile(t, root, mPath, "---\n"+
		"id: M-0007\ntitle: Foo\nstatus: in_progress\nparent: E-0001\n"+
		"acs:\n  - id: AC-1\n    title: First\n    status: open\n---\n\n"+
		"## Acceptance criteria\n")

	tr := &tree.Tree{Root: root, Entities: []*entity.Entity{{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001",
		ACs:  []entity.AcceptanceCriterion{{ID: "AC-1", Title: "First", Status: "open"}},
		Path: mPath,
	}}}
	if got := acsEmptyBodyOnStart(tr); len(got) != 0 {
		t.Errorf("missing heading should not fire this rule, got: %+v", got)
	}
}

// TestAcsEmptyBodyOnStart_EmptyACIDSkipped pins the defensive half of
// the loop's compound skip condition: a frontmatter entry with no id
// at all (a shape defect acs-shape/id already flags separately) must
// not panic or misbehave here — it is simply skipped.
func TestAcsEmptyBodyOnStart_EmptyACIDSkipped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mPath := "work/epics/E-0001-foo/M-0007-foo.md"
	writeMilestoneFile(t, root, mPath, "---\n"+
		"id: M-0007\ntitle: Foo\nstatus: in_progress\nparent: E-0001\n"+
		"acs:\n  - id: ''\n    title: First\n    status: open\n---\n\n"+
		"## Acceptance criteria\n")

	tr := &tree.Tree{Root: root, Entities: []*entity.Entity{{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001",
		ACs:  []entity.AcceptanceCriterion{{ID: "", Title: "First", Status: "open"}},
		Path: mPath,
	}}}
	if got := acsEmptyBodyOnStart(tr); len(got) != 0 {
		t.Errorf("empty AC id should not fire this rule, got: %+v", got)
	}
}

// TestCheckRun_DraftMilestoneZeroACsWarns pins M-0275/AC-1: a non-archived
// draft milestone with zero AC entities raises a warning-severity
// milestone-draft-incomplete-acs finding through the check aggregate, and a
// draft milestone that already has ACs does not. Warning, not error — draft
// is a legitimate mid-planning state (D-0047/G-0440), so this surfaces the
// missing-contract gap without blocking the milestone from resting in draft.
func TestCheckRun_DraftMilestoneZeroACsWarns(t *testing.T) {
	t.Parallel()
	epic := &entity.Entity{
		ID: "E-0001", Kind: entity.KindEpic, Title: "Foundations",
		Status: "active", Path: "work/epics/E-0001-foundations/epic.md",
	}

	// Non-archived draft milestone with zero ACs → warning fires.
	bare := makeTree(epic, &entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Bare", Status: "draft", Parent: "E-0001",
		Path: "work/epics/E-0001-foundations/M-0007-bare.md",
	})
	f := findingByCode(Run(bare, nil), CodeMilestoneDraftIncompleteACs, "")
	if f == nil {
		t.Fatal("AC-1: expected milestone-draft-incomplete-acs finding for a zero-AC draft milestone")
	}
	if f.Severity != SeverityWarning {
		t.Errorf("AC-1: severity = %q, want warning", f.Severity)
	}

	// Draft milestone that already has ACs → no finding.
	populated := makeTree(epic, &entity.Entity{
		ID: "M-0008", Kind: entity.KindMilestone, Title: "Populated", Status: "draft", Parent: "E-0001",
		Path: "work/epics/E-0001-foundations/M-0008-populated.md",
		ACs:  []entity.AcceptanceCriterion{{ID: "AC-1", Title: "Something observable", Status: "open"}},
	})
	if f := findingByCode(Run(populated, nil), CodeMilestoneDraftIncompleteACs, ""); f != nil {
		t.Errorf("AC-1: a draft milestone with ACs must not fire the finding; got %+v", f)
	}

	// Archive-scoped: an archived zero-AC draft milestone stays silent
	// (covers the IsArchivedPath skip; AC-3 pins archive-scoping in full).
	archived := makeTree(epic, &entity.Entity{
		ID: "M-0009", Kind: entity.KindMilestone, Title: "Archived", Status: "draft", Parent: "E-0001",
		Path: "work/epics/archive/E-0001-foundations/M-0009-archived.md",
	})
	if f := findingByCode(Run(archived, nil), CodeMilestoneDraftIncompleteACs, ""); f != nil {
		t.Errorf("AC-1: an archived zero-AC draft milestone must not fire the finding; got %+v", f)
	}
}

// TestCheckRun_DraftMilestoneEmptyACBodyWarns pins M-0275/AC-2: a non-archived
// draft milestone whose AC carries a `### AC-N` heading but no body prose raises
// a warning-severity milestone-draft-incomplete-acs/empty-body finding through
// the check aggregate — the draft-rung, warning-severity mirror of
// acsEmptyBodyOnStart's in_progress/done error. A populated AC body stays
// silent, and an archived milestone with the same empty body stays silent.
func TestCheckRun_DraftMilestoneEmptyACBodyWarns(t *testing.T) {
	t.Parallel()

	// Draft milestone, AC-1 heading present but body empty → warning fires,
	// keyed to the composite AC id.
	root := t.TempDir()
	mPath := "work/epics/E-0001-foundations/M-0007-empty.md"
	writeMilestoneFile(t, root, mPath, "---\n"+
		"id: M-0007\ntitle: Foo\nstatus: draft\nparent: E-0001\n"+
		"acs:\n  - id: AC-1\n    title: First\n    status: open\n---\n\n"+
		"## Acceptance criteria\n\n### AC-1 — First\n")
	tr := &tree.Tree{Root: root, Entities: []*entity.Entity{{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "draft", Parent: "E-0001",
		ACs:  []entity.AcceptanceCriterion{{ID: "AC-1", Title: "First", Status: "open"}},
		Path: mPath,
	}}}
	f := findingByCode(Run(tr, nil), CodeMilestoneDraftIncompleteACs, "empty-body")
	if f == nil {
		t.Fatal("AC-2: expected milestone-draft-incomplete-acs/empty-body finding for a draft milestone with an empty AC body")
	}
	if f.Severity != SeverityWarning {
		t.Errorf("AC-2: severity = %q, want warning", f.Severity)
	}
	if f.EntityID != "M-0007/AC-1" {
		t.Errorf("AC-2: entityID = %q, want M-0007/AC-1", f.EntityID)
	}

	// Populated AC body → no empty-body finding.
	root2 := t.TempDir()
	writeMilestoneFile(t, root2, mPath, "---\n"+
		"id: M-0007\ntitle: Foo\nstatus: draft\nparent: E-0001\n"+
		"acs:\n  - id: AC-1\n    title: First\n    status: open\n---\n\n"+
		"## Acceptance criteria\n\n### AC-1 — First\n\nReal prose.\n")
	tr2 := &tree.Tree{Root: root2, Entities: []*entity.Entity{{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "draft", Parent: "E-0001",
		ACs:  []entity.AcceptanceCriterion{{ID: "AC-1", Title: "First", Status: "open"}},
		Path: mPath,
	}}}
	if f := findingByCode(Run(tr2, nil), CodeMilestoneDraftIncompleteACs, "empty-body"); f != nil {
		t.Errorf("AC-2: a populated AC body must not fire empty-body; got %+v", f)
	}

	// Archived draft milestone with the same empty body → silent (archive-scoped).
	root3 := t.TempDir()
	aPath := "work/epics/archive/E-0001-foundations/M-0009-archived.md"
	writeMilestoneFile(t, root3, aPath, "---\n"+
		"id: M-0009\ntitle: Foo\nstatus: draft\nparent: E-0001\n"+
		"acs:\n  - id: AC-1\n    title: First\n    status: open\n---\n\n"+
		"## Acceptance criteria\n\n### AC-1 — First\n")
	tr3 := &tree.Tree{Root: root3, Entities: []*entity.Entity{{
		ID: "M-0009", Kind: entity.KindMilestone, Title: "Foo",
		Status: "draft", Parent: "E-0001",
		ACs:  []entity.AcceptanceCriterion{{ID: "AC-1", Title: "First", Status: "open"}},
		Path: aPath,
	}}}
	if f := findingByCode(Run(tr3, nil), CodeMilestoneDraftIncompleteACs, "empty-body"); f != nil {
		t.Errorf("AC-2: an archived draft milestone with an empty AC body must stay silent; got %+v", f)
	}
}

// TestCheckRun_DraftMilestoneEmptyACBody_CarveOuts pins M-0275/AC-2's skip
// branches, mirroring acsEmptyBodyOnStart's own carve-outs one FSM stage
// earlier: a cancelled AC, an AC with no frontmatter id, and an AC with no
// `### AC-N` body heading at all each leave empty-body silent (the last is
// acs-body-coherence/missing-heading's concern, not this rule's).
func TestCheckRun_DraftMilestoneEmptyACBody_CarveOuts(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		acsYAML string
		acs     []entity.AcceptanceCriterion
		body    string
	}{
		{
			name:    "cancelled AC skipped",
			acsYAML: "acs:\n  - id: AC-1\n    title: First\n    status: cancelled\n",
			acs:     []entity.AcceptanceCriterion{{ID: "AC-1", Title: "First", Status: "cancelled"}},
			body:    "## Acceptance criteria\n\n### AC-1 — First\n",
		},
		{
			name:    "empty AC id skipped",
			acsYAML: "acs:\n  - id: ''\n    title: First\n    status: open\n",
			acs:     []entity.AcceptanceCriterion{{ID: "", Title: "First", Status: "open"}},
			body:    "## Acceptance criteria\n",
		},
		{
			name:    "missing heading skipped",
			acsYAML: "acs:\n  - id: AC-1\n    title: First\n    status: open\n",
			acs:     []entity.AcceptanceCriterion{{ID: "AC-1", Title: "First", Status: "open"}},
			body:    "## Acceptance criteria\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			mPath := "work/epics/E-0001-foundations/M-0007-foo.md"
			writeMilestoneFile(t, root, mPath, "---\n"+
				"id: M-0007\ntitle: Foo\nstatus: draft\nparent: E-0001\n"+
				tc.acsYAML+"---\n\n"+tc.body)
			tr := &tree.Tree{Root: root, Entities: []*entity.Entity{{
				ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
				Status: "draft", Parent: "E-0001",
				ACs:  tc.acs,
				Path: mPath,
			}}}
			if f := findingByCode(Run(tr, nil), CodeMilestoneDraftIncompleteACs, "empty-body"); f != nil {
				t.Errorf("empty-body must stay silent for %s; got %+v", tc.name, f)
			}
		})
	}
}

func TestMilestoneCancelledIncompleteACs_FiresOnOpen(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "cancelled", Parent: "E-0001",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "x", Status: "met"},
			{ID: "AC-2", Title: "y", Status: "open"},
			{ID: "AC-3", Title: "z", Status: "open"},
		},
	})
	got := milestoneCancelledIncompleteACs(tr)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
	}
	if got[0].EntityID != "M-0007" {
		t.Errorf("entityID = %q, want M-007", got[0].EntityID)
	}
	// Message should list both open AC ids.
	if !contains(got[0].Message, "AC-2") || !contains(got[0].Message, "AC-3") {
		t.Errorf("message should list both open AC ids: %q", got[0].Message)
	}
}

func TestMilestoneCancelledIncompleteACs_TerminalACsAccepted(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "cancelled", Parent: "E-0001",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "x", Status: "met"},
			{ID: "AC-2", Title: "y", Status: "deferred"},
			{ID: "AC-3", Title: "z", Status: "cancelled"},
		},
	})
	if got := milestoneCancelledIncompleteACs(tr); len(got) != 0 {
		t.Errorf("met/deferred/cancelled are acceptable terminals, got: %+v", got)
	}
}

func TestMilestoneCancelledIncompleteACs_NotCancelledSkipped(t *testing.T) {
	t.Parallel()
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "x", Status: "open"},
		},
	})
	if got := milestoneCancelledIncompleteACs(tr); len(got) != 0 {
		t.Errorf("non-cancelled milestones don't trigger the rule, got: %+v", got)
	}
}

func TestAcsBodyCoherence_PairsByID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mPath := "work/epics/E-01-foundations/M-007-warnings.md"
	abs := filepath.Join(root, filepath.FromSlash(mPath))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `---
id: M-007
title: Engine warning surface
status: in_progress
parent: E-03
tdd: required
acs:
  - id: AC-1
    title: First
    status: open
    tdd_phase: red
  - id: AC-2
    title: Second
    status: open
    tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — First
prose
### AC-2: Second
prose
`
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tr := &tree.Tree{
		Root: root,
		Entities: []*entity.Entity{{
			ID: "M-0007", Kind: entity.KindMilestone, Title: "Engine",
			Status: "in_progress", Parent: "E-0003", TDD: "required",
			ACs: []entity.AcceptanceCriterion{
				{ID: "AC-1", Title: "First", Status: "open", TDDPhase: "red"},
				{ID: "AC-2", Title: "Second", Status: "open", TDDPhase: "red"},
			},
			Path: mPath,
		}},
	}
	if got := acsBodyCoherence(tr); len(got) != 0 {
		t.Errorf("paired body+frontmatter should produce no findings, got: %+v", got)
	}
}

func TestAcsBodyCoherence_MissingHeading(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mPath := "work/epics/E-01/M-007.md"
	abs := filepath.Join(root, filepath.FromSlash(mPath))
	_ = os.MkdirAll(filepath.Dir(abs), 0o755)
	content := `---
id: M-007
title: Foo
status: in_progress
parent: E-01
acs:
  - id: AC-1
    title: First
    status: open
  - id: AC-2
    title: Second
    status: open
---

## Acceptance criteria

### AC-1 — First
only AC-1 has a body heading
`
	_ = os.WriteFile(abs, []byte(content), 0o644)

	tr := &tree.Tree{
		Root: root,
		Entities: []*entity.Entity{{
			ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
			Status: "in_progress", Parent: "E-0001",
			ACs: []entity.AcceptanceCriterion{
				{ID: "AC-1", Title: "First", Status: "open"},
				{ID: "AC-2", Title: "Second", Status: "open"},
			},
			Path: mPath,
		}},
	}
	got := acsBodyCoherence(tr)
	f := findingByCode(got, CodeACsBodyCoherence, "missing-heading")
	if f == nil {
		t.Fatalf("expected acs-body-coherence/missing-heading; got: %+v", got)
	}
	if f.EntityID != "M-0007/AC-2" {
		t.Errorf("entityID = %q, want M-007/AC-2", f.EntityID)
	}
}

func TestAcsBodyCoherence_OrphanHeading(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mPath := "work/epics/E-01/M-007.md"
	abs := filepath.Join(root, filepath.FromSlash(mPath))
	_ = os.MkdirAll(filepath.Dir(abs), 0o755)
	content := `---
id: M-007
title: Foo
status: in_progress
parent: E-01
acs:
  - id: AC-1
    title: First
    status: open
---

## Acceptance criteria

### AC-1 — First
prose
### AC-2 — Orphan
no frontmatter entry for this one
`
	_ = os.WriteFile(abs, []byte(content), 0o644)

	tr := &tree.Tree{
		Root: root,
		Entities: []*entity.Entity{{
			ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
			Status: "in_progress", Parent: "E-0001",
			ACs: []entity.AcceptanceCriterion{
				{ID: "AC-1", Title: "First", Status: "open"},
			},
			Path: mPath,
		}},
	}
	got := acsBodyCoherence(tr)
	if findingByCode(got, CodeACsBodyCoherence, "orphan-heading") == nil {
		t.Errorf("expected acs-body-coherence/orphan-heading; got: %+v", got)
	}
}

// TestScanACHeadings_CountsDuplicates pins the count semantics G-0247
// relies on: scanACHeadings reports how many heading lines carry each
// id, not just whether the id is present, so acsBodyCoherence can flag
// a duplicated `### AC-N` heading.
func TestScanACHeadings_CountsDuplicates(t *testing.T) {
	t.Parallel()
	body := []byte("### AC-1 — first\n### AC-1 — dup\n### AC-2 — second\n")
	got := scanACHeadings(body)
	if got["AC-1"] != 2 {
		t.Errorf("AC-1 count = %d, want 2", got["AC-1"])
	}
	if got["AC-2"] != 1 {
		t.Errorf("AC-2 count = %d, want 1", got["AC-2"])
	}
}

// TestAcsBodyCoherence_DuplicateHeading is the G-0247 check-side guard:
// a body with two `### AC-1` headings for an id that IS in frontmatter
// is neither missing- nor orphan-heading, so the set-collapse used to
// pass it clean. The duplicate-heading subcode flags it.
func TestAcsBodyCoherence_DuplicateHeading(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mPath := "work/epics/E-01/M-007.md"
	abs := filepath.Join(root, filepath.FromSlash(mPath))
	_ = os.MkdirAll(filepath.Dir(abs), 0o755)
	content := `---
id: M-007
title: Foo
status: in_progress
parent: E-01
acs:
  - id: AC-1
    title: First
    status: open
---

## Acceptance criteria

### AC-1 — <observable behavior>

placeholder prose

### AC-1 — First

the real one
`
	_ = os.WriteFile(abs, []byte(content), 0o644)

	tr := &tree.Tree{
		Root: root,
		Entities: []*entity.Entity{{
			ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
			Status: "in_progress", Parent: "E-0001",
			ACs: []entity.AcceptanceCriterion{
				{ID: "AC-1", Title: "First", Status: "open"},
			},
			Path: mPath,
		}},
	}
	got := acsBodyCoherence(tr)
	f := findingByCode(got, CodeACsBodyCoherence, "duplicate-heading")
	if f == nil {
		t.Fatalf("expected acs-body-coherence/duplicate-heading; got: %+v", got)
	}
	if f.EntityID != "M-0007/AC-1" {
		t.Errorf("entityID = %q, want M-0007/AC-1", f.EntityID)
	}
	// A duplicate of a frontmatter id must not also masquerade as a
	// missing- or orphan-heading.
	if findingByCode(got, CodeACsBodyCoherence, "missing-heading") != nil {
		t.Errorf("duplicate of a frontmatter id should not also be missing-heading: %+v", got)
	}
	if findingByCode(got, CodeACsBodyCoherence, "orphan-heading") != nil {
		t.Errorf("duplicate of a frontmatter id should not also be orphan-heading: %+v", got)
	}
}

func TestAcsBodyCoherence_PermissiveSeparator(t *testing.T) {
	t.Parallel()
	// All four heading shapes are accepted; the coherence check pairs
	// by id only and emits no findings on a well-paired set.
	root := t.TempDir()
	mPath := "work/epics/E-01/M-007.md"
	abs := filepath.Join(root, filepath.FromSlash(mPath))
	_ = os.MkdirAll(filepath.Dir(abs), 0o755)
	content := `---
id: M-007
title: Foo
status: in_progress
parent: E-01
acs:
  - id: AC-1
    title: emdash
    status: open
  - id: AC-2
    title: hyphen
    status: open
  - id: AC-3
    title: colon
    status: open
  - id: AC-4
    title: id only
    status: open
---

### AC-1 — em-dash form
### AC-2 - hyphen form
### AC-3: colon form
### AC-4
`
	_ = os.WriteFile(abs, []byte(content), 0o644)

	tr := &tree.Tree{
		Root: root,
		Entities: []*entity.Entity{{
			ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
			Status: "in_progress", Parent: "E-0001",
			ACs: []entity.AcceptanceCriterion{
				{ID: "AC-1", Title: "emdash", Status: "open"},
				{ID: "AC-2", Title: "hyphen", Status: "open"},
				{ID: "AC-3", Title: "colon", Status: "open"},
				{ID: "AC-4", Title: "id only", Status: "open"},
			},
			Path: mPath,
		}},
	}
	if got := acsBodyCoherence(tr); len(got) != 0 {
		t.Errorf("permissive separator forms should pair cleanly, got: %+v", got)
	}
}

func TestAcsBodyCoherence_TitleTextNotChecked(t *testing.T) {
	t.Parallel()
	// Frontmatter title and body heading title disagree — kernel
	// stays blind. Pairs by id only.
	root := t.TempDir()
	mPath := "work/epics/E-01/M-007.md"
	abs := filepath.Join(root, filepath.FromSlash(mPath))
	_ = os.MkdirAll(filepath.Dir(abs), 0o755)
	content := `---
id: M-007
title: Foo
status: in_progress
parent: E-01
acs:
  - id: AC-1
    title: Frontmatter title
    status: open
---

### AC-1 — Body title is different
prose
`
	_ = os.WriteFile(abs, []byte(content), 0o644)

	tr := &tree.Tree{
		Root: root,
		Entities: []*entity.Entity{{
			ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
			Status: "in_progress", Parent: "E-0001",
			ACs:  []entity.AcceptanceCriterion{{ID: "AC-1", Title: "Frontmatter title", Status: "open"}},
			Path: mPath,
		}},
	}
	if got := acsBodyCoherence(tr); len(got) != 0 {
		t.Errorf("title-text mismatch should NOT trigger coherence; pairs by id only. got: %+v", got)
	}
}

func TestRefsResolve_CompositeIDInAddressedBy(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Status: "active", Path: "epic.md"},
		&entity.Entity{
			ID: "M-0007", Kind: entity.KindMilestone, Status: "in_progress", Parent: "E-0001", Path: "m.md",
			ACs: []entity.AcceptanceCriterion{{ID: "AC-1", Title: "First", Status: "open"}},
		},
		// Gap addressing AC-1 via the composite id.
		&entity.Entity{
			ID: "G-0001", Kind: entity.KindGap, Status: "open",
			AddressedBy: []string{"M-0007/AC-1"},
			Path:        "gap.md",
		},
	)
	if got := refsResolve(tr); len(got) != 0 {
		t.Errorf("valid composite id should resolve cleanly, got: %+v", got)
	}
}

func TestRefsResolve_CompositeUnresolvedMilestone(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{
			ID: "G-0001", Kind: entity.KindGap, Status: "open",
			AddressedBy: []string{"M-0007/AC-1"},
			Path:        "gap.md",
		},
	)
	got := refsResolve(tr)
	if findingByCode(got, CodeRefsResolve, "unresolved-milestone") == nil {
		t.Errorf("expected refs-resolve/unresolved-milestone; got: %+v", got)
	}
}

func TestRefsResolve_CompositeUnresolvedAC(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Status: "active", Path: "epic.md"},
		&entity.Entity{
			ID: "M-0007", Kind: entity.KindMilestone, Status: "in_progress", Parent: "E-0001", Path: "m.md",
			ACs: []entity.AcceptanceCriterion{{ID: "AC-1", Title: "First", Status: "open"}},
		},
		&entity.Entity{
			ID: "D-0001", Kind: entity.KindDecision, Status: "accepted",
			RelatesTo: []string{"M-0007/AC-99"},
			Path:      "d.md",
		},
	)
	got := refsResolve(tr)
	if findingByCode(got, CodeRefsResolve, "unresolved-ac") == nil {
		t.Errorf("expected refs-resolve/unresolved-ac; got: %+v", got)
	}
}

func TestRefsResolve_CompositeRejectedOnClosedTargetField(t *testing.T) {
	t.Parallel()
	// milestone.parent is a closed-target field (epic only).
	// Using a composite id there should produce a regular `unresolved`
	// finding (the composite isn't in the index), not a special path.
	tr := makeTree(
		&entity.Entity{
			ID: "M-0007", Kind: entity.KindMilestone, Status: "in_progress",
			Parent: "M-0001/AC-1", // composite — not allowed for parent
			Path:   "m.md",
		},
	)
	got := refsResolve(tr)
	if findingByCode(got, CodeRefsResolve, "unresolved") == nil {
		t.Errorf("expected refs-resolve/unresolved on closed-target composite; got: %+v", got)
	}
}

// contains is a tiny substring helper local to this file.
func contains(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
