package skills

import (
	_ "embed"
	"strings"
)

//go:embed embedded-guidance/claude/worktree-entry.md
var claudeWorktreeEntry string

//go:embed embedded-guidance/claude/skill-invocation.md
var claudeSkillInvocation string

//go:embed embedded-guidance/claude/review-dispatch.md
var claudeReviewDispatch string

//go:embed embedded-guidance/claude/epic-worktree-entry.md
var claudeEpicWorktreeEntry string

//go:embed embedded-guidance/claude/milestone-worktree-entry.md
var claudeMilestoneWorktreeEntry string

//go:embed embedded-guidance/claude/epic-worktree-placement.md
var claudeEpicWorktreePlacement string

//go:embed embedded-guidance/claude/milestone-worktree-placement.md
var claudeMilestoneWorktreePlacement string

//go:embed embedded-guidance/claude/epic-external-worktree.md
var claudeEpicExternalWorktree string

//go:embed embedded-guidance/claude/milestone-external-worktree.md
var claudeMilestoneExternalWorktree string

// ClaudeRenderBindings supplies Claude's layout and operational fragments.
// Fragment files end in a newline for source editing; inline substitution
// leaves paragraph boundaries to the shared source.
func ClaudeRenderBindings() RenderBindings {
	return RenderBindings{
		Target: ClaudeTarget,
		Fragments: HostFragments{
			WorktreeEntry:              strings.TrimSuffix(claudeWorktreeEntry, "\n"),
			SkillInvocation:            strings.TrimSuffix(claudeSkillInvocation, "\n"),
			ReviewDispatch:             strings.TrimSuffix(claudeReviewDispatch, "\n"),
			EpicWorktreeEntry:          strings.TrimSuffix(claudeEpicWorktreeEntry, "\n"),
			MilestoneWorktreeEntry:     strings.TrimSuffix(claudeMilestoneWorktreeEntry, "\n"),
			EpicWorktreePlacement:      strings.TrimSuffix(claudeEpicWorktreePlacement, "\n"),
			MilestoneWorktreePlacement: strings.TrimSuffix(claudeMilestoneWorktreePlacement, "\n"),
			EpicExternalWorktree:       strings.TrimSuffix(claudeEpicExternalWorktree, "\n"),
			MilestoneExternalWorktree:  strings.TrimSuffix(claudeMilestoneExternalWorktree, "\n"),
		},
	}
}

// renderBindingsForTarget keeps custom layouts usable while selecting native
// instructions for the supported hosts. Other layouts retain Claude semantics.
func renderBindingsForTarget(target Target) RenderBindings {
	bindings := ClaudeRenderBindings()
	if target.Name == "codex" {
		bindings = CodexRenderBindings()
	}
	bindings.Target = target
	return bindings
}
