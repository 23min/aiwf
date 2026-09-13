package policies

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/entity"
)

// section_set_single_source.go — M-0332/AC-1. entity.RequiredSections owns
// each kind's required body sections. A surface that restates the set as a
// per-kind table is a second copy of it, free to drift, and the write seams
// now refuse a body omitting a section — so the set is learnable by running
// the verb and the copies buy nothing.
//
// The scan is scoped to the table shape because that is what states the set
// as a set. Prose advising what to write *inside* a section is not a copy of
// the membership and is not matched.
//
// Both sides derive: the section names come from entity.RequiredSections, so
// a section added to a kind's set changes what the scan looks for. Neither
// artefact can move without the other following, and a reword of the prose
// around a table can neither break the scan nor evade it.

// sectionSetCorpus names the trees where a restatement would be
// consequential: what aiwf ships into a consumer repo, and the normative
// docs held in lockstep with the code.
//
// The exploratory and forward-looking tiers are absent by intent. A table
// there records what someone was thinking, not what a reader should believe
// about the current kernel. Archived subtrees are absent for the same reason
// under ADR-0004 — a frozen snapshot is not a claim about today.
//
// TestSectionSetCorpusRootsExist holds these against the real tree, so a
// root renamed out from under the scan fails rather than quietly narrowing
// what it reads.
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
// Matching is case-insensitive on the kind cell alone, since which case a
// table picked would not change what it restates.
func kindStatedByRow(line string) (entity.Kind, bool) {
	row := strings.TrimSpace(line)
	if !strings.HasPrefix(row, "|") {
		return "", false
	}
	cells := strings.Split(row, "|")
	if len(cells) < 3 {
		return "", false
	}
	named := strings.Trim(cells[1], " `*_")
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

// PolicySectionSetSingleSource reports every markdown table row in the
// corpus that restates a kind's required section set.
//
// A corpus root that does not exist is silent, which is walkMarkdown's own
// behaviour and is what lets the firing fixture run against a synthetic tree
// where most of the corpus is legitimately absent. Whether the real tree
// still carries every root is a separate claim, made by
// TestSectionSetCorpusRootsExist.
func PolicySectionSetSingleSource(root string) ([]Violation, error) {
	var out []Violation
	for _, rel := range sectionSetCorpus {
		files, err := walkMarkdown(filepath.Join(root, rel))
		if err != nil { //coverage:ignore walkMarkdown fails only on a mid-walk IO fault; the corpus roots are tracked paths, so reaching this needs fault injection
			return nil, err
		}
		for _, f := range files {
			for i, line := range strings.Split(string(f.Contents), "\n") {
				k, ok := kindStatedByRow(line)
				if !ok {
					continue
				}
				out = append(out, Violation{
					Policy: "section-set-single-source",
					File:   relTo(root, f.AbsPath),
					Line:   i + 1,
					Detail: fmt.Sprintf(
						"table row restates %s's required section set: %s — entity.RequiredSections owns it and `aiwf template <kind>` prints it, so delete the row rather than correcting it",
						k, strings.TrimSpace(line),
					),
				})
			}
		}
	}
	return out, nil
}
