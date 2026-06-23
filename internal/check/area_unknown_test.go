package check

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// TestAreaUnknown_DeclaredArea_NoFinding pins M-0172/AC-1: an entity whose
// `area` is a member of the declared set produces no finding.
func TestAreaUnknown_DeclaredArea_NoFinding(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Area: "platform"},
		&entity.Entity{ID: "G-0001", Kind: entity.KindGap, Area: "billing"},
	)
	got := AreaUnknown(tr, []string{"platform", "billing"})
	if len(got) != 0 {
		t.Fatalf("declared areas should produce no findings, got %+v", got)
	}
}

// TestAreaUnknown_UndeclaredArea_Fires pins M-0172/AC-2: a present,
// non-empty area not in the declared set fires exactly one warning whose
// message names the entity id, the offending value, and the declared set.
func TestAreaUnknown_UndeclaredArea_Fires(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "G-0001", Kind: entity.KindGap, Path: "work/gaps/G-0001-x.md", Area: "platfrm"},
	)
	got := AreaUnknown(tr, []string{"platform", "billing"})
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 finding, got %d: %+v", len(got), got)
	}
	f := got[0]
	if f.Code != CodeAreaUnknown {
		t.Errorf("Code = %q, want %q", f.Code, CodeAreaUnknown)
	}
	if f.Severity != SeverityWarning {
		t.Errorf("Severity = %q, want %q", f.Severity, SeverityWarning)
	}
	if f.EntityID != "G-0001" {
		t.Errorf("EntityID = %q, want G-0001", f.EntityID)
	}
	if f.Field != "area" {
		t.Errorf("Field = %q, want area", f.Field)
	}
	// Message must name the id, the offending value, and each declared member.
	for _, want := range []string{"G-0001", "platfrm", "platform", "billing"} {
		if !strings.Contains(f.Message, want) {
			t.Errorf("Message %q does not contain %q", f.Message, want)
		}
	}
}

// TestAreaUnknown_AbsentOrEmpty_NeverFires pins M-0172/AC-3: absence is
// never evaluated. Absent, explicit-null, and empty `area` all deserialize
// to "" and never fire, even with a declared set present.
func TestAreaUnknown_AbsentOrEmpty_NeverFires(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Area: ""},
		&entity.Entity{ID: "G-0001", Kind: entity.KindGap, Area: ""},
	)
	got := AreaUnknown(tr, []string{"platform", "billing"})
	if len(got) != 0 {
		t.Fatalf("empty/absent area must never fire, got %+v", got)
	}
}

// TestAreaUnknown_NoAreasBlock_Inert pins M-0172/AC-4: with no declared
// set (no areas block, nil or empty), the rule is inert regardless of
// entity area values. Complements M-0171/AC-4's metamorphic guard that
// check.Run itself stays area-agnostic.
func TestAreaUnknown_NoAreasBlock_Inert(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Area: "anything"},
		&entity.Entity{ID: "G-0001", Kind: entity.KindGap, Area: "whatever"},
	)
	for _, declared := range [][]string{nil, {}} {
		got := AreaUnknown(tr, declared)
		if len(got) != 0 {
			t.Fatalf("declared=%v: rule must be inert, got %+v", declared, got)
		}
	}
}

// TestAreaUnknown_ArchivedEntity_NeverFires pins M-0172/AC-5: an entity
// under a per-kind archive/ subdirectory never fires (ADR-0004 §"check
// shape rules"); its active-tree twin with the same bad area does.
func TestAreaUnknown_ArchivedEntity_NeverFires(t *testing.T) {
	t.Parallel()
	active := &entity.Entity{ID: "G-0001", Kind: entity.KindGap, Path: "work/gaps/G-0001-x.md", Area: "bogus"}
	archived := &entity.Entity{ID: "G-0002", Kind: entity.KindGap, Path: "work/gaps/archive/G-0002-y.md", Area: "bogus"}
	got := AreaUnknown(makeTree(active, archived), []string{"platform"})
	if len(got) != 1 {
		t.Fatalf("expected only the active entity to fire, got %d: %+v", len(got), got)
	}
	if got[0].EntityID != "G-0001" {
		t.Errorf("fired for %q, want active G-0001", got[0].EntityID)
	}
}
