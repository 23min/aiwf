package policies

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/projectguidance"
)

// ceilingReader serves a fixture file set; a path absent from the map is
// absent from the tree.
func ceilingReader(files map[string]string) func(string) (string, bool) {
	return func(p string) (string, bool) {
		content, ok := files[p]
		return content, ok
	}
}

// words returns a text of n whitespace-separated words.
func words(n int) string {
	return strings.TrimSpace(strings.Repeat("w ", n))
}

// TestMeasureGuidanceLoad is M-0333 AC-4's measure: a host's handwritten
// primed load is its entry point outside the managed blocks plus the
// handwritten text of every document a required read reaches,
// transitively and once each; the managed blocks, with their imports
// expanded, are the generated figure; a conditional reference adds
// nothing to either. An import line is a directive, not text, so it
// counts no words itself.
func TestMeasureGuidanceLoad(t *testing.T) {
	t.Parallel()
	gStart, gEnd, _ := initrepo.GuidanceMarkers()
	files := map[string]string{
		"CLAUDE.md": words(10) + "\n\nSee [design](docs/design.md).\n\n" +
			gStart + "\n@.claude/aiwf-guidance.md\n" + gEnd + "\n",
		"AGENTS.md": words(5) + "\n\nRead [CLAUDE.md](CLAUDE.md) in full.\n\n" +
			gStart + "\n" + words(7) + "\n" + gEnd + "\n",
		".claude/aiwf-guidance.md": words(20),
		"docs/design.md":           words(1000),
	}
	table := map[guidanceRef]readKind{
		{From: "CLAUDE.md", To: "docs/design.md"}: readConditional,
		{From: "AGENTS.md", To: "CLAUDE.md"}:      readRequired,
	}

	claude, vs := measureGuidanceLoad(ceilingReader(files), "CLAUDE.md", table)
	if len(vs) != 0 {
		t.Fatalf("claude violations: %+v", vs)
	}
	// 10 words + "See [design](docs/design.md)." (2 words).
	if claude.Handwritten != 12 || claude.Generated != 20 {
		t.Errorf("claude = %+v, want 12 handwritten and 20 generated", claude)
	}

	codex, vs := measureGuidanceLoad(ceilingReader(files), "AGENTS.md", table)
	if len(vs) != 0 {
		t.Fatalf("codex violations: %+v", vs)
	}
	// AGENTS.md's 5 + "Read [CLAUDE.md](CLAUDE.md) in full." (4), plus the
	// required read of CLAUDE.md's handwritten 12. CLAUDE.md's own block is
	// not Codex's generated load: Codex does not expand a Claude import.
	if codex.Handwritten != 21 || codex.Generated != 7 {
		t.Errorf("codex = %+v, want 21 handwritten and 7 generated", codex)
	}
	if !equalStrings(codex.RequiredReads, []string{"CLAUDE.md"}) {
		t.Errorf("codex required reads = %v, want [CLAUDE.md]", codex.RequiredReads)
	}
}

