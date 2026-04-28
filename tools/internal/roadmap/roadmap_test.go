package roadmap

import (
	"bytes"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

func TestRender_EmptyTree(t *testing.T) {
	got := string(Render(&tree.Tree{}))
	want := "# Roadmap\n\n_No epics yet._\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRender_EpicWithoutMilestones(t *testing.T) {
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-01", Title: "Foundations", Status: "active"},
		},
	}
	got := string(Render(tr))
	for _, want := range []string{
		"# Roadmap",
		"## E-01 — Foundations (active)",
		"_No milestones yet._",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestRender_GroupsAndSortsMilestones(t *testing.T) {
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			// Out-of-order on purpose to confirm we sort.
			{Kind: entity.KindEpic, ID: "E-02", Title: "Reporting", Status: "proposed"},
			{Kind: entity.KindEpic, ID: "E-01", Title: "Auth", Status: "active"},
			{Kind: entity.KindMilestone, ID: "M-002", Title: "Login", Status: "in_progress", Parent: "E-01"},
			{Kind: entity.KindMilestone, ID: "M-001", Title: "Schema", Status: "done", Parent: "E-01"},
			{Kind: entity.KindMilestone, ID: "M-010", Title: "Dashboards", Status: "draft", Parent: "E-02"},
		},
	}
	got := string(Render(tr))

	idxE01 := strings.Index(got, "## E-01")
	idxE02 := strings.Index(got, "## E-02")
	if idxE01 < 0 || idxE02 < 0 || idxE01 > idxE02 {
		t.Fatalf("epics not in id order:\n%s", got)
	}
	idxM001 := strings.Index(got, "M-001")
	idxM002 := strings.Index(got, "M-002")
	if idxM001 < 0 || idxM002 < 0 || idxM001 > idxM002 {
		t.Errorf("milestones within an epic not in id order:\n%s", got)
	}
	if !strings.Contains(got, "| M-010 | Dashboards | draft |") {
		t.Errorf("E-02's milestone row missing:\n%s", got)
	}
}

func TestRender_OrphanedMilestonesSurfaced(t *testing.T) {
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-01", Title: "Auth", Status: "active"},
			{Kind: entity.KindMilestone, ID: "M-001", Title: "Schema", Status: "done", Parent: "E-01"},
			{Kind: entity.KindMilestone, ID: "M-099", Title: "Stray", Status: "draft", Parent: "E-99"},
		},
	}
	got := string(Render(tr))
	if !strings.Contains(got, "## Unparented milestones") {
		t.Errorf("orphan section missing:\n%s", got)
	}
	if !strings.Contains(got, "| M-099 | Stray | E-99 | draft |") {
		t.Errorf("orphan row missing:\n%s", got)
	}
}

func TestRender_EscapesPipesAndNewlinesInTitles(t *testing.T) {
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-01", Title: "Pipes | inside | title", Status: "active"},
			{Kind: entity.KindMilestone, ID: "M-001", Title: "two\nlines", Status: "draft", Parent: "E-01"},
		},
	}
	got := string(Render(tr))
	if !strings.Contains(got, `Pipes \| inside \| title`) {
		t.Errorf("epic title pipes not escaped:\n%s", got)
	}
	if strings.Contains(got, "two\nlines") {
		t.Errorf("milestone title newline not collapsed:\n%s", got)
	}
	if !strings.Contains(got, "| M-001 | two lines | draft |") {
		t.Errorf("milestone row missing or malformed:\n%s", got)
	}
}

func TestRender_Deterministic(t *testing.T) {
	build := func() *tree.Tree {
		return &tree.Tree{
			Entities: []*entity.Entity{
				{Kind: entity.KindEpic, ID: "E-01", Title: "A", Status: "active"},
				{Kind: entity.KindEpic, ID: "E-02", Title: "B", Status: "proposed"},
				{Kind: entity.KindMilestone, ID: "M-001", Title: "X", Status: "draft", Parent: "E-01"},
				{Kind: entity.KindMilestone, ID: "M-002", Title: "Y", Status: "draft", Parent: "E-02"},
			},
		}
	}
	a := Render(build())
	b := Render(build())
	if !bytes.Equal(a, b) {
		t.Errorf("output differs across runs:\n%s\nvs\n%s", a, b)
	}
}

