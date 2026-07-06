package skills

import _ "embed"

// WorktreeRitualsCheckScript is the SessionStart/SubagentStart hook
// script (M-0236): it gates on cwd being inside a .claude/worktrees/
// checkout, then delegates the actual materialization answer to
// `aiwf doctor --check-rituals` rather than reimplementing
// MaterializedRituals in shell.
//
//go:embed embedded-hooks/worktree-rituals-check.sh
var WorktreeRitualsCheckScript []byte