// TestMeasureGuidanceLoad_Routing covers the routing shapes the model
// must handle: a transitive required read, a target two reads share, a
// cycle, a missing target, a reference the table does not classify, and an
// import, which only Claude Code follows.
func TestMeasureGuidanceLoad_Routing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		files       map[string]string
		table       map[guidanceRef]readKind
		handwritten int
		violations  []string
	}{
		{
			name: "a required read is followed transitively",
			files: map[string]string{
				"AGENTS.md": "[a](a.md)",
				"a.md":      "[b](b.md)",
				"b.md":      words(3),
			},
			table: map[guidanceRef]readKind{
				{From: "AGENTS.md", To: "a.md"}: readRequired,
				{From: "a.md", To: "b.md"}:      readRequired,
			},
			handwritten: 5,
		},
		{
			name: "a target two reads share is counted once",
			files: map[string]string{
				"AGENTS.md": "[a](a.md) [b](b.md)",
				"a.md":      "[s](s.md)",
				"b.md":      "[s](s.md)",
				"s.md":      words(4),
			},
			table: map[guidanceRef]readKind{
				{From: "AGENTS.md", To: "a.md"}: readRequired,
				{From: "AGENTS.md", To: "b.md"}: readRequired,
				{From: "a.md", To: "s.md"}:      readRequired,
				{From: "b.md", To: "s.md"}:      readRequired,
			},
			handwritten: 8,
		},
		{
			name: "a cycle terminates and counts each document once",
			files: map[string]string{
				"AGENTS.md": "[a](a.md)",
				"a.md":      "[back](AGENTS.md)",
			},
			table: map[guidanceRef]readKind{
				{From: "AGENTS.md", To: "a.md"}: readRequired,
				{From: "a.md", To: "AGENTS.md"}: readRequired,
			},
			handwritten: 2,
		},
		{
			name:        "a missing target is reported, not omitted",
			files:       map[string]string{"AGENTS.md": "[gone](gone.md)"},
			table:       map[guidanceRef]readKind{{From: "AGENTS.md", To: "gone.md"}: readRequired},
			handwritten: 1,
			violations:  []string{"gone.md"},
		},
		{
			name:        "an unclassified reference is reported, not omitted",
			files:       map[string]string{"AGENTS.md": "[new](new.md)", "new.md": words(50)},
			table:       map[guidanceRef]readKind{},
			handwritten: 1,
			violations:  []string{"new.md"},
		},
		{
			name:        "an import in handwritten text is a required read without a table entry",
			files:       map[string]string{"CLAUDE.md": "@extra.md\n", "extra.md": words(6)},
			table:       map[guidanceRef]readKind{},
			handwritten: 6,
		},
		{
			name:        "Codex reads an import as text and follows nothing",
			files:       map[string]string{"AGENTS.md": "@extra.md\n", "extra.md": words(6)},
			table:       map[guidanceRef]readKind{},
			handwritten: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entry := "AGENTS.md"
			if _, ok := tt.files["CLAUDE.md"]; ok {
				entry = "CLAUDE.md"
			}
			load, vs := measureGuidanceLoad(ceilingReader(tt.files), entry, tt.table)
			if load.Handwritten != tt.handwritten {
				t.Errorf("handwritten = %d, want %d", load.Handwritten, tt.handwritten)
			}
			var got []string
			for _, v := range vs {
				got = append(got, v.File)
				if v.Policy != "guidance-ceiling" {
					t.Errorf("violation Policy = %q, want guidance-ceiling", v.Policy)
				}
			}
			if !equalStrings(got, tt.violations) {
				t.Errorf("violation files = %v, want %v", got, tt.violations)
			}
		})
	}
}

// TestGuidanceCeiling_AboveCeilingFails pins the ceiling itself: a host
// whose handwritten primed load exceeds its ceiling fails, naming the
// host's entry point and both figures; at or under it passes.
func TestGuidanceCeiling_AboveCeilingFails(t *testing.T) {
	t.Parallel()
	files := map[string]string{"CLAUDE.md": words(11)}
	hosts := []guidanceHost{{Name: "claude-code", Entry: "CLAUDE.md", Ceiling: 10}}
	vs := guidanceCeilingViolations(ceilingReader(files), hosts, map[guidanceRef]readKind{})
	if len(vs) != 1 || vs[0].File != "CLAUDE.md" || !strings.Contains(vs[0].Detail, "11") || !strings.Contains(vs[0].Detail, "10") {
		t.Fatalf("want one violation on CLAUDE.md naming 11 over 10; got %+v", vs)
	}
	hosts[0].Ceiling = 11
	if vs := guidanceCeilingViolations(ceilingReader(files), hosts, map[guidanceRef]readKind{}); len(vs) != 0 {
		t.Errorf("at the ceiling: got %+v, want none", vs)
	}
}

// TestPolicy_GuidanceCeiling runs the ceiling over this repository and
// logs each host's figures, which is the measurement command M-0333's
// Validation records: go test -run TestPolicy_GuidanceCeiling -v ./internal/policies/
func TestPolicy_GuidanceCeiling(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	read := repoGuidanceReader(root)
	for _, h := range guidanceHosts {
		load, _ := measureGuidanceLoad(read, h.Entry, guidanceReadTable)
		t.Logf("%s: handwritten primed %d (ceiling %d), aiwf-generated %d, required reads %v",
			h.Name, load.Handwritten, h.Ceiling, load.Generated, load.RequiredReads)
	}
	runPolicy(t, PolicyGuidanceCeiling)
}

