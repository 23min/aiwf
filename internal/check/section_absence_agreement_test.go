package check

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// TestSectionAbsence_OneAnswerForEveryRuleThatAsks pins M-0329/AC-4:
// the rules asking whether a body carries a section resolve to one
// predicate, so they cannot answer differently about the same heading.
//
// Two rules ask it and they used to ask it two ways. The write-time
// guards scanned with a regexp tolerant of any whitespace after `##`;
// milestone-done-empty-release-note read entity.ParseBodySections,
// which matches the literal `"## "` and nothing else. Measured on
// `##\tGoal`: the guards reported the section present, the parser
// reported it absent — the same body, two answers.
//
// The heading spellings are the axis because they are where the two
// scanners diverged. Each row asks both surfaces about one spelling and
// requires a single verdict; the assertion is agreement, not a
// hardcoded expectation, so a future parser change moves both together
// or fails here.
func TestSectionAbsence_OneAnswerForEveryRuleThatAsks(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		heading string
	}{
		{"canonical", "## "},
		{"tab after the hashes", "##\t"},
		{"two spaces after the hashes", "##  "},
		{"trailing whitespace", "## "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// One body, two questions about it. The milestone's own
			// required set carries `Goal`, and the release-note rule
			// asks about `Release note`, so the body spells both the
			// same way and each surface is asked about its own.
			body := []byte(tc.heading + "Goal\n\nprose\n\n" +
				tc.heading + "Acceptance criteria\n\nprose\n\n" +
				tc.heading + ReleaseNoteSectionHeading + "\n\nThe verb now accepts a flag.\n")

			guardSaysAbsent := len(AbsentRequiredSections(entity.KindMilestone, body)) > 0

			root := writeReleaseNoteFixture(t, "done", string(body))
			ruleSaysAbsent := len(milestoneDoneEmptyReleaseNote(loadReleaseNoteTree(t, root))) > 0

			if guardSaysAbsent != ruleSaysAbsent {
				t.Errorf("the two rules disagree about %q: the write-time guard reports the section absent: %v; milestone-done-empty-release-note reports it unwritten: %v",
					tc.heading, guardSaysAbsent, ruleSaysAbsent)
			}
		})
	}
}
