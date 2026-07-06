package skills

import (
	"strings"
	"testing"
)

// TestWorktreeRitualsCheckScript_GatesOnWorktreeConvention pins AC-1's
// cwd-detection claim: the script only acts inside a .claude/worktrees/
// checkout (ADR-0023's default worktree.dir), silently no-op'ing (via an
// early `exit 0`) everywhere else — the main checkout included.
func TestWorktreeRitualsCheckScript_GatesOnWorktreeConvention(t *testing.T) {
	t.Parallel()
	script := string(WorktreeRitualsCheckScript)
	// Scoped to the functional case-pattern token itself, not any
	// occurrence of the substring in the file (the header comment also
	// mentions ".claude/worktrees/" and must not satisfy this alone).
	if !strings.Contains(script, `*/.claude/worktrees/*)`) {
		t.Errorf("hook script's case pattern does not gate on the .claude/worktrees/ convention:\n%s", script)
	}
	if !strings.Contains(script, "exit 0") {
		t.Errorf("hook script has no silent-exit-0 path for the non-worktree case:\n%s", script)
	}
}

// TestWorktreeRitualsCheckScript_DelegatesToCheckRitualsFlag pins AC-1's
// "reusing aiwf doctor's existing rituals-presence check rather than
// reimplementing it" claim: the script's only source of truth for
// materialization state is `aiwf doctor --check-rituals`, never a
// reimplemented skills/agents/templates presence check in shell.
func TestWorktreeRitualsCheckScript_DelegatesToCheckRitualsFlag(t *testing.T) {
	t.Parallel()
	script := string(WorktreeRitualsCheckScript)
	if !strings.Contains(script, "aiwf doctor --check-rituals") {
		t.Errorf("hook script does not delegate to `aiwf doctor --check-rituals`:\n%s", script)
	}
}
