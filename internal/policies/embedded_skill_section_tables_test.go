package policies

import (
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/entity"
)

// embedded_skill_section_tables_test.go — M-0305/AC-4. The `aiwf-show` skill
// tells a JSON consumer which body keys an envelope carries, so it names in
// slug form the same set `entity.RequiredSections` owns. It materializes into
// every consumer repo, and it is what a consuming assistant reads rather than
// the Go table.
//
// It is checked against `entity.RequiredSections` rather than generated from
// it, because it is prose with commentary a generator would flatten.
//
// Path constants and the reader live in skill_fixture_helpers_test.go.

// backtickedToken matches each `...` span in a markdown table cell.
var backtickedToken = regexp.MustCompile("`([^`]*)`")

// TestAiwfShowSkill_BodyKeyRowsNameTheOwnedSlugs pins that the `aiwf-show`
// skill's body-key table names every slug the owned set implies, so a key a
// consumer is told to read resolves against a real `aiwf show` envelope. It
// is what catches a misspelled slug — the row said `whats_missing` where
// `SectionSlug` derives `what_s_missing`, and no envelope ever carried it.
//
// Each row states its keys in one shape: the keys every entity of the kind
// carries, then a hedge, then — after an em dash — the ones the shipped prose
// template contributes when the author keeps those sections. Only the part
// before the em dash is unconditional, and it must equal the owned set.
//
// Both directions matter and they fail differently. A row omitting an owned
// slug tells a consumer a key does not exist — the misspelling this test was
// written for, where the row said `whats_missing` against the `what_s_missing`
// that `SectionSlug` derives. A row naming a key unconditionally that a legal
// entity does not carry is the opposite error and the more damaging one: an
// assistant reading it finds the key absent, concludes the entity is malformed,
// and "repairs" it by writing an empty heading — which for a born-complete kind
// is an error-severity `entity-body-empty` finding that blocks that consumer's
// push. Sections a template marks optional live after the em dash for exactly
// that reason.
func TestAiwfShowSkill_BodyKeyRowsNameTheOwnedSlugs(t *testing.T) {
	t.Parallel()
	for _, k := range entity.AllKinds() {
		t.Run(string(k), func(t *testing.T) {
			t.Parallel()
			unconditional := skillTableRowUnconditionalKeys(t, string(k))
			var want []string
			for _, section := range entity.RequiredSections(k) {
				want = append(want, entity.SectionSlug(section))
			}
			if diff := cmp.Diff(want, unconditional); diff != "" {
				t.Errorf("the aiwf-show skill's body-key row for %s does not name the owned set unconditionally (-want +row):\n%s\n"+
					"Keys a template contributes only when its section is kept belong after the em dash. Update %s.",
					k, diff, aiwfShowSkillPath)
			}
		})
	}
}

// skillTableRowUnconditionalKeys returns the backticked keys a body-key row
// names before its em dash — the ones it claims every entity of the kind
// carries. Keys after the em dash are the shipped template's conditional
// additions and are deliberately excluded.
func skillTableRowUnconditionalKeys(t *testing.T, kind string) []string {
	t.Helper()
	for _, line := range strings.Split(readVerbSkill(t, aiwfShowSkillPath), "\n") {
		cells := strings.Split(line, "|")
		if len(cells) < 3 || strings.TrimSpace(cells[1]) != kind {
			continue
		}
		unconditional, _, _ := strings.Cut(cells[2], "—")
		var keys []string
		for _, m := range backtickedToken.FindAllStringSubmatch(unconditional, -1) {
			if !strings.Contains(m[1], "<") { // the `## <Section>` hedge is not a key
				keys = append(keys, m[1])
			}
		}
		return keys
	}
	t.Fatalf("%s has no body-key row for kind %q", aiwfShowSkillPath, kind)
	return nil
}
