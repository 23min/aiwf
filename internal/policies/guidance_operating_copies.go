package policies

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/23min/aiwf/internal/projectguidance"
)

// PolicyGuidanceNoOperatingCopies holds this repository's development
// guidance free of handwritten copies of the shipped operating rules
// (M-0336 AC-1). An operating rule's home is the shipped guidance, which
// this repository reads through the same managed blocks a consumer does; a
// second, handwritten statement of it forks. A paragraph or list item
// carrying every fragment of one of m0211OperatingAnchors restates that
// rule. The managed blocks are generated from the shipped source and are
// not copies, so they are left out. The anchors cover only the rules they
// name: semantic duplication of anything else is held at review.
func PolicyGuidanceNoOperatingCopies(root string) ([]Violation, error) {
	return operatingCopyViolations(developmentGuidance(repoGuidanceReader(root))), nil
}

// developmentGuidance returns the handwritten text of each development
// guidance document present in the tree: the two host instruction files
// with aiwf's managed blocks removed, the project router, and the
// documents it links to.
func developmentGuidance(read func(string) (string, bool)) map[string]string {
	out := map[string]string{}
	for _, p := range []string{projectguidance.ClaudeFile, projectguidance.AgentsFile} {
		if content, ok := read(p); ok {
			out[p] = projectguidance.HandwrittenText(content)
		}
	}
	router, ok := read(projectguidance.RouterFile)
	if !ok {
		return out
	}
	out[projectguidance.RouterFile] = router
	for _, p := range projectguidance.RoutedDocuments(router) {
		if content, ok := read(p); ok {
			out[p] = content
		}
	}
	return out
}

// listItemStart opens a list item, which is a paragraph of its own here.
var listItemStart = regexp.MustCompile(`^(?:[-*+]|\d+\.) `)

// guidanceParagraphs splits a document at blank lines and at each list
// item that starts at the margin.
func guidanceParagraphs(doc string) []string {
	var out []string
	var current []string
	flush := func() {
		if len(current) > 0 {
			out = append(out, strings.Join(current, "\n"))
			current = nil
		}
	}
	for _, line := range strings.Split(doc, "\n") {
		if strings.TrimSpace(line) == "" || listItemStart.MatchString(line) {
			flush()
		}
		if strings.TrimSpace(line) != "" {
			current = append(current, line)
		}
	}
	flush()
	return out
}

// operatingCopyViolations reports each paragraph of the given documents
// that carries every fragment of an operating anchor.
func operatingCopyViolations(docs map[string]string) []Violation {
	var vs []Violation
	for _, file := range slices.Sorted(maps.Keys(docs)) {
		for _, para := range guidanceParagraphs(docs[file]) {
			lower := strings.ToLower(para)
			for _, a := range m0211OperatingAnchors {
				if a.present(lower) {
					vs = append(vs, Violation{
						Policy: "guidance-no-operating-copies",
						File:   file,
						Detail: fmt.Sprintf("restates the shipped operating rule %q (it carries %q in one paragraph); the shipped guidance is that rule's home, so delete this copy and record it with a `copy of` disposition", a.name, a.fragments),
					})
				}
			}
		}
	}
	return vs
}
