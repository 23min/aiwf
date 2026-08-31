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

// milestoneDoneEmptyReleaseNote fires (warning) when a non-archived milestone at
// `done` has no `## Release note` an author filled in — the section absent, or
// present and carrying nothing but whitespace, headings, or the template's own
// guidance comment.
//
// The section is where a milestone records its own user-visible delta, and the
// epic wrap composes the epic's changelog entry from those notes and copies that
// entry verbatim into the changelog. An empty one is a shipped change that
// reaches a release described by nobody who did the work.
//
// It reports and does not block. `aiwf check` exits non-zero on error severity
// alone, and `promote` gates its projection findings the same way, so this rule
// reaches neither the push nor the `done` transition. That is deliberate: at
// error severity it would demand a section the kernel's own scaffold does not
// write — `entity.RequiredSections` for a milestone is Goal and Acceptance
// criteria — so a milestone created through `aiwf add` could not reach `done`
// at all, and `--force` does not relax a projection finding.
//
// Absence counts, rather than only an empty section that is present. Scoping to
// present-and-empty would make deleting the heading an escape, and it would buy
// nothing: the archive gate below already spares a milestone written before the
// section existed, once the sweep has moved it.
//
// Archive-scoped per ADR-0004: an archived milestone is historical state, not
// active drift, and every milestone reaching `done` is swept there eventually.
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
		if !isAllWhitespaceOrHeadings([]byte(sections[releaseNoteSectionSlug]), true) {
			continue
		}
		findings = append(findings, Finding{
			Code:     CodeMilestoneDoneEmptyReleaseNote,
			Severity: SeverityWarning,
			Message: fmt.Sprintf("milestone %s is done without a `## %s` an author wrote; the epic wrap composes its changelog entry from these notes, so this milestone's change reaches the release described by nobody who did the work — write the user-visible delta, or the words \"no user-visible change\" when there is none",
				e.ID, ReleaseNoteSectionHeading),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "release_note",
		})
	}
	return findings
}
