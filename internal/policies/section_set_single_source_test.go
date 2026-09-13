package policies

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// section_set_single_source_test.go — M-0332/AC-1. entity.RequiredSections
// owns each kind's required body sections. A surface that restates the set as
// a per-kind table is a second copy of it, free to drift, and the write seams
// now refuse a body omitting a section — so the set is learnable by running
// the verb and the copies buy nothing.
//
// The census is scoped to the table shape because that is what states the set
// as a set. Prose advising what to write *inside* a section is not a copy of
// the membership and is not matched.
//
// Both sides derive: the section names come from entity.RequiredSections, so
// a section added to a kind's set changes what the census looks for. Neither
// artefact can move without the other following, and a reword of the prose
// around a table can neither break the census nor evade it.

// sectionSetCorpus is where a restatement would be consequential: the trees
// aiwf ships into a consumer repo, and the normative docs held in lockstep
// with the code.
//
// The exploratory and forward-looking tiers are deliberately absent. A table
// there records what someone was thinking, not what a reader should believe
// about the current kernel. Archived subtrees are absent for the same reason
// under ADR-0004 — a frozen snapshot is not a claim about today.
var sectionSetCorpus = []string{
	filepath.Join("internal", "skills", "embedded"),
	filepath.Join("internal", "skills", "embedded-rituals"),
	filepath.Join("internal", "skills", "embedded-guidance"),
	filepath.Join("docs", "adr"),
	filepath.Join("docs", "design"),
	filepath.Join("docs", "architecture.md"),
	filepath.Join("docs", "overview.md"),
	filepath.Join("docs", "workflows.md"),
	filepath.Join("docs", "skill-author-guide.md"),
}

// kindStatedByRow reports the kind a markdown table row enumerates the
// required sections of, if any: its first cell names the kind and its
// remaining cells carry every section that kind requires.
//
// Matching is case-insensitive on the kind cell alone. The two tables this
// milestone retires key their rows differently ("Gap" against "gap"), and
// which case a future table picked would not change what it restates.
func kindStatedByRow(line string) (entity.Kind, bool) {
	row := strings.TrimSpace(line)
	if !strings.HasPrefix(row, "|") {
		return "", false
	}
	cells := strings.Split(row, "|")
	if len(cells) < 3 {
		return "", false
	}
	named := strings.TrimSpace(cells[1])
	for _, k := range entity.AllKinds() {
		if !strings.EqualFold(named, string(k)) {
			continue
		}
		rest := strings.Join(cells[2:], " ")
		for _, section := range entity.RequiredSections(k) {
			if !strings.Contains(rest, section) {
				return "", false
			}
		}
		return k, true
	}
	return "", false
}

// sectionSetCorpusFiles returns every markdown file in the corpus. A root it
// cannot read is fatal rather than empty: a census reporting nothing for
// bytes it never saw is worse than no census, because the clean verdict is
// what stops the next reader looking.
func sectionSetCorpusFiles(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	var files []string
	for _, rel := range sectionSetCorpus {
		full := filepath.Join(root, rel)
		info, err := os.Stat(full)
		if err != nil {
			t.Fatalf("stat %s: %v", rel, err)
		}
		if !info.IsDir() {
			files = append(files, rel)
			continue
		}
		err = filepath.WalkDir(full, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "archive" {
					return fs.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(d.Name(), ".md") {
				fromRoot, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				files = append(files, fromRoot)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", rel, err)
		}
	}
	if len(files) == 0 {
		t.Fatal("the section-set corpus matched no markdown files; the roots are wrong")
	}
	return files
}

// TestNoSurfaceRestatesTheSectionSetAsAPerKindTable pins that the required
// section set is stated by the kernel and by no shipped skill or normative
// design doc.
//
// The claim is the rule, not the two passages M-0332 deleted to reach it: a
// table re-added to a third surface fails this exactly as a restored one
// would, and so does a table *corrected* to agree with the kernel — which is
// the point, since a passage rewritten to agree is still a second copy.
func TestNoSurfaceRestatesTheSectionSetAsAPerKindTable(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	var found []string
	for _, rel := range sectionSetCorpusFiles(t) {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			if k, ok := kindStatedByRow(line); ok {
				found = append(found, fmt.Sprintf("%s:%d states %s's section set: %s",
					rel, i+1, k, strings.TrimSpace(line)))
			}
		}
	}
	if len(found) > 0 {
		t.Errorf("%d table row(s) restate a kind's required section set:\n%s\n\n"+
			"entity.RequiredSections owns the set and the write seams enforce it, so a reader "+
			"learns it by running `aiwf template <kind>`. Delete the row rather than correcting it.",
			len(found), strings.Join(found, "\n"))
	}
}

// TestKindStatedByRow pins the detector itself. The census above asserts an
// absence, so on a clean tree it never reaches a matching row — a detector
// that matched nothing at all would pass it just as well. These cases are
// what separate the two.
//
// The matching row is built from entity.RequiredSections rather than spelled
// out, so it tracks the set the census reads and cannot pin a stale shape.
func TestKindStatedByRow(t *testing.T) {
	t.Parallel()

	var cells []string
	for _, section := range entity.RequiredSections(entity.KindGap) {
		cells = append(cells, "`## "+section+"`")
	}
	statesTheSet := "| gap | " + strings.Join(cells, ", ") + " |"

	for _, tc := range []struct {
		name string
		line string
		want entity.Kind
	}{
		{"a row carrying the kind's sections names that kind", statesTheSet, entity.KindGap},
		{"a row keyed by a kind but carrying other content is not a restatement",
			"| gap | `what_s_missing`, `why_it_matters`, plus author-added sections |", ""},
		{"prose naming every section of a kind is not a table row",
			"**Gaps.** `## What's missing` is the defect; `## Why it matters` is the consequence.", ""},
		{"a row whose first cell names no kind is not a restatement",
			"| R-AUDIT-0085 | `## What's missing` / `## Why it matters` |", ""},
		{"a row too short to carry cells is not a restatement", "|", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := kindStatedByRow(tc.line)
			if ok != (tc.want != "") {
				t.Fatalf("kindStatedByRow(%q) matched = %v, want %v", tc.line, ok, tc.want != "")
			}
			if got != tc.want {
				t.Errorf("kindStatedByRow(%q) = %q, want %q", tc.line, got, tc.want)
			}
		})
	}
}
