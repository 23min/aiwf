package roadmap

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
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

// G-0115 tests: normalizeEntityLinks rewrites entity-file link URLs in
// Goal body prose so they resolve relative to repo root (ROADMAP.md's
// emission location) and use the entity's canonical on-disk slug.

func TestNormalizeEntityLinks_RewritesNarrowLegacySlug(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindGap, ID: "G-0055", Path: "work/gaps/G-0055-canonical-slug.md"},
		},
	}
	in := []byte("See [G-0055](../../gaps/G-055-old-narrow-slug.md) for context.")
	want := "See [G-0055](work/gaps/G-0055-canonical-slug.md) for context."
	if got := string(normalizeEntityLinks(in, tr)); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestNormalizeEntityLinks_RewritesWrongRelativePath(t *testing.T) {
	t.Parallel()
	// ROADMAP.md is at repo root; a Goal body's `../../../docs/adr/...`
	// (correct relative to the epic file) becomes wrong when copied
	// into ROADMAP. Normalization replaces it with the canonical
	// repo-root-relative path.
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindADR, ID: "ADR-0010", Path: "docs/adr/ADR-0010-branch-model.md"},
		},
	}
	in := []byte("Per [ADR-0010](../../../docs/adr/ADR-0010-branch-model.md), the kernel enforces ...")
	want := "Per [ADR-0010](docs/adr/ADR-0010-branch-model.md), the kernel enforces ..."
	if got := string(normalizeEntityLinks(in, tr)); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestNormalizeEntityLinks_CanonicalizesNarrowLinkText(t *testing.T) {
	t.Parallel()
	// Even if the author wrote the link text narrow ([G-055]), the
	// renderer emits the canonical 4-digit form per ADR-0008.
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindGap, ID: "G-0055", Path: "work/gaps/G-0055-canonical.md"},
		},
	}
	in := []byte("Per [G-055](some/old/path.md), see the empirical evidence.")
	want := "Per [G-0055](work/gaps/G-0055-canonical.md), see the empirical evidence."
	if got := string(normalizeEntityLinks(in, tr)); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestNormalizeEntityLinks_LeavesUnknownEntityAlone(t *testing.T) {
	t.Parallel()
	// Entity id doesn't resolve in the tree (typo, archived without
	// loading, external ref) — link passes through unchanged so
	// `aiwf check`'s dangling-refs policy surfaces it elsewhere.
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindGap, ID: "G-0001", Path: "work/gaps/G-0001-foo.md"},
		},
	}
	in := []byte("See [G-9999](some/path.md) for context.")
	want := "See [G-9999](some/path.md) for context."
	if got := string(normalizeEntityLinks(in, tr)); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestNormalizeEntityLinks_LeavesBareIDAlone(t *testing.T) {
	t.Parallel()
	// Bare-id refs (no markdown link syntax) are valid per the
	// loader-resolves convention and require no rewriting.
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindGap, ID: "G-0055", Path: "work/gaps/G-0055-foo.md"},
		},
	}
	in := []byte("See G-0055 for context, and don't touch this [external link](https://example.com).")
	want := "See G-0055 for context, and don't touch this [external link](https://example.com)."
	if got := string(normalizeEntityLinks(in, tr)); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestNormalizeEntityLinks_RewritesMultipleLinks(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{
		Entities: []*entity.Entity{
			{Kind: entity.KindGap, ID: "G-0055", Path: "work/gaps/G-0055-tdd-policy.md"},
			{Kind: entity.KindMilestone, ID: "M-0070", Path: "work/epics/E-0018-foo/M-0070-doctor.md"},
			{Kind: entity.KindADR, ID: "ADR-0010", Path: "docs/adr/ADR-0010-branch-model.md"},
		},
	}
	in := []byte("See [G-0055](old/G-055.md), [M-0070](M-070.md), and [ADR-0010](../../../docs/adr/ADR-0010-foo.md).")
	want := "See [G-0055](work/gaps/G-0055-tdd-policy.md), [M-0070](work/epics/E-0018-foo/M-0070-doctor.md), and [ADR-0010](docs/adr/ADR-0010-branch-model.md)."
	if got := string(normalizeEntityLinks(in, tr)); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

// TestNormalizeEntityLinks_AllEntityKinds: the regex covers every entity
// kind in the kernel (E-/M-/G-/D-/C-/ADR-). Table-driven over each so
// adding a new kind to the regex without a test fails CI.
func TestNormalizeEntityLinks_AllEntityKinds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		id     string
		kind   entity.Kind
		path   string
		inURL  string
		wantID string // canonicalized
	}{
		{"epic", "E-0030", entity.KindEpic, "work/epics/E-0030-foo/epic.md", "old/E-030.md", "E-0030"},
		{"milestone", "M-0070", entity.KindMilestone, "work/epics/E-0018-bar/M-0070-baz.md", "old/M-070.md", "M-0070"},
		{"gap", "G-0055", entity.KindGap, "work/gaps/G-0055-foo.md", "old/G-055.md", "G-0055"},
		{"decision", "D-0001", entity.KindDecision, "work/decisions/D-0001-foo.md", "old/D-1.md", "D-0001"},
		{"contract", "C-0001", entity.KindContract, "work/contracts/C-0001-foo.md", "old/C-001.md", "C-0001"},
		{"adr", "ADR-0010", entity.KindADR, "docs/adr/ADR-0010-foo.md", "../../../docs/adr/ADR-0010-foo.md", "ADR-0010"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := &tree.Tree{Entities: []*entity.Entity{{Kind: tc.kind, ID: tc.id, Path: tc.path}}}
			in := []byte("Per [" + tc.id + "](" + tc.inURL + ") see ...")
			want := "Per [" + tc.wantID + "](" + tc.path + ") see ..."
			if got := string(normalizeEntityLinks(in, tr)); got != want {
				t.Errorf("got %q\nwant %q", got, want)
			}
		})
	}
}

