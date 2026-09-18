package skills

import (
	_ "embed"
	"strings"
)

//go:embed embedded-guidance/claude/worktree-entry.md
var claudeWorktreeEntry string

// ClaudeRenderBindings supplies Claude's layout and operational fragments.
// Fragment files end in a newline for source editing; inline substitution
// leaves paragraph boundaries to the shared source.
func ClaudeRenderBindings() RenderBindings {
	return RenderBindings{
		Target: ClaudeTarget,
		Fragments: HostFragments{
			WorktreeEntry: strings.TrimSuffix(claudeWorktreeEntry, "\n"),
		},
	}
}