// TestRepoGuidanceReader pins the reader: a file reads from disk, a
// missing one reports absent, and the materialized Claude fragment —
// gitignored, so absent in CI — reads as the embedded source rendered,
// which is what the import loads.
func TestRepoGuidanceReader(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile := repoFileWriter(t, root)
	writeFile("CLAUDE.md", "fixture rules\n")
	read := repoGuidanceReader(root)
	if content, ok := read("CLAUDE.md"); !ok || content != "fixture rules\n" {
		t.Errorf("CLAUDE.md = %q, %v; want the file on disk", content, ok)
	}
	if _, ok := read("no/such/file.md"); ok {
		t.Error("a missing file must read as absent")
	}
	fragment, ok := read(".claude/aiwf-guidance.md")
	if !ok || len(strings.Fields(fragment)) == 0 {
		t.Error("the Claude fragment must read as the rendered embedded source")
	}
}

// TestMeasureGuidanceLoad_OutsideTheRepository pins what the model leaves
// out: an import from outside the repository is personal or global
// material and is not counted, and a web link or a bare anchor is not a
// reference into the repository. A missing entry point is reported.
func TestMeasureGuidanceLoad_OutsideTheRepository(t *testing.T) {
	t.Parallel()
	files := map[string]string{
		"CLAUDE.md": "@~/.claude/personal.md\n@/etc/global.md\nSee [web](https://example.com/x.md) and [here](#section).\n",
	}
	load, vs := measureGuidanceLoad(ceilingReader(files), "CLAUDE.md", map[guidanceRef]readKind{})
	if len(vs) != 0 || len(load.RequiredReads) != 0 || load.Handwritten != 4 {
		t.Errorf("load = %+v, violations %+v; want 4 handwritten words, no reads, no violations", load, vs)
	}
	if _, vs := measureGuidanceLoad(ceilingReader(files), "AGENTS.md", nil); len(vs) != 1 || vs[0].File != "AGENTS.md" {
		t.Errorf("a missing entry point: violations %+v, want one on AGENTS.md", vs)
	}
}

// TestMeasureGuidanceLoad_ReferenceForms pins the reference forms the
// model reads as CommonMark and Claude Code read them: an import inline in
// prose is a required read, one inside a code block is not; reference-style,
// titled and angle-bracket links are links; a directory exists.
func TestMeasureGuidanceLoad_ReferenceForms(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		claude      string
		handwritten int
		violations  []string
	}{
		{
			name:        "an inline import is a required read",
			claude:      "See @big.md for details.\n",
			handwritten: 4 + 100,
		},
		{
			name:        "an import inside a code block is not",
			claude:      "Example:\n\n```\n@big.md\n```\n",
			handwritten: 3,
		},
		{
			name:        "a reference-style link is a link the table must classify",
			claude:      "Read [the doc][d].\n\n[d]: new.md\n",
			handwritten: 5,
			violations:  []string{"new.md"},
		},
		{
			name:        "a titled link is a link the table must classify",
			claude:      "Read [the doc](new.md \"title\").\n",
			handwritten: 4,
			violations:  []string{"new.md"},
		},
		{
			name:        "an angle-bracket target resolves",
			claude:      "Read [the doc](<new.md>).\n",
			handwritten: 3,
			violations:  []string{"new.md"},
		},
		{
			name:        "a repository-rooted link resolves from the root",
			claude:      "Read [the doc](/new.md).\n",
			handwritten: 3,
			violations:  []string{"new.md"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			files := map[string]string{"CLAUDE.md": tt.claude, "big.md": words(100), "new.md": "x"}
			load, vs := measureGuidanceLoad(ceilingReader(files), "CLAUDE.md", map[guidanceRef]readKind{})
			if load.Handwritten != tt.handwritten {
				t.Errorf("handwritten = %d, want %d", load.Handwritten, tt.handwritten)
			}
			var got []string
			for _, v := range vs {
				got = append(got, v.File)
			}
			if !equalStrings(got, tt.violations) {
				t.Errorf("violation files = %v, want %v", got, tt.violations)
			}
		})
	}
}

