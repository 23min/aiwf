package check

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// TestEntityIDNarrowWidth tests the drift-check rule: a narrow-width
// id anywhere in the active tree (entities outside `<kind>/archive/`)
// is a defect, and fires one error-severity finding per narrow file.
// Whether canonical ids sit alongside it makes no difference.
//
// Archive entries never fire: per ADR-0008's "Drift control" subsection
// they are excluded outright, and per ADR-0004's forget-by-default
// principle a narrow archived file stays narrow permanently. The
// archive fixtures below carry `G-001` rather than a one-digit id
// because entity.IDFromPath rejects anything below the gap grammar's
// three-digit floor — a one-digit fixture is dropped before the
// archive test runs, so it would assert nothing about the exclusion.
//
// ADR never fires either, structurally rather than by exemption: the
// ADR grammar floor is canonical width, so IDFromPath admits no narrow
// ADR for the width test to see.
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
			name: "uniform narrow active tree fires on every narrow entry",
			tr: makeTree(
				&entity.Entity{ID: "E-22", Kind: entity.KindEpic, Path: "work/epics/E-22-foo/epic.md"},
				&entity.Entity{ID: "M-100", Kind: entity.KindMilestone, Path: "work/epics/E-22-foo/M-100-bar.md"},
			),
			wantCount:   2,
			wantNarrows: []string{"E-22", "M-100"},
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
			name: "a lone narrow entity fires",
			tr: makeTree(
				&entity.Entity{ID: "G-100", Kind: entity.KindGap, Path: "work/gaps/G-100-foo.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"G-100"},
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
			name: "canonical alongside narrow fires on the narrow entry only",
			tr: makeTree(
				&entity.Entity{ID: "E-0023", Kind: entity.KindEpic, Path: "work/epics/E-0023-new/epic.md"},
				&entity.Entity{ID: "G-100", Kind: entity.KindGap, Path: "work/gaps/G-100-old.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"G-100"},
		},
		{
			name: "multiple narrow entries fire once each",
			tr: makeTree(
				&entity.Entity{ID: "E-0023", Kind: entity.KindEpic, Path: "work/epics/E-0023-new/epic.md"},
				&entity.Entity{ID: "G-100", Kind: entity.KindGap, Path: "work/gaps/G-100-old.md"},
				&entity.Entity{ID: "D-001", Kind: entity.KindDecision, Path: "work/decisions/D-001-old.md"},
			),
			wantCount:   2,
			wantNarrows: []string{"G-100", "D-001"},
		},
		{
			name: "narrow archive entries never fire alongside a canonical active tree",
			tr: makeTree(
				&entity.Entity{ID: "E-0023", Kind: entity.KindEpic, Path: "work/epics/E-0023-new/epic.md"},
				&entity.Entity{ID: "G-001", Kind: entity.KindGap, Path: "work/gaps/archive/G-001-old.md"},
			),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			name: "narrow archive entries never fire; the narrow active entry alongside them still does",
			tr: makeTree(
				&entity.Entity{ID: "E-22", Kind: entity.KindEpic, Path: "work/epics/E-22-foo/epic.md"},
				&entity.Entity{ID: "G-001", Kind: entity.KindGap, Path: "work/gaps/archive/G-001-old.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"E-22"},
		},
		{
			name: "narrow archive alongside a mixed active tree: fires only on the active narrow",
			tr: makeTree(
				&entity.Entity{ID: "E-0023", Kind: entity.KindEpic, Path: "work/epics/E-0023-new/epic.md"},
				&entity.Entity{ID: "G-100", Kind: entity.KindGap, Path: "work/gaps/G-100-old.md"},
				&entity.Entity{ID: "G-001", Kind: entity.KindGap, Path: "work/gaps/archive/G-001-archived.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"G-100"},
		},
		{
			name: "a canonical ADR never fires, and the narrow entry beside it still does",
			tr: makeTree(
				&entity.Entity{ID: "ADR-0001", Kind: entity.KindADR, Path: "docs/adr/ADR-0001-foo.md"},
				&entity.Entity{ID: "E-22", Kind: entity.KindEpic, Path: "work/epics/E-22-foo/epic.md"},
			),
			wantCount:   1,
			wantNarrows: []string{"E-22"},
		},
		{
			name: "ADR alone is silent",
			tr: makeTree(
				&entity.Entity{ID: "ADR-0001", Kind: entity.KindADR, Path: "docs/adr/ADR-0001-foo.md"},
			),
			wantCount:   0,
			wantNarrows: nil,
		},
		{
			name: "ADR alongside a canonical and a narrow entry fires on the narrow only",
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
				if f.Code != CodeEntityIDNarrowWidth {
					t.Errorf("Code = %q, want entity-id-narrow-width", f.Code)
				}
				if f.Severity != SeverityError {
					t.Errorf("Severity = %q, want error", f.Severity)
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

// TestEntityIDNarrowWidth_RemediationNamesNoVerb pins what the finding
// tells an operator to do. No verb widens an id in place, so the
// message must point at undoing the hand-edit or file move that
// produced the narrow id. Naming a verb here would send the reader
// somewhere that cannot help: `aiwf reallocate` assigns a *different*
// number rather than the same one at canonical width.
func TestEntityIDNarrowWidth_RemediationNamesNoVerb(t *testing.T) {
	t.Parallel()
	got := entityIDNarrowWidth(makeTree(
		&entity.Entity{ID: "G-100", Kind: entity.KindGap, Path: "work/gaps/G-100-old.md"},
	))
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1: %+v", len(got), got)
	}
	msg := got[0].Message
	for _, forbidden := range []string{"rewidth", "reallocate"} {
		if strings.Contains(msg, forbidden) {
			t.Errorf("remediation names %q, which cannot widen an id in place: %s", forbidden, msg)
		}
	}
	for _, want := range []string{"hand-edit", "move"} {
		if !strings.Contains(msg, want) {
			t.Errorf("remediation does not mention %q — an operator cannot act on it: %s", want, msg)
		}
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
