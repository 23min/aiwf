---
id: G-0413
title: aiwf worktree add never chains EnterWorktree into the new worktree
status: addressed
addressed_by_commit:
    - f89f1eed
---
## What's missing

`aiwf worktree add` creates the git worktree and materializes aiwf's rituals into it, but a Claude Code session that calls it never moves there — the harness session's own cwd is separate state, relocated only by the harness-level `EnterWorktree` tool call. None of the three ritual skills that shell out to `aiwf worktree add` (`aiwfx-start-milestone`, `aiwfx-start-epic`, `wf-patch`) invoke it. Each skill's worktree-creation step ends with the `aiwf worktree add <branch> --base <base>` command and nothing else; the operator (or agent) is left to `cd` there by hand, which changes a subprocess's working directory but not the harness session's.

## Why it matters

Three concrete losses follow from the session cwd never actually moving. The statusline hook reads the harness session's own cwd — only `EnterWorktree` relocates it — so a worktree entered via plain `cd` never surfaces there, hiding which branch or entity is actually in flight. `EnterWorktree`-entered worktrees get a tracked keep/remove prompt at session end; a worktree entered by plain `cd` is invisible to that mechanism, so cleanup depends entirely on wrap rituals remembering to `git worktree remove` it by hand. And CWD-dependent caches (memory files, the plans directory) don't refresh to the new location, since they key off the harness's notion of where the session is, not a subprocess's `cd` target.

## Resolution shape

`aiwf worktree add` already ships `--print-path` (`internal/cli/worktree/worktree.go`), purpose-built for shell composition (`cd "$(aiwf worktree add ... --print-path)"`) — no CLI change is needed. The fix is a skill-layer edit: in `aiwfx-start-milestone`, `aiwfx-start-epic`, and `wf-patch`, at the point where the skill's own direct-work session is about to operate in the new worktree, capture the path via `--print-path` and then call the harness `EnterWorktree(path: <path>)` tool as an explicit second step — CLI creation stays a subprocess concern, harness session relocation becomes a second, separate tool call made by the skill's caller.

Out of scope: CLAUDE.md's "Subagent worktree isolation" section, where the parent creates the worktree but a *different* `Agent`-tool invocation does the work rather than the harness session itself moving — `EnterWorktree` has no meaning there, since an `Agent` dispatch already receives an explicit path in its prompt rather than inheriting a session cwd.

This gap closes when all three ritual skills chain `EnterWorktree` after `aiwf worktree add --print-path` at their direct-work call sites, and each skill edit carries its `internal/policies/` structural-test companion per the `skill-edit-structural-test-backstop` chokepoint.
