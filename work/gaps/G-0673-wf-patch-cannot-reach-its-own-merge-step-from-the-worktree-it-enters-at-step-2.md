---
id: G-0673
title: wf-patch cannot reach its own merge step from the worktree it enters at step 2
status: open
---
## What's missing

`wf-patch` step 2 instructs the session to call the harness `EnterWorktree(path:
<printed path>)` after `aiwf worktree add --print-path`. Step 11 then instructs
the same session to `cd` into mainline's worktree, and step 12 runs the merge
there. A session that has entered a worktree through `EnterWorktree` can do
neither: the harness refuses `cd` and `git -C` to any path outside the entered
worktree, so the merge the ritual exists to produce is unreachable from the
place the ritual told the session to stand.

The skill carries the exit instruction but places it after the step that needs
it. Step 14's cleanup says to leave a harness-entered worktree with
`ExitWorktree` before removing it — three steps past the merge.

Measured 2026-09-12 in this repo, from a session that had entered
`.claude/worktrees/patch/` through `EnterWorktree`. `git -C` to the main
checkout, `git -C` to a sibling worktree, and compound shell commands were each
refused; non-git reads outside the worktree were allowed. The refusal names the
target "the shared checkout" even where the target is a sibling worktree.

`aiwfx-start-milestone` step 5 and `aiwfx-start-epic` step 8 chain
`EnterWorktree` at the same kind of call site. Whether their wrap paths meet the
same wall is not established here.

## Why it matters

One session cannot complete the ritual. The merge is `wf-patch`'s whole output —
the branch plus an explicit `--no-ff` merge is the audit trail the skill exists
to produce — and the operator has to run it in a second terminal while the
session that did the work waits. That splits a declared-sequence gate across two
actors, which is where an approved sequence stops matching what actually ran.

The conflict is between two correct-looking instructions rather than a mistake
in either. G-0413 established that `aiwf worktree add` does not relocate the
session and that only `EnterWorktree` does, for three reasons that still hold:
the statusline reads the session's own cwd, harness-entered worktrees get a
tracked keep/remove prompt, and cwd-dependent caches refresh. What it did not
weigh is that entering the worktree forecloses the ritual's later steps.

The wall is entirely a consequence of the harness call. Measured 2026-09-12 from
the same session's launch directory — itself a worktree, entered by no tool —
both `cd /workspaces/aiwf` and `git -C /workspaces/aiwf` succeed and report
`main`. Declining step 2 would therefore avoid it, which is exactly what the fix
must not do: G-0413 added that call for three reasons that still hold.

## Resolution shape

The fix is ordering, and it is sufficient. Move the `ExitWorktree` instruction
out of step 14's cleanup and into step 11, where the ritual first needs the
session outside the worktree; step 14 then removes a worktree the session has
already left, which is the order `git worktree remove` wants anyway.

Exiting restores the reach the merge needs. Measured 2026-09-12: a session that
had entered the patch worktree and could reach nothing outside it called
`ExitWorktree` with `keep`, returned to its launch directory, and from there ran
`cd` and `git -C` into the main checkout, both reporting `main`. The worktree and
its branch survived the exit, so the work in flight is not at risk and the
session can re-enter with `EnterWorktree(path: …)` if it needs to.

Step 11 should say which of the two it wants, since they are no longer
equivalent after the exit: `cd` into mainline's worktree, as the step already
reads, or `git -C` from wherever the session landed. The `cd` keeps the plain
commands in steps 12 to 15 working as written.

The same question reaches `aiwfx-start-milestone` and `aiwfx-start-epic`, whose
wrap rituals merge into a parent branch held by another worktree, and whose
start rituals chain the same call.

## Related

- G-0413 — established the `EnterWorktree` chain the three rituals now carry.
  Addressed; this gap is the cost that fix did not weigh.
- G-0669 — the patch during which this surfaced, twice.
