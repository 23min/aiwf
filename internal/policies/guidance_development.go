package policies

import "github.com/23min/aiwf/internal/projectguidance"

// developmentGuidance returns the handwritten text of each development
// guidance document present in the tree: the two host instruction files
// with aiwf's managed blocks removed, the project router, and the
// documents it links to except the outputs aiwf owns, which are generated.
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
	record, _ := read(projectguidance.OwnedRecordFile)
	for _, p := range projectguidance.HandwrittenDocuments(router, projectguidance.OwnedOutputs(record)) {
		if content, ok := read(p); ok {
			out[p] = content
		}
	}
	return out
}
