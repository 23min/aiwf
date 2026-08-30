package check

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// CodeMilestoneDoneEmptyReleaseNote is reported when a milestone reaches `done`
// carrying a `## Release note` section nobody filled in.
const CodeMilestoneDoneEmptyReleaseNote = "milestone-done-empty-release-note"

// ReleaseNoteSectionHeading is the milestone-spec section this rule reads. It is
// exported so a policy can resolve it against the heading the shipped template
// actually carries: renaming the section coherently across the template and the
// rituals would otherwise leave this rule looking for a heading no spec has, and
// it would stop firing permanently with nothing red.
const ReleaseNoteSectionHeading = "Release note"

// releaseNoteSectionSlug is that heading in the slug form ParseBodySections
// keys by.
var releaseNoteSectionSlug = entity.SectionSlug(ReleaseNoteSectionHeading)

// milestoneDoneEmptyReleaseNote fires (error) when a non-archived milestone at
// `done` has no `## Release note` an author filled in — the section absent, or
// present and carrying nothing but whitespace, headings, or the template's own
// guidance comment.
//
// The section is where a milestone records its own user-visible delta, and the
// epic wrap composes the epic's changelog entry from those notes and copies that
// entry verbatim into the changelog. An empty one is a shipped change that
// reaches a release described by nobody who did the work — measured on a real
// release that shipped three such changes undocumented.
//
// The rule governs two surfaces from one definition, because `promote` runs the
// projection findings as preconditions and gates on error severity
// (`verb.Promote` -> `check.HasErrors`): it reports standing state in
// `aiwf check`, and it refuses the `done` promote that would produce that state.
// Error severity is what makes the second surface real — at warning it would
// report only after the fact, and the milestone wrap pushes before it promotes.
//
// A milestone with nothing user-facing is not blocked, it is asked for four
// words: the template names "no user-visible change" as a valid note. That is
// the escape, rather than a scope that lets an unwritten note through.
//
// Archive-scoped per ADR-0004: an archived milestone is historical state, not
// active drift, and every milestone reaching `done` is swept there eventually.
// The live window this rule governs is the promote itself and the span before
// the sweep. That gate is also why the rule costs nothing to adopt: every
// milestone already at `done` in this repo is archived, so the rule reports on
// none of them.
func milestoneDoneEmptyReleaseNote(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if e.Kind != entity.KindMilestone {
			continue
		}
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		if e.Status != entity.StatusDone {
			continue
		}
		// Read failures silently produce zero findings; the load-error path
		// already covers a file that cannot be read.
		raw, err := os.ReadFile(filepath.Join(t.Root, e.Path))
		if err != nil {
			continue
		}
		_, body, ok := entity.Split(raw)
		if !ok {
			continue
		}
		// Comments are stripped first so a spec carrying only the template's
		// guidance comment reads as the empty section it is.
		sections := entity.ParseBodySections(stripHTMLComments(body))
		// An absent section counts as empty: scoping to present-and-empty would
		// make deleting the heading an escape from the rule.
		if !isAllWhitespaceOrHeadings([]byte(sections[releaseNoteSectionSlug]), false) {
			continue
		}
		findings = append(findings, Finding{
			Code:     CodeMilestoneDoneEmptyReleaseNote,
			Severity: SeverityError,
			Message: fmt.Sprintf("milestone %s is done without a `## Release note` an author wrote; the epic wrap composes its changelog entry from these notes, so this milestone's change reaches the release described by nobody who did the work — write the user-visible delta, or the words \"no user-visible change\" when there is none",
				e.ID),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "release_note",
		})
	}
	return findings
}