func TestRender_IgnoresNonEpicNonMilestoneKinds(t *testing.T) {
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-01", Title: "Foo", Status: "active"},
			{Kind: entity.KindADR, ID: "ADR-0001", Title: "Use Postgres", Status: "accepted"},
			{Kind: entity.KindGap, ID: "G-001", Title: "Auth gap", Status: "open"},
			{Kind: entity.KindDecision, ID: "D-001", Title: "Sunset v1", Status: "accepted"},
			{Kind: entity.KindContract, ID: "C-001", Title: "Public API", Status: "draft"},
		},
	}
	got := string(Render(tr))
	for _, mustNotContain := range []string{"ADR-0001", "G-001", "D-001", "C-001"} {
		if strings.Contains(got, mustNotContain) {
			t.Errorf("output should not mention %q (only epics + milestones):\n%s", mustNotContain, got)
		}
	}
}

// TestExtractCandidates_FromCanonicalSection: the documented `##
// Candidates` heading is recognized and the section body is returned
// verbatim, stopping at the next `## ` heading.
func TestExtractCandidates_FromCanonicalSection(t *testing.T) {
	src := []byte(`# Roadmap

## E-01 — Foo (active)

| Milestone | Title | Status |
|---|---|---|
| M-001 | x | draft |

## Candidates

- Anomaly detection — interesting if telemetry lands.
- Subsystem boundaries — needs a champion.

## Notes

other stuff
`)
	got := ExtractCandidates(src)
	if got == nil {
		t.Fatal("ExtractCandidates returned nil")
	}
	want := "## Candidates\n\n- Anomaly detection — interesting if telemetry lands.\n- Subsystem boundaries — needs a champion.\n"
	if string(got) != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

// TestExtractCandidates_BacklogAlias: "Backlog" is accepted as a
// drop-in heading for repos that prefer that wording.
func TestExtractCandidates_BacklogAlias(t *testing.T) {
	src := []byte("## Backlog\n\n- One\n- Two\n")
	got := ExtractCandidates(src)
	if got == nil {
		t.Fatal("Backlog heading not recognized")
	}
	if !bytes.Contains(got, []byte("- One")) {
		t.Errorf("body lost: %s", got)
	}
}

// TestExtractCandidates_None: no recognized heading returns nil so
// the caller can skip the append.
func TestExtractCandidates_None(t *testing.T) {
	src := []byte("# Roadmap\n\n## E-01 — Foo (active)\n\nstuff\n")
	if got := ExtractCandidates(src); got != nil {
		t.Errorf("expected nil, got %s", got)
	}
}

// TestExtractCandidates_RunsToEOF: when the candidates section is
// the last block in the file it captures through EOF.
func TestExtractCandidates_RunsToEOF(t *testing.T) {
	src := []byte("## Candidates\n\n- One\n- Two\n")
	got := ExtractCandidates(src)
	if !bytes.Contains(got, []byte("- Two")) {
		t.Errorf("EOF section truncated: %s", got)
	}
}

// TestAppendCandidates_RoundTrip ensures a regenerated roadmap with
// candidates appended matches the input shape closely enough that a
// second pass leaves it unchanged.
func TestAppendCandidates_RoundTrip(t *testing.T) {
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-01", Title: "Foo", Status: "active"},
		},
	}
	gen := Render(tr)
	cands := []byte("## Candidates\n\n- Idea A\n- Idea B\n")
	merged := AppendCandidates(gen, cands)
	if !bytes.Contains(merged, []byte("## E-01")) {
		t.Errorf("epic section lost in merge:\n%s", merged)
	}
	if !bytes.Contains(merged, []byte("- Idea A")) {
		t.Errorf("candidates lost in merge:\n%s", merged)
	}
	// Re-extracting the candidates from the merged output yields back
	// the same section.
	roundTrip := ExtractCandidates(merged)
	if !bytes.Equal(bytes.TrimSpace(roundTrip), bytes.TrimSpace(cands)) {
		t.Errorf("round trip changed candidates:\nbefore: %q\nafter:  %q", cands, roundTrip)
	}
}

// TestAppendCandidates_NilCandidates: a nil candidates input returns
// the generated output verbatim.
func TestAppendCandidates_NilCandidates(t *testing.T) {
	gen := []byte("# Roadmap\n\n_x_\n")
	got := AppendCandidates(gen, nil)
	if !bytes.Equal(got, gen) {
		t.Errorf("AppendCandidates with nil tail mutated input:\n%q", got)
	}
}
