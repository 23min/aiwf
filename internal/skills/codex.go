package skills

// CodexTarget returns the local skill layout and aiwf-owned template support
// directory. Claude role cards and hooks have no output in this target.
// Host selection and guidance wiring are separate from artifact placement.
func CodexTarget() Target {
	return Target{
		Name:         "codex",
		SkillsDir:    ".agents/skills",
		TemplatesDir: ".agents/aiwf/templates",
	}
}
