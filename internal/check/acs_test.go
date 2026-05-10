package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/tree"
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
			tr := makeTree(&entity.Entity{
				ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
				Status: "in_progress", Parent: "E-0001", ACs: tt.acs,
			})
			got := acsShape(tr)
			if findingByCode(got, "acs-shape", tt.wantSub) == nil {
				t.Errorf("expected acs-shape/%s; got: %+v", tt.wantSub, got)
			}
		})
	}
}

func TestAcsShape_TitleStatusTDDPhaseAndPolicy(t *testing.T) {
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
			wantCode: "acs-shape",
			wantSub:  "title",
		},
		{
			name:     "status missing",
			ac:       entity.AcceptanceCriterion{ID: "AC-1", Title: "x", Status: ""},
			wantCode: "acs-shape",
			wantSub:  "status",
		},
		{
			name:     "status invalid",
			ac:       entity.AcceptanceCriterion{ID: "AC-1", Title: "x", Status: "frobnicate"},
			wantCode: "acs-shape",
			wantSub:  "status",
		},
		{
			name:     "tdd_phase invalid",
			ac:       entity.AcceptanceCriterion{ID: "AC-1", Title: "x", Status: "open", TDDPhase: "blue"},
			wantCode: "acs-shape",
			wantSub:  "tdd-phase",
		},
		{
			name:     "tdd_phase required but absent",
			ac:       entity.AcceptanceCriterion{ID: "AC-1", Title: "x", Status: "open"},
			tdd:      "required",
			wantCode: "acs-shape",
			wantSub:  "tdd-phase",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestAcsShape_TDDPolicyInvalid(t *testing.T) {
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001", TDD: "strict",
	})
	got := acsShape(tr)
	if findingByCode(got, "acs-shape", "tdd-policy") == nil {
		t.Errorf("expected acs-shape/tdd-policy; got: %+v", got)
	}
}

func TestAcsShape_NonMilestoneSkipped(t *testing.T) {
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
	if findingByCode(got, "acs-shape", "id") == nil {
		t.Errorf("expected an acs-shape/id finding for the position-3 duplicate AC-2; got: %+v", got)
	}
}

// TestAcsTitleProse_FlagsLongTitle covers the standing-check half of
// G20: an AC that landed via hand-edit (or pre-G20 tooling) with a
// prose-y title surfaces as a warning so the human knows to refactor.
func TestAcsTitleProse_FlagsLongTitle(t *testing.T) {
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "**Full embedment inventory.** A machine-reviewable table enumerates every rule.", Status: "open"},
		},
	})
	got := acsTitleProse(tr)
	f := findingByCode(got, "acs-title-prose", "")
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
	if got[0].Code != "acs-tdd-audit" {
		t.Errorf("code = %q, want \"acs-tdd-audit\"", got[0].Code)
	}
	if got[0].Severity != SeverityError {
		t.Errorf("severity = %q, want error", got[0].Severity)
	}
	if got[0].EntityID != "M-0007/AC-1" {
		t.Errorf("entityID = %q, want M-007/AC-1", got[0].EntityID)
	}
}

func TestAcsTDDAudit_AdvisoryFiresAsWarning(t *testing.T) {
	tr := makeTree(&entity.Entity{
		ID: "M-0007", Kind: entity.KindMilestone, Title: "Foo",
		Status: "in_progress", Parent: "E-0001", TDD: "advisory",
		ACs: []entity.AcceptanceCriterion{
			{ID: "AC-1", Title: "x", Status: "met", TDDPhase: "refactor"},
		},
	})
	got := acsTDDAudit(tr)
	if len(got) != 1 || got[0].Severity != SeverityWarning || got[0].Code != "acs-tdd-audit" {
		t.Errorf("expected one warning finding with code \"acs-tdd-audit\", got: %+v", got)
	}
}

func TestAcsTDDAudit_NoneSkipped(t *testing.T) {
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

func TestMilestoneDoneIncompleteACs_FiresOnOpen(t *testing.T) {
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

func TestAcsBodyCoherence_PairsByID(t *testing.T) {
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
	f := findingByCode(got, "acs-body-coherence", "missing-heading")
	if f == nil {
		t.Fatalf("expected acs-body-coherence/missing-heading; got: %+v", got)
	}
	if f.EntityID != "M-0007/AC-2" {
		t.Errorf("entityID = %q, want M-007/AC-2", f.EntityID)
	}
}

func TestAcsBodyCoherence_OrphanHeading(t *testing.T) {
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
	if findingByCode(got, "acs-body-coherence", "orphan-heading") == nil {
		t.Errorf("expected acs-body-coherence/orphan-heading; got: %+v", got)
	}
}

func TestAcsBodyCoherence_PermissiveSeparator(t *testing.T) {
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
	tr := makeTree(
		&entity.Entity{
			ID: "G-0001", Kind: entity.KindGap, Status: "open",
			AddressedBy: []string{"M-0007/AC-1"},
			Path:        "gap.md",
		},
	)
	got := refsResolve(tr)
	if findingByCode(got, "refs-resolve", "unresolved-milestone") == nil {
		t.Errorf("expected refs-resolve/unresolved-milestone; got: %+v", got)
	}
}

func TestRefsResolve_CompositeUnresolvedAC(t *testing.T) {
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
	if findingByCode(got, "refs-resolve", "unresolved-ac") == nil {
		t.Errorf("expected refs-resolve/unresolved-ac; got: %+v", got)
	}
}

func TestRefsResolve_CompositeRejectedOnClosedTargetField(t *testing.T) {
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
	if findingByCode(got, "refs-resolve", "unresolved") == nil {
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
