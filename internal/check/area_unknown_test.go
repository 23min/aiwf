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

// TestAreaUnknown_GlobalIsKnown pins M-0184/AC-2: the reserved `global`
// sentinel is a valid area value even when it is NOT among the declared
// members, so a global-tagged entity never fires area-unknown. The
// undeclared sibling on the same tree still fires, proving the rule still
// polices typos.
func TestAreaUnknown_GlobalIsKnown(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "ADR-0001", Kind: entity.KindADR, Path: "docs/adr/ADR-0001-x.md", Area: entity.AreaGlobal},
		&entity.Entity{ID: "G-0001", Kind: entity.KindGap, Path: "work/gaps/G-0001-x.md", Area: "platfrm"},
	)
	got := AreaUnknown(tr, []string{"platform", "billing"})
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 finding (the typo), got %d: %+v", len(got), got)
	}
	if got[0].EntityID != "G-0001" {
		t.Errorf("fired for %q, want only the undeclared G-0001 (global is recognized)", got[0].EntityID)
	}
}

// TestAreaUnknown_GlobalNotBlockedUnderStrict is the load-bearing half of
// M-0184/AC-2: under `areas.required: true`, the strictness post-pass
// escalates area-unknown to error. A global-tagged entity must not be
// blocked — and it isn't, because AreaUnknown never emitted a finding for
// it (nothing to escalate). Pins that the cross-cutting escape valve
// survives strict mode.
func TestAreaUnknown_GlobalNotBlockedUnderStrict(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "ADR-0001", Kind: entity.KindADR, Path: "docs/adr/ADR-0001-x.md", Area: entity.AreaGlobal},
	)
	findings := AreaUnknown(tr, []string{"platform", "billing"})
	ApplyAreaRequiredStrict(findings, true)
	for _, f := range findings {
		if f.EntityID == "ADR-0001" {
			t.Errorf("global entity must never be blocked under strict mode, got finding: %+v", f)
		}
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings for a global-only tree, got %+v", findings)
	}
}

// TestApplyAreaRequiredStrict_EscalatesAreaUnknown pins M-0178/AC-7:
// when required=true, every `area-unknown` finding is bumped from warning
// to error so the pre-push hook blocks a present-but-undeclared area.
// When required=false, all severities pass through unchanged (byte-for-
// byte today). The bumper is scoped to exactly CodeAreaUnknown — other
// findings pass through untouched. ApplyAreaRequiredStrict mirrors
// ApplyTDDStrict: AreaUnknown stays warning-emitting; the escalation is a
// separate, testable post-pass.
func TestApplyAreaRequiredStrict_EscalatesAreaUnknown(t *testing.T) {
	t.Parallel()
	build := func() []Finding {
		return []Finding{
			{Code: CodeAreaUnknown, Severity: SeverityWarning, EntityID: "G-0001"},
			{Code: CodeAreaUnknown, Severity: SeverityWarning, EntityID: "E-0002"},
			{Code: CodeAreaRequired, Severity: SeverityError, EntityID: "G-0003"},
			{Code: CodeEntityBodyEmpty, Severity: SeverityWarning, EntityID: "M-0001"},
		}
	}

	t.Run("required=true bumps area-unknown to error", func(t *testing.T) {
		findings := build()
		ApplyAreaRequiredStrict(findings, true)
		var saw int
		for _, f := range findings {
			if f.Code == CodeAreaUnknown {
				saw++
				if f.Severity != SeverityError {
					t.Errorf("area-unknown %s severity = %v, want error under required",
						f.EntityID, f.Severity)
				}
			}
			if f.Code == CodeAreaRequired && f.Severity != SeverityError {
				t.Errorf("area-required severity = %v, want error preserved", f.Severity)
			}
			if f.Code == CodeEntityBodyEmpty && f.Severity != SeverityWarning {
				t.Errorf("entity-body-empty severity = %v, want warning unchanged (required only escalates area-unknown)",
					f.Severity)
			}
		}
		if saw != 2 {
			t.Errorf("expected both area-unknown findings present, saw %d", saw)
		}
	})

	t.Run("required=false passes severities through", func(t *testing.T) {
		findings := build()
		ApplyAreaRequiredStrict(findings, false)
		for _, f := range findings {
			if f.Code == CodeAreaUnknown && f.Severity != SeverityWarning {
				t.Errorf("area-unknown %s severity = %v, want warning unchanged when required=false",
					f.EntityID, f.Severity)
			}
		}
	})

	t.Run("nil findings slice is a no-op", func(t *testing.T) {
		ApplyAreaRequiredStrict(nil, true)
	})
}
