package policies

import (
	"fmt"
	"maps"
	"regexp"
	"slices"

	"github.com/23min/aiwf/internal/check"
)

// PolicyGuidanceIDsThroughLinks holds this repository's development
// guidance to citing entity ids only through link destinations (M-0336
// AC-2). An id written in the guidance's prose rots as its entity changes
// status or moves to archive, and the reasoning it points at belongs to
// the entity; a link keeps the citation where a reader can follow it. The
// scan is the shipped-surface one, skill-body-id: it masks link
// destinations, passes the canonical letter-N placeholder, and keeps code
// in scope, since a command example is read as prose is.
func PolicyGuidanceIDsThroughLinks(root string) ([]Violation, error) {
	return guidanceIDViolations(developmentGuidance(repoGuidanceReader(root))), nil
}

// flaggedToken is the quoted token in a skill-body-id finding's message.
var flaggedToken = regexp.MustCompile(`"[^"]*"`)

// guidanceIDViolations reports each id-shaped token the shipped-surface
// scan flags in the given documents.
func guidanceIDViolations(docs map[string]string) []Violation {
	var vs []Violation
	for _, file := range slices.Sorted(maps.Keys(docs)) {
		findings := check.ScanSkillBodyID([]byte(docs[file]), file)
		for i := range findings {
			f := &findings[i]
			vs = append(vs, Violation{
				Policy: "guidance-ids-through-links",
				File:   file,
				Line:   f.Line,
				Detail: fmt.Sprintf("%s is an id-shaped token outside a link destination; cite an entity through a link to its document, with the id only in the destination, and write a placeholder in the canonical letter-N form", flaggedToken.FindString(f.Message)),
			})
		}
	}
	return vs
}
