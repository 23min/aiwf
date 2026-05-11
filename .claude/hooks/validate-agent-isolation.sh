#!/usr/bin/env bash
# PreToolUse hook for the Agent tool. Refuses to honor `isolation: "worktree"`
# as a load-bearing isolation mechanism — the kwarg is silently dropped by the
# harness in known failure cases (aiwf gap G-0099) and the work lands in the
# live tree with no detection.
#
# Implements the parent-side precondition pattern from G-0099 by blocking the
# unreliable surface. See CLAUDE.md § "Subagent worktree isolation" for the
# pattern the hook expects callers to use instead.

set -euo pipefail

HOOK_INPUT=$(cat)

if echo "$HOOK_INPUT" | grep -qE '"isolation"[[:space:]]*:[[:space:]]*"worktree"'; then
  cat <<'EOF'
{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"Agent invoked with isolation: \"worktree\". That kwarg is a request to the harness, not a precondition — it has been observed to silently drop, leaving the work in the live tree with no detection (aiwf G-0099). Use the parent-side precondition pattern instead: (1) parent runs `git worktree add <path> -b <branch> <base>`; (2) parent verifies via `git worktree list` that the path appears; (3) parent invokes Agent without the isolation kwarg, naming the worktree path explicitly in the prompt so the subagent operates via absolute paths or `git -C <path>`; (4) on return parent verifies the subagent's commits live on the worktree branch. See CLAUDE.md § \"Subagent worktree isolation\" and aiwf gap G-0099 for the full pattern."}}
EOF
  exit 0
fi

exit 0
