package roadmap

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// writeEpicFile writes an epic.md fixture under root with the given
// frontmatter+body content and returns the repo-relative path the
// roadmap renderer will look up.
func writeEpicFile(t *testing.T, root, slug, content string) string {
	t.Helper()
	rel := filepath.Join("work", "epics", slug, "epic.md")
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return rel
}

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
			{Kind: entity.KindEpic, ID: "E-0001", Title: "Foundations", Status: "active"},
		},
	}
	got := string(Render(tr))
	for _, want := range []string{
		"# Roadmap",
		"## E-0001 — Foundations (active)",
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
			{Kind: entity.KindEpic, ID: "E-0002", Title: "Reporting", Status: "proposed"},
			{Kind: entity.KindEpic, ID: "E-0001", Title: "Auth", Status: "active"},
			{Kind: entity.KindMilestone, ID: "M-0002", Title: "Login", Status: "in_progress", Parent: "E-0001"},
			{Kind: entity.KindMilestone, ID: "M-0001", Title: "Schema", Status: "done", Parent: "E-0001"},
			{Kind: entity.KindMilestone, ID: "M-0010", Title: "Dashboards", Status: "draft", Parent: "E-0002"},
		},
	}
	got := string(Render(tr))

	idxE01 := strings.Index(got, "## E-0001")
	idxE02 := strings.Index(got, "## E-0002")
	if idxE01 < 0 || idxE02 < 0 || idxE01 > idxE02 {
		t.Fatalf("epics not in id order:\n%s", got)
	}
	idxM001 := strings.Index(got, "M-0001")
	idxM002 := strings.Index(got, "M-0002")
	if idxM001 < 0 || idxM002 < 0 || idxM001 > idxM002 {
		t.Errorf("milestones within an epic not in id order:\n%s", got)
	}
	if !strings.Contains(got, "| M-0010 | Dashboards | draft |") {
		t.Errorf("E-0002's milestone row missing:\n%s", got)
	}
}

func TestRender_OrphanedMilestonesSurfaced(t *testing.T) {
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-0001", Title: "Auth", Status: "active"},
			{Kind: entity.KindMilestone, ID: "M-0001", Title: "Schema", Status: "done", Parent: "E-0001"},
			{Kind: entity.KindMilestone, ID: "M-0099", Title: "Stray", Status: "draft", Parent: "E-0099"},
		},
	}
	got := string(Render(tr))
	if !strings.Contains(got, "## Unparented milestones") {
		t.Errorf("orphan section missing:\n%s", got)
	}
	if !strings.Contains(got, "| M-0099 | Stray | E-0099 | draft |") {
		t.Errorf("orphan row missing:\n%s", got)
	}
}

func TestRender_EscapesPipesAndNewlinesInTitles(t *testing.T) {
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-0001", Title: "Pipes | inside | title", Status: "active"},
			{Kind: entity.KindMilestone, ID: "M-0001", Title: "two\nlines", Status: "draft", Parent: "E-0001"},
		},
	}
	got := string(Render(tr))
	if !strings.Contains(got, `Pipes \| inside \| title`) {
		t.Errorf("epic title pipes not escaped:\n%s", got)
	}
	if strings.Contains(got, "two\nlines") {
		t.Errorf("milestone title newline not collapsed:\n%s", got)
	}
	if !strings.Contains(got, "| M-0001 | two lines | draft |") {
		t.Errorf("milestone row missing or malformed:\n%s", got)
	}
}

