package stresstest

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// verb_sequence_list_invariant_test.go — M-0250/AC-3: pins
// classifyListInvariant and diffListRow against fabricated inputs, so
// every divergence shape (missing row, extra row, mismatched field)
// is exercised deterministically.

func TestClassifyListInvariant_MatchingSetsProduceNoViolations(t *testing.T) {
	t.Parallel()
	want := []*entity.Entity{
		{ID: "E-0001", Kind: entity.KindEpic, Status: "proposed", Title: "epic a", Path: "work/epics/E-0001-epic-a/epic.md"},
	}
	got := []listRow{
		{ID: "E-0001", Kind: "epic", Status: "proposed", Title: "epic a", Path: "work/epics/E-0001-epic-a/epic.md"},
	}
	violations := classifyListInvariant("label", got, want)
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %+v", violations)
	}
}

func TestClassifyListInvariant_GroundTruthEntityMissingFromListIsAViolation(t *testing.T) {
	t.Parallel()
	want := []*entity.Entity{
		{ID: "E-0001", Kind: entity.KindEpic, Status: "proposed", Title: "epic a", Path: "work/epics/E-0001-epic-a/epic.md"},
	}
	var got []listRow
	violations := classifyListInvariant("label", got, want)
	if len(violations) != 1 {
		t.Fatalf("violations = %+v, want exactly 1", violations)
	}
}

func TestClassifyListInvariant_ExtraListRowNotInGroundTruthIsAViolation(t *testing.T) {
	t.Parallel()
	var want []*entity.Entity
	got := []listRow{
		{ID: "E-0001", Kind: "epic", Status: "proposed", Title: "epic a", Path: "work/epics/E-0001-epic-a/epic.md"},
	}
	violations := classifyListInvariant("label", got, want)
	if len(violations) != 1 {
		t.Fatalf("violations = %+v, want exactly 1", violations)
	}
}

func TestClassifyListInvariant_MismatchedFieldIsAViolation(t *testing.T) {
	t.Parallel()
	want := []*entity.Entity{
		{ID: "E-0001", Kind: entity.KindEpic, Status: "proposed", Title: "epic a", Path: "work/epics/E-0001-epic-a/epic.md"},
	}
	got := []listRow{
		{ID: "E-0001", Kind: "epic", Status: "active", Title: "epic a", Path: "work/epics/E-0001-epic-a/epic.md"}, // stale status
	}
	violations := classifyListInvariant("label", got, want)
	if len(violations) != 1 {
		t.Fatalf("violations = %+v, want exactly 1", violations)
	}
}

func TestClassifyListInvariant_IdsAreCanonicalizedBeforeComparison(t *testing.T) {
	t.Parallel()
	// Ground truth may carry a narrower legacy-width id on disk;
	// list always emits canonical width. The comparison must not
	// treat these as two different entities.
	want := []*entity.Entity{
		{ID: "E-01", Kind: entity.KindEpic, Status: "proposed", Title: "epic a", Path: "work/epics/E-0001-epic-a/epic.md"},
	}
	got := []listRow{
		{ID: "E-0001", Kind: "epic", Status: "proposed", Title: "epic a", Path: "work/epics/E-0001-epic-a/epic.md"},
	}
	violations := classifyListInvariant("label", got, want)
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %+v", violations)
	}
}

func TestClassifyListInvariant_ParentIsCanonicalizedBeforeComparison(t *testing.T) {
	t.Parallel()
	want := []*entity.Entity{
		{ID: "M-0001", Kind: entity.KindMilestone, Status: "draft", Title: "m", Parent: "E-01", Path: "work/epics/E-0001-epic-a/M-0001-m.md"},
	}
	got := []listRow{
		{ID: "M-0001", Kind: "milestone", Status: "draft", Title: "m", Parent: "E-0001", Path: "work/epics/E-0001-epic-a/M-0001-m.md"},
	}
	violations := classifyListInvariant("label", got, want)
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %+v", violations)
	}
}

func TestParseListVerbEnvelope_ErrorsOnMalformedJSON(t *testing.T) {
	t.Parallel()
	if _, err := parseListVerbEnvelope([]string{"list"}, []byte("not valid json")); err == nil {
		t.Fatal("expected an error parsing malformed JSON output")
	}
}

func TestParseListVerbEnvelope_DecodesAWellFormedEnvelope(t *testing.T) {
	t.Parallel()
	env, err := parseListVerbEnvelope([]string{"list"}, []byte(`{"status":"ok","result":[{"id":"E-0001","kind":"epic"}]}`))
	if err != nil {
		t.Fatalf("parseListVerbEnvelope: %v", err)
	}
	if env.Status != "ok" || len(env.Result) != 1 || env.Result[0].ID != "E-0001" {
		t.Fatalf("unexpected decoded envelope: %+v", env)
	}
}

func TestCheckListInvariant_RealBinary_ErrorsWhenBinaryMissing(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	dir := newVerbSequenceTestRepo(t)

	_, err := checkListInvariant(filepath.Join(t.TempDir(), "no-such-aiwf-binary"), dir, "label")
	if err == nil {
		t.Fatal("expected checkListInvariant to error when the aiwf binary path doesn't exist")
	}
	if !strings.Contains(err.Error(), "running aiwf list --archived") {
		t.Fatalf("expected the launch failure to surface via runAiwfListJSON's wrapping, got: %v", err)
	}
}

func TestDiffListRow_MatchingRowsProduceEmptyDiff(t *testing.T) {
	t.Parallel()
	row := listRow{ID: "E-0001", Kind: "epic", Status: "proposed", Title: "t", Parent: "", Path: "p"}
	if diff := diffListRow(row, row); diff != "" {
		t.Errorf("diff = %q, want empty", diff)
	}
}

func TestDiffListRow_NamesEveryMismatchedField(t *testing.T) {
	t.Parallel()
	want := listRow{ID: "E-0001", Kind: "epic", Status: "proposed", Title: "t", Parent: "", Path: "p1"}
	got := listRow{ID: "E-0001", Kind: "milestone", Status: "active", Title: "t2", Parent: "E-0002", Path: "p2"}
	diff := diffListRow(want, got)
	for _, field := range []string{"kind", "status", "title", "parent", "path"} {
		if !strings.Contains(diff, field) {
			t.Errorf("diff %q does not mention %q", diff, field)
		}
	}
}
