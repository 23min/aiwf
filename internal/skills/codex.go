package skills

import (
	_ "embed"
	"strings"
)

//go:embed embedded-guidance/codex/skill-invocation.md
var codexSkillInvocation string

//go:embed embedded-guidance/codex/review-dispatch.md
var codexReviewDispatch string

//go:embed embedded-guidance/codex/worktree-entry.md
var codexWorktreeEntry string

//go:embed embedded-guidance/codex/worktree-placement.md
var codexWorktreePlacement string

//go:embed embedded-guidance/codex/external-worktree.md
var codexExternalWorktree string

// CodexRenderBindings supplies native instructions without custom role files.
// Entry and placement instructions apply across patch, epic and milestone work.
func CodexRenderBindings() RenderBindings {
	entry := strings.TrimSuffix(codexWorktreeEntry, "\n")
	placement := strings.TrimSuffix(codexWorktreePlacement, "\n")
	external := strings.TrimSuffix(codexExternalWorktree, "\n")
	return RenderBindings{
		Target: CodexTarget(),
		Fragments: HostFragments{
			SkillInvocation:            strings.TrimSuffix(codexSkillInvocation, "\n"),
			ReviewDispatch:             strings.TrimSuffix(codexReviewDispatch, "\n"),
			WorktreeEntry:              entry,
			EpicWorktreeEntry:          entry,
			MilestoneWorktreeEntry:     entry,
			EpicWorktreePlacement:      placement,
			MilestoneWorktreePlacement: placement,
			EpicExternalWorktree:       external,
			MilestoneExternalWorktree:  external,
		},
	}
}

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
