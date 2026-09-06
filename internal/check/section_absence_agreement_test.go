package check

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// TestSectionPresence_EverySurfaceAnswersAlike pins M-0329/AC-4: every
// rule that asks whether a body carries a section resolves to one
// parser, so none of them can answer differently about the same body.
//
// Three surfaces ask it, and each is made to reveal its own answer by a
// body shaped so that its verdict turns on nothing else:
//
//   - AbsentRequiredSections reports the section, or does not.
//   - EmptyRequiredSections judges content, so it can only report a
//     section it found — on a body whose section is present and empty,
//     reporting it means it saw the heading.
//   - milestone-done-empty-release-note fires on absent and on empty
//     alike, so it reveals nothing on an empty section — on one carrying
//     real prose, staying silent means it saw the heading.
//
// The spellings are the axis because they are where two parsers
// diverged: the write-time guards once matched a regexp tolerant of any
// whitespace after `##`, while the check rules matched the literal
// `"## "`. Measured on `##\tGoal`, the guards reported the section
// present and the parser reported it absent.
//
// The assertion is agreement, never a hardcoded expectation. Which
// spellings the shared parser accepts is its business; that every rule
// gets the same answer is this milestone's.
func TestSectionPresence_EverySurfaceAnswersAlike(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		heading string
	}{
		{"canonical", "## "},
		{"tab after the hashes", "##\t"},
		{"two spaces after the hashes", "##  "},
		{"no space after the hashes", "##"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// `Goal` present and empty; the rest of the kind's set filled,
			// so nothing but this heading decides either verdict.
			emptyGoal := []byte(tc.heading + "Goal\n\n\n\n" + tc.heading + "Acceptance criteria\n\nprose\n")
			guardSees := len(AbsentRequiredSections(entity.KindMilestone, emptyGoal)) == 0
			emptinessSees := len(EmptyRequiredSections(entity.KindMilestone, emptyGoal)) > 0

			// `Release note` carrying real prose: the rule is silent only
			// if it found the heading and read what is under it.
			writtenNote := tc.heading + "Goal\n\nprose\n\n" + tc.heading + ReleaseNoteSectionHeading + "\n\nThe verb now accepts a flag.\n"
			root := writeReleaseNoteFixture(t, "done", writtenNote)
			ruleSees := len(milestoneDoneEmptyReleaseNote(loadReleaseNoteTree(t, root))) == 0

			if guardSees != emptinessSees || guardSees != ruleSees {
				t.Errorf("the rules disagree about whether %q is a heading — write-time guard sees it: %v; entity-body-empty sees it: %v; milestone-done-empty-release-note sees it: %v",
					tc.heading, guardSees, emptinessSees, ruleSees)
			}
		})
	}
}
