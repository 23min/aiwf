---
id: G-0673
title: wf-patch cannot reach its own merge step from the worktree it enters at step 2
status: open
---
## What's missing

`wf-patch` step 2 instructs the session to call Claude Code's
`EnterWorktree(path: <printed path>)` after `aiwf worktree add --print-path`, and
`aiwfx-start-milestone` step 5 and `aiwfx-start-epic` step 8 carry the same
instruction. A session entered that way is confined to the worktree: Claude Code
refuses `cd` and `git -C` into any other checkout. `wf-patch` step 11 then tells
the same session to `cd` into mainline's worktree, and step 12 merges there, so
the merge the ritual exists to produce is unreachable from where step 2 put the
session. The only instruction to leave, `ExitWorktree`, sits in step 14's
cleanup, after the merge. `aiwfx-wrap-milestone` and `aiwfx-wrap-epic` also merge
from the worktree holding their target branch; neither path was run end to end.

Measured 2026-09-12 in this repo, from a session that had entered
`.claude/worktrees/patch/` through `EnterWorktree`: `git -C` to the main
checkout, `git -C` to a sibling worktree, and compound shell commands were each
refused, while non-git reads outside the worktree were allowed. The refusal
names the target "the shared checkout" even where the target is a sibling
worktree. From a session whose working directory was a worktree it had not
entered through the tool, `cd /workspaces/aiwf` and `git -C /workspaces/aiwf`
both succeeded and reported `main`.

## Why it matters

One session cannot complete the ritual. The merge is `wf-patch`'s whole output —
the branch plus an explicit `--no-ff` merge is the audit trail the skill exists
to produce — and the operator has to run it in a second terminal while the
session that did the work waits. That splits a declared-sequence gate across two
actors, which is where an approved sequence stops matching what actually ran.
The same confinement also refuses git commands in the review worktrees the
session hands to its reviewer subagents, and ordinary loops and pipelines inside
the entered worktree — both counted on 2026-09-16 from this repo's Claude Code
session transcripts.
