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
