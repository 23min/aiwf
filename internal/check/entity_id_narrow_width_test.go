package check

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// TestEntityIDNarrowWidth tests the M-083 AC-1 drift-check rule.
// The rule classifies the active tree (excluding `<kind>/archive/`)
// as uniform-narrow, uniform-canonical, or mixed. Only the mixed
// state fires findings — one warning per narrow-width active file.
//
// Per ADR-0008's "Drift control" subsection, archive entries are
// excluded from the mixed-state computation entirely; pre-existing
// narrow files in archive stay narrow forever per ADR-0004's
// forget-by-default principle.
func TestEntityIDNarrowWidth(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		tr          *tree.Tree
		wantCount   int
		wantNarrows []string // entity ids the rule should fire on, in any order
	}{
		{
			name:        "empty active tree is silent",
			tr:          makeTree(),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			name: "uniform narrow active tree is silent",
			tr: makeTree(
				&entity.Entity{ID: "E-22", Kind: entity.KindEpic, Path: "work/epics/E-22-foo/epic.md"},
				&entity.Entity{ID: "M-100", Kind: entity.KindMilestone, Path: "work/epics/E-22-foo/M-100-bar.md"},
			),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			name: "uniform canonical active tree is silent",
			tr: makeTree(
				&entity.Entity{ID: "E-0023", Kind: entity.KindEpic, Path: "work/epics/E-0023-foo/epic.md"},
				&entity.Entity{ID: "M-0083", Kind: entity.KindMilestone, Path: "work/epics/E-0023-foo/M-0083-bar.md"},
			),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			name: "single narrow entity is silent",
			tr: makeTree(
				&entity.Entity{ID: "G-100", Kind: entity.KindGap, Path: "work/gaps/G-100-foo.md"},
			),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			name: "single canonical entity is silent",
			tr: makeTree(
				&entity.Entity{ID: "G-0100", Kind: entity.KindGap, Path: "work/gaps/G-0100-foo.md"},
			),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			name: "mixed active tree fires on narrow entries only",
			tr: makeTree(
				&entity.Entity{ID: "E-0023", Kind: entity.KindEpic, Path: "work/epics/E-0023-new/epic.md"},
				&entity.Entity{ID: "G-100", Kind: entity.KindGap, Path: "work/gaps/G-100-old.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"G-100"},
		},
		{
			name: "mixed active tree with multiple narrow entries fires once each",
			tr: makeTree(
				&entity.Entity{ID: "E-0023", Kind: entity.KindEpic, Path: "work/epics/E-0023-new/epic.md"},
				&entity.Entity{ID: "G-100", Kind: entity.KindGap, Path: "work/gaps/G-100-old.md"},
				&entity.Entity{ID: "D-001", Kind: entity.KindDecision, Path: "work/decisions/D-001-old.md"},
			),
			wantCount:   2,
			wantNarrows: []string{"G-100", "D-001"},
		},
		{
			name: "narrow archive entries do not trigger mixed-state when active tree is uniform-canonical",
			tr: makeTree(
				&entity.Entity{ID: "E-0023", Kind: entity.KindEpic, Path: "work/epics/E-0023-new/epic.md"},
				&entity.Entity{ID: "G-1", Kind: entity.KindGap, Path: "work/gaps/archive/G-1-old.md"},
			),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			name: "narrow archive entries do not trigger mixed-state when active tree is uniform-narrow",
			tr: makeTree(
				&entity.Entity{ID: "E-22", Kind: entity.KindEpic, Path: "work/epics/E-22-foo/epic.md"},
				&entity.Entity{ID: "G-1", Kind: entity.KindGap, Path: "work/gaps/archive/G-1-old.md"},
			),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			name: "mixed active tree alongside narrow archive: warning fires only on active narrow",
			tr: makeTree(
				&entity.Entity{ID: "E-0023", Kind: entity.KindEpic, Path: "work/epics/E-0023-new/epic.md"},
				&entity.Entity{ID: "G-100", Kind: entity.KindGap, Path: "work/gaps/G-100-old.md"},
				&entity.Entity{ID: "G-1", Kind: entity.KindGap, Path: "work/gaps/archive/G-1-archived.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"G-100"},
		},
		{
			name: "ADR is exempt from mixed-state computation: ADR-0001 + uniform-narrow E/M/G/D/C is silent",
			tr: makeTree(
				&entity.Entity{ID: "ADR-0001", Kind: entity.KindADR, Path: "docs/adr/ADR-0001-foo.md"},
				&entity.Entity{ID: "E-22", Kind: entity.KindEpic, Path: "work/epics/E-22-foo/epic.md"},
			),
			// ADR has no narrow-legacy form; the active set considered
			// for classification is just E-22 (uniform-narrow) → silent.
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			name: "ADR alone is silent (not enough non-ADR entities to classify)",
			tr: makeTree(
				&entity.Entity{ID: "ADR-0001", Kind: entity.KindADR, Path: "docs/adr/ADR-0001-foo.md"},
			),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			name: "ADR alongside mixed E/M tree fires on the narrow E/M only",
			tr: makeTree(
				&entity.Entity{ID: "ADR-0001", Kind: entity.KindADR, Path: "docs/adr/ADR-0001-foo.md"},
				&entity.Entity{ID: "E-0023", Kind: entity.KindEpic, Path: "work/epics/E-0023-new/epic.md"},
				&entity.Entity{ID: "M-100", Kind: entity.KindMilestone, Path: "work/epics/E-0023-new/M-100-bar.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"M-100"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := entityIDNarrowWidth(tc.tr)
			if len(got) != tc.wantCount {
				t.Fatalf("entityIDNarrowWidth findings = %d, want %d: %+v",
					len(got), tc.wantCount, got)
			}
			if tc.wantCount == 0 {
				return
			}
			seen := make(map[string]bool, len(got))
			for _, f := range got {
				seen[f.EntityID] = true
				if f.Code != "entity-id-narrow-width" {
					t.Errorf("Code = %q, want entity-id-narrow-width", f.Code)
				}
				if f.Severity != SeverityWarning {
					t.Errorf("Severity = %q, want warning", f.Severity)
				}
				if f.Path == "" {
					t.Errorf("Path must be set on finding for %s", f.EntityID)
				}
			}
			for _, want := range tc.wantNarrows {
				if !seen[want] {
					t.Errorf("expected finding for entity %q, got %+v", want, got)
				}
			}
		})
	}
}

// TestIsNarrowID_DefensiveBranches exercises the defensive
// fall-through branches in isNarrowID directly. The parent rule
// (entityIDNarrowWidth) only feeds path-validated ids in production,
// so these inputs are unreachable through the rule's table-driven
// fixtures. Kept as a separate unit test so a future change to the
// helper's contract still catches the malformed-input cases.
func TestIsNarrowID_DefensiveBranches(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		id   string
		want bool
	}{
		{"prefix only, no digits", "E-", false},
		{"non-digit in tail", "E-12a", false},
		{"unknown prefix", "X-12", false},
		{"empty string", "", false},
		{"narrow E", "E-1", true},
		{"narrow M", "M-99", true},
		{"canonical E", "E-0001", false},
		{"natural-width above pad", "M-12345", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isNarrowID(tc.id); got != tc.want {
				t.Errorf("isNarrowID(%q) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}
