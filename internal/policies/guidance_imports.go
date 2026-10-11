package policies

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/23min/aiwf/internal/projectguidance"
)

// PolicyGuidanceImportsInRepository holds both host entry points to
// importing nothing from outside the repository (M-0335 AC-1). Project
// guidance is delivered as tracked files under .guidance/ and reached
// through the router; an import from the home directory or an absolute path
// makes a session's rules depend on one machine's setup, which is what
// that delivery replaced. Managed blocks are read too, since an updater
// writing such an import back is the same regression.
func PolicyGuidanceImportsInRepository(root string) ([]Violation, error) {
	read := repoGuidanceReader(root)
	docs := map[string]string{}
	for _, entry := range []string{projectguidance.ClaudeFile, projectguidance.AgentsFile} {
		// A missing entry point imports nothing; the ceiling policy reports it.
		docs[entry], _ = read(entry)
	}
	return outsideImports(docs), nil
}

// outsideImports reports each import, in each document, of a path under
// the home directory or an absolute one.
func outsideImports(docs map[string]string) []Violation {
	var vs []Violation
	for _, file := range slices.Sorted(maps.Keys(docs)) {
		for _, target := range markdownImports(docs[file]) {
			if !strings.HasPrefix(target, "~") && !strings.HasPrefix(target, "/") {
				continue
			}
			vs = append(vs, Violation{
				Policy: "guidance-imports-in-repository",
				File:   file,
				Detail: fmt.Sprintf("@%s is an import from outside the repository; deliver the guidance as a tracked file the project router links to instead.", target),
			})
		}
	}
	return vs
}
