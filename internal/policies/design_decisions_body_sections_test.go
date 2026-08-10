package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"

	"github.com/google/go-cmp/cmp"
)

// designDecisionsPath is the Normative-tier design doc carrying the
// body-sections table this test holds to the owned set.
const designDecisionsPath = "docs/design/design-decisions.md"

// bodySectionsTableHeader anchors the table. Scoping the parse to the row
// following this header is what makes the assertion structural: the doc carries
// several tables, and a document-wide search for `## Goal` would find the prose
// around any of them.
const bodySectionsTableHeader = "| Kind | Body sections |"

// TestDesignDecisionsBodySectionsTableMatchesOwnedSet pins the Normative-tier
// body-sections table to what `aiwf add` actually scaffolds, which is what the
// table's own caption claims it lists.
//
// Equality, where the prose-template assertion is containment, because the two
// surfaces answer different questions. A prose template is a superset by design:
// it carries commentary and optional sections a scaffold has no business writing.
// This table is captioned as the scaffold's output, so a section it names that
// `aiwf add` does not write is not extra detail — it is the caption being false.
//
// The expected set is read from entity.RequiredSections rather than restated, so
// a kind gaining or losing a section fails here until the doc follows.
func TestDesignDecisionsBodySectionsTableMatchesOwnedSet(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, designDecisionsPath))
	if err != nil {
		t.Fatalf("reading %s: %v", designDecisionsPath, err)
	}
	table, unknown := parseBodySectionsTable(string(raw))
	if len(table) == 0 {
		t.Fatalf("%s: no rows parsed under %q; the table moved or its header was reworded",
			designDecisionsPath, bodySectionsTableHeader)
	}
	if len(unknown) > 0 {
		t.Errorf("%s: the body-sections table names row(s) %v matching no kind", designDecisionsPath, unknown)
	}
	for _, kind := range entity.AllKinds() {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			got, listed := table[kind]
			if !listed {
				t.Fatalf("%s: the body-sections table names no row for kind %q", designDecisionsPath, kind)
			}
			want := entity.RequiredSections(kind)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("%s: the body-sections table does not name what `aiwf add` scaffolds for %s (-want +table):\n%s",
					designDecisionsPath, kind, diff)
			}
		})
	}
}

// TestParseBodySectionsTable pins the oracle the doc assertion rests on. A parser
// that silently dropped a row, or read a `### ` illustration as a section, would
// make the comparison above agree with the doc for the wrong reason.
func TestParseBodySectionsTable(t *testing.T) {
	t.Parallel()
	doc := strings.Join([]string{
		"prose above the table is ignored",
		bodySectionsTableHeader,
		"|---|---|",
		"| Epic | `## Goal` / `## Scope` |",
		"| Milestone | `## Goal` (per-AC `### AC-N — <title>`) |",
		"| Initiative | `## Goal` |",
		"| malformed-row-with-one-cell",
		"",
		"prose below the table is outside it",
	}, "\n")

	table, unknown := parseBodySectionsTable(doc)

	want := map[entity.Kind][]string{
		entity.KindEpic:      {"Goal", "Scope"},
		entity.KindMilestone: {"Goal"}, // the `### AC-N` token is not a section
	}
	if diff := cmp.Diff(want, table); diff != "" {
		t.Errorf("parsed table (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"initiative"}, unknown); diff != "" {
		t.Errorf("unrecognized rows (-want +got):\n%s", diff)
	}
}

// parseBodySectionsTable reads the rows under bodySectionsTableHeader and returns
// each kind's listed section names in table order, plus the display names of any
// rows matching no kind.
//
// A cell names its sections as backticked `## <name>` tokens. Tokens that are not
// section headings — a per-AC `### AC-N` illustration, an inline code reference —
// carry a different prefix and are skipped rather than parsed as sections.
func parseBodySectionsTable(doc string) (sectionsByKind map[entity.Kind][]string, unknownRows []string) {
	out := map[entity.Kind][]string{}
	var unknown []string
	inTable := false
	for line := range strings.SplitSeq(doc, "\n") {
		if strings.HasPrefix(line, bodySectionsTableHeader) {
			inTable = true
			continue
		}
		if !inTable {
			continue
		}
		if !strings.HasPrefix(line, "|") {
			break // the table ended
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if len(cells) < 2 {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(cells[0]))
		if name == "" || strings.HasPrefix(name, "---") {
			continue // the header's separator row
		}
		kind, ok := kindByName(name)
		if !ok {
			unknown = append(unknown, name)
			continue
		}
		out[kind] = sectionsInCell(cells[1])
	}
	return out, unknown
}

// sectionsInCell extracts the `## <name>` section headings a table cell names.
func sectionsInCell(cell string) []string {
	var sections []string
	for i, tok := range strings.Split(cell, "`") {
		if i%2 == 0 {
			continue // outside a backtick pair
		}
		if rest, ok := strings.CutPrefix(tok, "## "); ok {
			sections = append(sections, strings.TrimSpace(rest))
		}
	}
	return sections
}

// kindByName resolves a table row's display name to its kind.
func kindByName(name string) (entity.Kind, bool) {
	for _, k := range entity.AllKinds() {
		if string(k) == name {
			return k, true
		}
	}
	return "", false
}