// TestRender_NormalizesEntityLinksInGoal: integration test. An epic
// Goal section containing a broken entity-file link is normalized
// when rendered into the roadmap. Closes G-0115's user-visible bug:
// `aiwf render roadmap --write` no longer produces dangling-refs.
func TestRender_NormalizesEntityLinksInGoal(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := writeEpicFile(t, root, "E-0030-chokepoint", `---
id: E-0030
title: Branch chokepoint
status: proposed
---

## Goal

Make ADR-0010 enforceable. See [ADR-0010](../../../docs/adr/ADR-0010-branch-model.md) for the model and [G-0055](../../gaps/G-055-narrow.md) for context.

## Scope

unrelated
`)
	tr := &tree.Tree{
		Root: root,
		Entities: []*entity.Entity{
			{Kind: entity.KindEpic, ID: "E-0030", Title: "Branch chokepoint", Status: "proposed", Path: path},
			{Kind: entity.KindADR, ID: "ADR-0010", Path: "docs/adr/ADR-0010-branch-model.md"},
			{Kind: entity.KindGap, ID: "G-0055", Path: "work/gaps/G-0055-canonical.md"},
		},
	}
	got := string(Render(tr))

	// Both links should be rewritten to canonical repo-root paths.
	wantADR := "[ADR-0010](docs/adr/ADR-0010-branch-model.md)"
	wantGap := "[G-0055](work/gaps/G-0055-canonical.md)"
	for _, want := range []string{wantADR, wantGap} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered roadmap missing rewritten link %q:\n%s", want, got)
		}
	}
	// The original broken links must NOT appear in the output.
	for _, badURL := range []string{"../../../docs/adr/ADR-0010", "../../gaps/G-055-narrow"} {
		if strings.Contains(got, badURL) {
			t.Errorf("rendered roadmap still contains broken URL fragment %q:\n%s", badURL, got)
		}
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
