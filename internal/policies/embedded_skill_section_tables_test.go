package policies

import (
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/entity"
)

// embedded_skill_section_tables_test.go — M-0305/AC-4. Two shipped verb
// skills restate the per-kind body-section set: `aiwf-add` tells an author
// which sections to write, and `aiwf-show` tells a JSON consumer which keys
// to expect. Both materialize into every consumer repo, and both are what an
// authoring assistant reads rather than the Go table.
//
// They are checked against `entity.RequiredSections` rather than generated
// from it, because each is prose with commentary a generator would flatten.
//
// Path constants and the reader live in verb_skill_factual_test.go, which
// already pins other facts in these same two files.

// backtickedToken matches each `...` span in a markdown table cell.
var backtickedToken = regexp.MustCompile("`([^`]*)`")

// skillTableRow returns the backticked tokens in kind's row of the table
// whose header cell names columnHeader. Both skills carry more than one
// table keyed by kind, so the search is scoped to the named one.
//
// Fatal when the table or the row is missing: a silently empty result would
// let every assertion below pass over nothing.
func skillTableRow(t *testing.T, relPath, columnHeader, kind string) []string {
	t.Helper()
	inTable := false
	for _, line := range strings.Split(readVerbSkill(t, relPath), "\n") {
		cells := strings.Split(line, "|")
		if len(cells) < 3 {
			inTable = false
			continue
		}
		if strings.TrimSpace(cells[2]) == columnHeader {
			inTable = true
			continue
		}
		if !inTable || strings.TrimSpace(cells[1]) != kind {
			continue
		}
		var tokens []string
		for _, m := range backtickedToken.FindAllStringSubmatch(cells[2], -1) {
			tokens = append(tokens, m[1])
		}
		return tokens
	}
	t.Fatalf("%s has no %q table row for kind %q; the skill must document every kind's body sections",
		relPath, columnHeader, kind)
	return nil
}

// TestAiwfAddSkill_RequiredSectionTableMatchesOwnedSet pins the `aiwf-add`
// skill's required-body-sections table against the owned definition, exactly
// and in order. That table's own column heading is "Required body sections",
// so containment would be the wrong claim — a section it names that the set
// does not is an instruction to write a section nothing asks for.
func TestAiwfAddSkill_RequiredSectionTableMatchesOwnedSet(t *testing.T) {
	t.Parallel()
	for _, k := range entity.AllKinds() {
		t.Run(string(k), func(t *testing.T) {
			t.Parallel()
			var want []string
			for _, section := range entity.RequiredSections(k) {
				want = append(want, "## "+section)
			}
			got := skillTableRow(t, aiwfAddSkillPath, "Required body sections", string(k))
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("the aiwf-add skill's required-sections row for %s disagrees with the owned set (-want +got):\n%s\n\n"+
					"entity.RequiredSections owns the set; %s describes it. Update the table row.", k, diff, aiwfAddSkillPath)
			}
		})
	}
}

// TestAiwfShowSkill_BodyKeyRowsNameTheOwnedSlugs pins that the `aiwf-show`
// skill's body-key table names every slug the owned set implies, so a key a
// consumer is told to read resolves against a real `aiwf show` envelope. It
// is what catches a misspelled slug — the row said `whats_missing` where
// `SectionSlug` derives `what_s_missing`, and no envelope ever carried it.
//
// Containment, not equality: that table describes what `show` emits, which
// is every `## ` heading a body has, so the rows legitimately name keys the
// owned set knows nothing about. The limit that buys is worth stating —
// a key listed here that `show` never emits is NOT caught, and would need a
// fixture driving the real projection rather than a table read.
func TestAiwfShowSkill_BodyKeyRowsNameTheOwnedSlugs(t *testing.T) {
	t.Parallel()
	for _, k := range entity.AllKinds() {
		t.Run(string(k), func(t *testing.T) {
			t.Parallel()
			got := skillTableRow(t, aiwfShowSkillPath, "Body keys", string(k))
			named := make(map[string]bool, len(got))
			for _, key := range got {
				named[key] = true
			}
			for _, section := range entity.RequiredSections(k) {
				if slug := entity.SectionSlug(section); !named[slug] {
					t.Errorf("the aiwf-show skill's body-key row for %s omits %q, the slug `## %s` derives.\n"+
						"Row names: %v\nUpdate %s.", k, slug, section, got, aiwfShowSkillPath)
				}
			}
		})
	}
}
