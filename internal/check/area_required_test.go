package check

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// TestAreaRequired_FiresForAllRootKinds pins M-0178/AC-2: with
// required:true and a declared set, AreaRequired emits an error-severity
// finding for every untagged, non-archived root entity across all five
// self-tagging kinds (epic, gap, ADR, decision, contract), naming the
// entity + the declared set. A tagged entity and an archived untagged
// entity raise nothing. A "skip ADR/decision/contract" mutation reddens
// the per-kind count.
func TestAreaRequired_FiresForAllRootKinds(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Path: "work/epics/E-0001-x/epic.md"},
		&entity.Entity{ID: "G-0001", Kind: entity.KindGap, Path: "work/gaps/G-0001-x.md"},
		&entity.Entity{ID: "ADR-0001", Kind: entity.KindADR, Path: "docs/adr/ADR-0001-x.md"},
		&entity.Entity{ID: "D-0001", Kind: entity.KindDecision, Path: "work/decisions/D-0001-x.md"},
		&entity.Entity{ID: "C-0001", Kind: entity.KindContract, Path: "work/contracts/C-0001-x/contract.md"},
		// Tagged → no finding.
		&entity.Entity{ID: "E-0002", Kind: entity.KindEpic, Path: "work/epics/E-0002-y/epic.md", Area: "platform"},
		// Archived untagged → no finding (ADR-0004 §"check shape rules").
		&entity.Entity{ID: "G-0002", Kind: entity.KindGap, Path: "work/gaps/archive/G-0002-z.md"},
	)
	got := AreaRequired(tr, []string{"platform", "billing"}, true)
	if len(got) != 5 {
		t.Fatalf("expected exactly 5 findings, got %d: %+v", len(got), got)
	}
	fired := map[string]bool{}
	for _, f := range got {
		if f.Code != CodeAreaRequired {
			t.Errorf("%s Code = %q, want %q", f.EntityID, f.Code, CodeAreaRequired)
		}
		if f.Severity != SeverityError {
			t.Errorf("%s Severity = %q, want %q", f.EntityID, f.Severity, SeverityError)
		}
		if f.Field != "area" {
			t.Errorf("%s Field = %q, want area", f.EntityID, f.Field)
		}
		// Message must name the entity and the declared set.
		if !strings.Contains(f.Message, f.EntityID) || !strings.Contains(f.Message, "platform") {
			t.Errorf("Message %q must name the entity id and the declared set", f.Message)
		}
		fired[f.EntityID] = true
	}
	for _, want := range []string{"E-0001", "G-0001", "ADR-0001", "D-0001", "C-0001"} {
		if !fired[want] {
			t.Errorf("expected a finding for %s", want)
		}
	}
	for _, notWant := range []string{"E-0002", "G-0002"} {
		if fired[notWant] {
			t.Errorf("did not expect a finding for %s (tagged or archived)", notWant)
		}
	}
}

// TestAreaRequired_GlobalSatisfies pins M-0184/AC-6: a `global`-tagged
// root entity carries a non-empty area, so area-required never fires for
// it even under required:true — the cross-cutting sentinel satisfies the
// present-at-all chokepoint exactly like any declared member. Pure
// regression pin (AreaRequired already short-circuits on a non-empty
// area); guards against a future change that special-cases global.
func TestAreaRequired_GlobalSatisfies(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "ADR-0001", Kind: entity.KindADR, Path: "docs/adr/ADR-0001-x.md", Area: entity.AreaGlobal},
	)
	if got := AreaRequired(tr, []string{"platform"}, true); len(got) != 0 {
		t.Errorf("global-tagged entity must satisfy area-required, got %+v", got)
	}
}

// TestAreaRequired_InertWhenOff pins M-0178/AC-3: with required false the
// rule returns nil (pre-knob parity), and a direct call with no declared
// members covers the defensive empty-declared guard. Making the rule fire
// when required is false reddens the first case.
func TestAreaRequired_InertWhenOff(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Path: "work/epics/E-0001-x/epic.md"},
		&entity.Entity{ID: "G-0001", Kind: entity.KindGap, Path: "work/gaps/G-0001-x.md"},
	)
	// required=false with a declared set → inert.
	if got := AreaRequired(tr, []string{"platform"}, false); got != nil {
		t.Errorf("required=false must be inert, got %+v", got)
	}
	// required=true but no declared members → the defensive empty-declared
	// guard keeps the rule inert (unreachable via config.Load, which rejects
	// required:true+zero-members — driven directly here for coverage).
	if got := AreaRequired(tr, nil, true); got != nil {
		t.Errorf("empty declared set must be inert, got %+v", got)
	}
}

// TestAreaRequired_NoDoubleReport pins M-0178/AC-4: a milestone (area
// derived from the parent epic, blanked at load) never fires; an untagged
// epic carrying two untagged milestones yields exactly one finding (the
// epic). Removing the KindMilestone skip would jump the count to three.
func TestAreaRequired_NoDoubleReport(t *testing.T) {
	t.Parallel()
	tr := makeTree(
		&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Path: "work/epics/E-0001-x/epic.md"},
		&entity.Entity{ID: "M-0001", Kind: entity.KindMilestone, Path: "work/epics/E-0001-x/M-0001-a.md"},
		&entity.Entity{ID: "M-0002", Kind: entity.KindMilestone, Path: "work/epics/E-0001-x/M-0002-b.md"},
	)
	got := AreaRequired(tr, []string{"platform"}, true)
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 finding (the epic), got %d: %+v", len(got), got)
	}
	if got[0].EntityID != "E-0001" {
		t.Errorf("finding fired for %q, want the epic E-0001", got[0].EntityID)
	}
	if CodeAreaRequired == CodeAreaUnknown {
		t.Errorf("CodeAreaRequired (%q) must differ from CodeAreaUnknown (%q)", CodeAreaRequired, CodeAreaUnknown)
	}
}
