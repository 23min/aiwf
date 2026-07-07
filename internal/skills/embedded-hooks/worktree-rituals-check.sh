#!/usr/bin/env bash
# SessionStart / SubagentStart hook: warns when the session starts with
# cwd inside a .claude/worktrees/ checkout whose rituals aren't fully
# materialized. SessionStart/SubagentStart cannot block or abort — a
# nonzero exit renders this script's stderr as a harness notice while
# the session or subagent proceeds regardless, so this is advisory only.

set -euo pipefail

cwd="$(pwd -P)"

case "$cwd" in
  */.claude/worktrees/*) ;;
  *) exit 0 ;;
esac

root="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0

exec aiwf doctor --check-rituals --root "$root"