func TestRender_Deterministic(t *testing.T) {
	build := func() *tree.Tree {
		return &tree.Tree{
			Entities: []*entity.Entity{
				{Kind: entity.KindEpic, ID: "E-0001", Title: "A", Status: "active"},
				{Kind: entity.KindEpic, ID: "E-0002", Title: "B", Status: "proposed"},
				{Kind: entity.KindMilestone, ID: "M-0001", Title: "X", Status: "draft", Parent: "E-0001"},
				{Kind: entity.KindMilestone, ID: "M-0002", Title: "Y", Status: "draft", Parent: "E-0002"},
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
			{Kind: entity.KindEpic, ID: "E-0001", Title: "Foo", Status: "active"},
			{Kind: entity.KindADR, ID: "ADR-0001", Title: "Use Postgres", Status: "accepted"},
			{Kind: entity.KindGap, ID: "G-0001", Title: "Auth gap", Status: "open"},
			{Kind: entity.KindDecision, ID: "D-0001", Title: "Sunset v1", Status: "accepted"},
			{Kind: entity.KindContract, ID: "C-0001", Title: "Public API", Status: "draft"},
		},
	}
	got := string(Render(tr))
	for _, mustNotContain := range []string{"ADR-0001", "G-0001", "D-0001", "C-0001"} {
		if strings.Contains(got, mustNotContain) {
			t.Errorf("output should not mention %q (only epics + milestones):\n%s", mustNotContain, got)
		}
	}
}

// TestRender_IncludesEpicGoal: when an epic's body has a populated
// `## Goal` section, the roadmap surfaces it as `### Goal` between the
// epic heading and the milestone table.
func TestRender_IncludesEpicGoal(t *testing.T) {
	root := t.TempDir()
	path := writeEpicFile(t, root, "E-0001-foo", `---
id: E-01
title: Foo
status: active
---

## Goal

Land the foundation pieces:

- piece A
- piece B

## Scope

unrelated content
`)

	tr := &tree.Tree{
		Root: root,
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-0001", Title: "Foo", Status: "active", Path: path},
		},
	}
	got := string(Render(tr))

	if !strings.Contains(got, "### Goal\n\nLand the foundation pieces:\n\n- piece A\n- piece B") {
		t.Errorf("goal body missing or malformed:\n%s", got)
	}
	if strings.Contains(got, "unrelated content") {
		t.Errorf("scope section leaked into goal:\n%s", got)
	}
	// Goal must appear before the (empty) milestones notice.
	if idxGoal, idxMs := strings.Index(got, "### Goal"), strings.Index(got, "_No milestones yet._"); idxGoal < 0 || idxMs < 0 || idxGoal > idxMs {
		t.Errorf("goal not positioned before milestones:\n%s", got)
	}
}

// TestRender_SkipsEmptyGoal: an epic whose `## Goal` section is
// whitespace-only (the BodyTemplate default) does not introduce an
// empty `### Goal` block.
func TestRender_SkipsEmptyGoal(t *testing.T) {
	root := t.TempDir()
	path := writeEpicFile(t, root, "E-0001-foo", `---
id: E-01
title: Foo
status: active
---

## Goal

## Scope
`)

	tr := &tree.Tree{
		Root: root,
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-0001", Title: "Foo", Status: "active", Path: path},
		},
	}
	got := string(Render(tr))
	if strings.Contains(got, "### Goal") {
		t.Errorf("empty Goal section should not be emitted:\n%s", got)
	}
}

// TestRender_NoBodyFile_NoGoal: a tree with no on-disk file (Path
// unset, or root unset) skips the goal lookup silently. This keeps
// purely-in-memory tests working.
func TestRender_NoBodyFile_NoGoal(t *testing.T) {
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-0001", Title: "Foo", Status: "active"},
		},
	}
	got := string(Render(tr))
	if strings.Contains(got, "### Goal") {
		t.Errorf("Goal emitted without a backing file:\n%s", got)
	}
}

func TestExtractSection_StopsAtNextH2(t *testing.T) {
	src := []byte("## Goal\n\nfirst\n\n## Scope\n\nsecond\n")
	got := extractSection(src, "Goal")
	if string(got) != "first" {
		t.Errorf("got %q, want %q", got, "first")
	}
}

func TestExtractSection_RunsToEOF(t *testing.T) {
	src := []byte("## Goal\n\nlone-section\n")
	got := extractSection(src, "Goal")
	if string(got) != "lone-section" {
		t.Errorf("got %q", got)
	}
}

func TestExtractSection_MissingHeading(t *testing.T) {
	src := []byte("## Scope\n\nno goal here\n")
	if got := extractSection(src, "Goal"); got != nil {
		t.Errorf("got %q, want nil", got)
	}
}

func TestExtractSection_WhitespaceOnly(t *testing.T) {
	src := []byte("## Goal\n\n   \n\n## Scope\n")
	if got := extractSection(src, "Goal"); got != nil {
		t.Errorf("got %q, want nil", got)
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
			{Kind: entity.KindEpic, ID: "E-0001", Title: "Foo", Status: "active"},
		},
	}
	gen := Render(tr)
	cands := []byte("## Candidates\n\n- Idea A\n- Idea B\n")
	merged := AppendCandidates(gen, cands)
	if !bytes.Contains(merged, []byte("## E-0001")) {
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
