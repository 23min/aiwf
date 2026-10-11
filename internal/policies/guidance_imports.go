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
// guidance is delivered as tracked files the router reaches; an import from
// the home directory, an absolute path or a path climbing out of the
// repository makes a session's rules depend on one machine's setup. Both
// files are read whole, managed blocks included, so an updater writing
// such an import is reported too.
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
// the home directory, an absolute one, or one that leaves the repository.
// An import is a filesystem path, so a leading slash is absolute here,
// though the shared resolver reads it as the repository root.
func outsideImports(docs map[string]string) []Violation {
	var vs []Violation
	for _, file := range slices.Sorted(maps.Keys(docs)) {
		for _, target := range markdownImports(docs[file]) {
			_, inRepository := projectguidance.ResolveReference(file, target)
			if inRepository && !strings.HasPrefix(target, "~") && !strings.HasPrefix(target, "/") {
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
