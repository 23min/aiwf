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

// releaseNoteSectionSlug is the section this rule reads, in the slug form
// ParseBodySections keys by.
var releaseNoteSectionSlug = entity.SectionSlug("Release note")

// milestoneDoneEmptyReleaseNote fires (warning) when a non-archived milestone at
// `done` carries a `## Release note` section with nothing an author wrote in it.
//
// The section is where a milestone records its own user-visible delta, and the
// epic wrap composes the epic's changelog entry from those notes and copies that
// entry verbatim into the changelog. A note left empty is therefore a shipped
// change that reaches a release described by nobody who did the work — the
// failure this rule exists to catch, measured on a real release that shipped
// three such changes undocumented.
//
// The rule governs two surfaces from one definition, because `promote` runs the
// projection findings as preconditions: it reports standing state in
// `aiwf check`, and it refuses a `done` promote that would produce that state.
//
// Scoped to a section that is *present* and empty, never to one that is absent.
// A spec written before the section existed carries no such heading, and this
// repo's own tree holds 281 such milestones at `done` — an absent-or-empty rule
// would report every one of them on the day it landed and be switched off rather
// than acted on. A spec scaffolded from the current template carries the
// heading, so present-and-empty selects exactly the milestones the section
// applies to. `entity-body-empty` draws the same line for the same reason.
//
// The residual is that deleting the heading evades the rule. That is the general
// hole G-0571 reports, and the obligation to reconcile this rule with the
// required-sections machinery that closes it is recorded there.
//
// Archive-scoped per ADR-0004: an archived milestone is historical state, not
// active drift. Every done milestone is swept there eventually, so the live
// window this rule governs is the promote itself and the span before the sweep.
//
// Warning rather than error, because a milestone can legitimately have nothing
// user-visible to report — the template asks for those words explicitly ("no
// user-visible change"), and an error would block the wrap before the author has
// the chance to write them.
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
		content, present := sections[releaseNoteSectionSlug]
		if !present {
			continue
		}
		if !isAllWhitespaceOrHeadings([]byte(content), false) {
			continue
		}
		findings = append(findings, Finding{
			Code:     CodeMilestoneDoneEmptyReleaseNote,
			Severity: SeverityWarning,
			Message: fmt.Sprintf("milestone %s is done with an empty `## Release note`; the epic wrap composes its changelog entry from these notes, so this milestone's change reaches the release described by nobody who did the work — write the user-visible delta, or the words \"no user-visible change\" when there is none",
				e.ID),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "release_note",
		})
	}
	return findings
}
