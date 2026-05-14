package htmlrender

import (
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// TestDefaultResolver_KindIndexData_UnknownReturnsNil — branch-
// coverage for the defaultResolver.KindIndexData nil-return path.
// Unrecognized plural slugs return (nil, nil) so the renderer's
// renderKindIndex helper can skip emission cleanly. Without this
// test the default arm of the switch in kindPluralToKind stays
// uncovered.
func TestDefaultResolver_KindIndexData_UnknownReturnsNil(t *testing.T) {
	t.Parallel()
	r := defaultResolver{tree: &tree.Tree{}}
	for _, plural := range []string{"", "gap", "milestones", "tomatoes"} {
		t.Run(plural, func(t *testing.T) {
			t.Parallel()
			data, err := r.KindIndexData(plural, false)
			if err != nil {
				t.Errorf("err = %v, want nil", err)
			}
			if data != nil {
				t.Errorf("data = %+v, want nil for unknown plural", data)
			}
		})
	}
}

// TestKindPluralToKind_UnknownReturnsFalse — direct branch-
// coverage of the kindPluralToKind switch's default arm.
func TestKindPluralToKind_UnknownReturnsFalse(t *testing.T) {
	t.Parallel()
	cases := []string{"", "gap", "milestones", "tomatoes"}
	for _, plural := range cases {
		t.Run(plural, func(t *testing.T) {
			t.Parallel()
			_, ok := kindPluralToKind(plural)
			if ok {
				t.Errorf("kindPluralToKind(%q): unknown plural reported as known", plural)
			}
		})
	}
}

// TestTitleForKindIndex_EmptyAndPreCapitalized exercises the
// "empty string" and "already-capitalized" no-op branches of
// the title helper. Active-default + all-set toggles each.
func TestTitleForKindIndex_EmptyAndPreCapitalized(t *testing.T) {
	t.Parallel()
	cases := []struct {
		plural          string
		includeArchived bool
		want            string
	}{
		{"", false, ""},
		{"", true, "All "},
		{"Gaps", false, "Gaps"},
	}
	for _, c := range cases {
		got := titleForKindIndex(c.plural, c.includeArchived)
		if got != c.want {
			t.Errorf("titleForKindIndex(%q, %v) = %q, want %q", c.plural, c.includeArchived, got, c.want)
		}
	}
}

// TestRenderKindIndex_UnknownKindIsSkipped — branch-coverage for
// the htmlrender package's renderKindIndex helper: when the
// resolver returns nil data, the helper exits without writing a
// file. The full file-emission path is exercised by every
// integration test that renders a known kind; this test pins the
// "no file written" branch via a stub resolver returning nil.
func TestRenderKindIndex_UnknownKindIsSkipped(t *testing.T) {
	t.Parallel()
	tmpls, err := loadTemplates()
	if err != nil {
		t.Fatalf("loadTemplates: %v", err)
	}
	stub := stubKindIndexResolver{}
	opts := Options{OutDir: t.TempDir(), Tree: &tree.Tree{}, Data: stub}
	if err := renderKindIndex(opts, tmpls, stub, "tomatoes", false); err != nil {
		t.Errorf("renderKindIndex on nil-returning resolver: err = %v, want nil", err)
	}
}

// stubKindIndexResolver is a minimal PageDataResolver that
// returns nil for every per-page request — used to exercise the
// renderer's skip-on-nil branches.
type stubKindIndexResolver struct{}

func (stubKindIndexResolver) IndexData() (*IndexData, error)               { return nil, nil }
func (stubKindIndexResolver) EpicData(string) (*EpicData, error)           { return nil, nil }
func (stubKindIndexResolver) MilestoneData(string) (*MilestoneData, error) { return nil, nil }
func (stubKindIndexResolver) EntityData(string) (*EntityData, error)       { return nil, nil }
func (stubKindIndexResolver) StatusData() (*StatusData, error)             { return nil, nil }
func (stubKindIndexResolver) KindIndexData(string, bool) (*KindIndexData, error) {
	return nil, nil
}
