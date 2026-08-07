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
// What the check buys is that neither can drift from the owned set in
// silence — which is how `Approach` came to be named required by a shipped
// skill while the normative design doc and the prose template both omitted
// it.
//
// Path constants and the reader live in verb_skill_factual_test.go, which
// already pins other facts in these same two files.

// backtickedToken matches each `...` span in a markdown table cell.
var backtickedToken = regexp.MustCompile("`([^`]*)`")

// skillTableRow returns the backticked tokens in kind's row of the table
// whose header cell names columnHeader. Both skills carry more than one
// table keyed by kind, so the search is scoped to the named one rather than
// taking the first row that matches — a row from the wrong table would
// otherwise satisfy the assertion.
//
// Fatal when the table or the row is missing: a silently empty result would
// let every assertion below pass over nothing.
func skillTableRow(t *testing.T, relPath, columnHeader, kind string) []string {
	t.Helper()
	lines := strings.Split(readVerbSkill(t, relPath), "\n")
	inTable := false
	for _, line := range lines {
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

// TestAiwfAddSkill_NamesNoSectionOutsideTheOwnedSets walks every `## X`
// token in the skill body — the per-kind prose paragraphs as well as the
// table — and requires each to name a section some kind actually owns.
//
// The table assertion above cannot see the prose, and the prose is where an
// author is told what to put in each section. A paragraph describing a
// retired section is the same defect as a stale table row, one surface over.
func TestAiwfAddSkill_NamesNoSectionOutsideTheOwnedSets(t *testing.T) {
	t.Parallel()
	owned := map[string]bool{}
	for _, k := range entity.AllKinds() {
		for _, section := range entity.RequiredSections(k) {
			owned[section] = true
		}
	}

	var stray []string
	for _, m := range backtickedToken.FindAllStringSubmatch(readVerbSkill(t, aiwfAddSkillPath), -1) {
		name, isHeading := strings.CutPrefix(m[1], "## ")
		// `## <Section>` is the skill's placeholder for "any heading the
		// author adds", not a claim about a specific section.
		if !isHeading || name == "<Section>" || owned[name] {
			continue
		}
		stray = append(stray, m[1])
	}
	if len(stray) > 0 {
		t.Errorf("%s names section(s) no kind owns: %s\n\n"+
			"Every `## X` the skill names must be a section entity.RequiredSections carries for some kind, "+
			"or the skill instructs an author to write a section nothing asks for.", aiwfAddSkillPath, strings.Join(stray, ", "))
	}
}

// TestAiwfShowSkill_BodyKeyRowsOpenWithTheOwnedSlugs pins the `aiwf-show`
// skill's body-key table against the owned set, slugified.
//
// The assertion is a prefix rather than equality because this table is
// deliberately a superset: `show`'s body map carries every `## ` heading a
// body has, so the rows list `work_log` and friends after the owned ones.
// A prefix still catches both failure modes that matter — a retired section
// left in place shifts the prefix, and a misspelled key breaks it.
func TestAiwfShowSkill_BodyKeyRowsOpenWithTheOwnedSlugs(t *testing.T) {
	t.Parallel()
	for _, k := range entity.AllKinds() {
		t.Run(string(k), func(t *testing.T) {
			t.Parallel()
			var want []string
			for _, section := range entity.RequiredSections(k) {
				want = append(want, entity.SectionSlug(section))
			}
			got := skillTableRow(t, aiwfShowSkillPath, "Body keys", string(k))
			if len(got) < len(want) {
				t.Fatalf("the aiwf-show skill's body-key row for %s names %d key(s), fewer than the %d the owned set carries: %v",
					k, len(got), len(want), got)
			}
			if diff := cmp.Diff(want, got[:len(want)]); diff != "" {
				t.Errorf("the aiwf-show skill's body-key row for %s does not open with the owned set's slugs (-want +got):\n%s\n\n"+
					"The row lists each kind's owned sections first, slugified, then any further keys a body carries. Update %s.",
					k, diff, aiwfShowSkillPath)
			}
		})
	}
}
