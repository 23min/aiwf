package skills

import (
	"fmt"
	"path/filepath"
)

// renderProvenance derives README-relative support paths from the same layout
// the artifact writers use. Validate these before materialization changes files.
func renderProvenance(target Target) ([]byte, error) {
	templates, err := filepath.Rel(target.SkillsDir, target.TemplatesDir)
	if err != nil {
		return nil, fmt.Errorf("locating templates %q from skills %q: %w", target.TemplatesDir, target.SkillsDir, err)
	}
	support := "aiwf also materializes entity templates\n(`" + filepath.ToSlash(templates) + "/*.md`), with their own `.aiwf-owned` manifest."
	if target.AgentsDir != "" {
		agents, err := filepath.Rel(target.SkillsDir, target.AgentsDir)
		if err != nil {
			return nil, fmt.Errorf("locating agents %q from skills %q: %w", target.AgentsDir, target.SkillsDir, err)
		}
		support = "aiwf also materializes the role agents (`" + filepath.ToSlash(agents) + "/*.md`) and entity templates\n(`" + filepath.ToSlash(templates) + "/*.md`), each with its own `.aiwf-owned` manifest."
	}
	return []byte(provenanceReadmeIntro + support + provenanceReadmeOutro), nil
}