// TestGuidanceCeilingViolations_ReportsASharedFindingOnce pins that a
// routing finding in a document both hosts read is reported once.
func TestGuidanceCeilingViolations_ReportsASharedFindingOnce(t *testing.T) {
	t.Parallel()
	files := map[string]string{"CLAUDE.md": "[x](gone.md)", "AGENTS.md": "[c](CLAUDE.md)"}
	hosts := []guidanceHost{{Name: "claude-code", Entry: "CLAUDE.md", Ceiling: 100}, {Name: "codex", Entry: "AGENTS.md", Ceiling: 100}}
	table := map[guidanceRef]readKind{{From: "AGENTS.md", To: "CLAUDE.md"}: readRequired, {From: "CLAUDE.md", To: "gone.md"}: readConditional}
	if vs := guidanceCeilingViolations(ceilingReader(files), hosts, table); len(vs) != 1 {
		t.Errorf("got %d violations %+v, want the missing target once", len(vs), vs)
	}
}

// TestRepoGuidanceReader_DirectoryExists pins that a link to a directory
// resolves: it exists and holds no text of its own.
func TestRepoGuidanceReader_DirectoryExists(t *testing.T) {
	t.Parallel()
	if content, ok := repoGuidanceReader(repoRoot(t))("docs/adr"); !ok || content != "" {
		t.Errorf("docs/adr = %q, %v; want an existing directory with no text", content, ok)
	}
}

// TestMeasureGuidanceLoad_FollowsTheRoutingBlock pins that the routing
// block's destinations are read before any task: the project router joins
// the handwritten load, the generated pack index the generated one, and
// what the router links to is classified like any other reference.
func TestMeasureGuidanceLoad_FollowsTheRoutingBlock(t *testing.T) {
	t.Parallel()
	rStart, rEnd, _ := projectguidance.RouteMarkers()
	files := map[string]string{
		"CLAUDE.md":            words(10) + "\n\n" + rStart + "\nRead [p](.guidance/project.md) then [i](.guidance/index.md).\n" + rEnd + "\n",
		".guidance/project.md": words(7) + " [t](../docs/testing.md)",
		".guidance/index.md":   words(30),
		"docs/testing.md":      words(500),
	}
	table := map[guidanceRef]readKind{
		{From: "CLAUDE.md", To: ".guidance/project.md"}:       readRequired,
		{From: "CLAUDE.md", To: ".guidance/index.md"}:         readGenerated,
		{From: ".guidance/project.md", To: "docs/testing.md"}: readConditional,
	}
	load, vs := measureGuidanceLoad(ceilingReader(files), "CLAUDE.md", table)
	if len(vs) != 0 {
		t.Fatalf("violations: %+v", vs)
	}
	// 10 handwritten + the router's 7 words and its one link word; the
	// block's own 4 words plus the index's 30 are generated.
	if load.Handwritten != 18 || load.Generated != 34 {
		t.Errorf("load = %+v, want 18 handwritten and 34 generated", load)
	}
}

// TestMeasureGuidanceLoad_RouterLinksDefaultToConditional pins the split
// rule: a link from the router is conditional unless the table declares
// it, while one from a host entry point's handwritten text must be
// classified; a router link must still resolve.
func TestMeasureGuidanceLoad_RouterLinksDefaultToConditional(t *testing.T) {
	t.Parallel()
	rStart, rEnd, _ := projectguidance.RouteMarkers()
	files := map[string]string{
		"CLAUDE.md":            "[h](host-link.md)\n\n" + rStart + "\n[p](.guidance/project.md)\n" + rEnd + "\n",
		".guidance/project.md": "[a](../docs/a.md) [gone](../docs/gone.md)",
		"docs/a.md":            words(500),
		"host-link.md":         words(5),
	}
	table := map[guidanceRef]readKind{{From: "CLAUDE.md", To: ".guidance/project.md"}: readRequired}
	load, vs := measureGuidanceLoad(ceilingReader(files), "CLAUDE.md", table)
	var got []string
	for _, v := range vs {
		got = append(got, v.File)
	}
	if want := []string{"host-link.md", "docs/gone.md"}; !equalStrings(got, want) {
		t.Errorf("violation files = %v, want %v", got, want)
	}
	if load.Handwritten != 3 {
		t.Errorf("handwritten = %d, want 3: a router link reads nothing into the primed load", load.Handwritten)
	}
}
