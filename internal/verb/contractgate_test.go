package verb

import (
	"testing"

	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// TestContractMutationGate_NoChangeYieldsNoIntroducedFindings: a
// pre-existing finding present in both the current and projected
// configs (the mutation didn't touch that entry) must not be reported
// — it wasn't introduced by this mutation.
func TestContractMutationGate_NoChangeYieldsNoIntroducedFindings(t *testing.T) {
	t.Parallel()
	tr := contractTree("C-0001", "proposed")
	current := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{
			{ID: "C-0001", Validator: "cue", Schema: "missing.cue", Fixtures: "missing-dir"},
		},
	}
	next := cloneContracts(current)

	got := contractMutationGate(tr, current, next, t.TempDir())
	if len(got) != 0 {
		t.Errorf("expected no introduced findings when current == next; got %+v", got)
	}
}

// TestContractMutationGate_IntroducedEntryFindingsSurface: a new
// entry the mutation adds, pointing at schema/fixtures paths that
// don't exist on disk, must surface exactly those two findings.
func TestContractMutationGate_IntroducedEntryFindingsSurface(t *testing.T) {
	t.Parallel()
	tr := contractTree("C-0001", "proposed")
	current := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
	}
	next := cloneContracts(current)
	next.Entries = append(next.Entries, aiwfyaml.Entry{
		ID: "C-0001", Validator: "cue", Schema: "missing.cue", Fixtures: "missing-dir",
	})

	got := contractMutationGate(tr, current, next, t.TempDir())

	var sawSchema, sawFixtures bool
	for _, f := range got {
		if f.Code != "contract-config" || f.EntityID != "C-0001" {
			t.Errorf("unexpected finding: %+v", f)
			continue
		}
		switch f.Subcode {
		case "missing-schema":
			sawSchema = true
		case "missing-fixtures":
			sawFixtures = true
		}
	}
	if !sawSchema || !sawFixtures {
		t.Errorf("expected missing-schema and missing-fixtures findings; got %+v", got)
	}
	if len(got) != 2 {
		t.Errorf("expected exactly 2 introduced findings; got %d: %+v", len(got), got)
	}
}

// TestContractMutationGate_ExcludesPreexistingFindingsOnUntouchedEntries:
// when the mutation adds a new entry while an unrelated entry already
// carries a pre-existing finding, only the new entry's findings are
// reported — proving the exclusion is a true before/after diff, not
// an id-filtered subset (bind's previous design point).
func TestContractMutationGate_ExcludesPreexistingFindingsOnUntouchedEntries(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{ID: "C-0001", Kind: entity.KindContract, Title: "stale", Status: "proposed", Path: "work/contracts/C-001-stale/contract.md"},
			{ID: "C-0002", Kind: entity.KindContract, Title: "new", Status: "proposed", Path: "work/contracts/C-002-new/contract.md"},
		},
	}
	current := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{
			{ID: "C-0001", Validator: "cue", Schema: "gone.cue", Fixtures: "gone"},
		},
	}
	next := cloneContracts(current)
	next.Entries = append(next.Entries, aiwfyaml.Entry{
		ID: "C-0002", Validator: "cue", Schema: "also-gone.cue", Fixtures: "also-gone",
	})

	got := contractMutationGate(tr, current, next, t.TempDir())

	for _, f := range got {
		if f.EntityID != "C-0002" {
			t.Errorf("pre-existing finding on untouched entry leaked into introduced set: %+v", f)
		}
	}
	if len(got) != 2 {
		t.Errorf("expected exactly 2 introduced findings for C-0002; got %d: %+v", len(got), got)
	}
}

// TestContractMutationGate_ResolvedFindingNotReported: a mutation
// that fixes a pre-existing issue (points the schema/fixtures at real
// paths) must not report the now-gone finding — the gate reports only
// additions, never removals.
func TestContractMutationGate_ResolvedFindingNotReported(t *testing.T) {
	t.Parallel()
	tr := contractTree("C-0001", "proposed")
	root := bindRepo(t)
	current := &aiwfyaml.Contracts{
		Validators: map[string]aiwfyaml.Validator{"cue": {Command: "cue"}},
		Entries: []aiwfyaml.Entry{
			{ID: "C-0001", Validator: "cue", Schema: "missing.cue", Fixtures: "missing-dir"},
		},
	}
	next := cloneContracts(current)
	next.Entries[0].Schema = "schema.cue"
	next.Entries[0].Fixtures = "fixtures"

	got := contractMutationGate(tr, current, next, root)
	if len(got) != 0 {
		t.Errorf("expected no introduced findings when the mutation resolves a pre-existing issue; got %+v", got)
	}
}

// TestDiffIntroducedFindings_MultisetCountsSurplusOccurrences: two
// contractcheck findings can never collide byte-for-byte in practice
// (each finding's message embeds its originating entry index or
// entity id — see contractcheck.Run), so real callers can never
// exercise a count above 1. diffIntroducedFindings is still specified
// as a multiset diff, not a set diff — a duplicate present N times in
// before and N times in after is fully excluded, and any surplus
// occurrence in after beyond before's count is reported. This drives
// that contract directly with literal findings, independent of
// contractcheck.Run's real-world uniqueness invariant.
func TestDiffIntroducedFindings_MultisetCountsSurplusOccurrences(t *testing.T) {
	t.Parallel()
	f := check.Finding{Code: "contract-config", Severity: check.SeverityError, Subcode: "missing-schema", EntityID: "C-0001"}

	t.Run("equal counts exclude entirely", func(t *testing.T) {
		t.Parallel()
		got := diffIntroducedFindings([]check.Finding{f, f}, []check.Finding{f, f})
		if len(got) != 0 {
			t.Errorf("expected no introduced findings when before and after share the same count; got %+v", got)
		}
	})

	t.Run("surplus occurrence in after is reported", func(t *testing.T) {
		t.Parallel()
		got := diffIntroducedFindings([]check.Finding{f}, []check.Finding{f, f})
		if diff := len(got); diff != 1 {
			t.Fatalf("expected exactly 1 introduced (surplus) finding; got %d: %+v", diff, got)
		}
		if got[0] != f {
			t.Errorf("introduced finding = %+v, want %+v", got[0], f)
		}
	})
}
