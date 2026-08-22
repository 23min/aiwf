---
id: G-0620
title: Rituals pass a worktree path in a shell variable between commands
status: open
priority: high
---
## What's missing

Three rituals resolve the worktree holding their merge target into a shell
variable in one step and read it back in later steps. `wf-patch` sets `MAIN_WT`
at step 11 and drives steps 12 through 15 from it; `aiwfx-wrap-epic` and
`aiwfx-wrap-milestone` do the same with their own variables. Each warns against
leaving the variable empty, and each lists "a new shell" among the ways that
happens.

For an assistant running these rituals, every command is a new shell. Measured in
Claude Code on 2026-08-22, one Bash tool call per line:

```text
$ MAIN_WT=$(git worktree list --porcelain | awk ...)   # call 1
  [/workspaces/aiwf]
$ echo "[$MAIN_WT]"                                    # call 2
  []
$ git -C "" rev-parse --abbrev-ref HEAD
  main
```

The variable is gone by the next call, and `git -C ""` does not fail — it runs
against the current worktree. So the documented path resolves the target
correctly, discards the answer, and then operates on whichever branch the session
is standing on. The empty case the rituals treat as an exception is the only case
this consumer reaches.

Working directory is the state that does survive between calls, in this harness
and in an operator's terminal alike. It is the mechanism the rituals were written
to avoid.

## Why it matters

The failure is silent at both commands that could catch it. The merge lands
nowhere and reports exit 0; the commit that follows reports `nothing to commit,
working tree clean` and also exits 0. The ritual then promotes the entity,
regenerates the roadmap, and deletes the branch — each step resting on a merge
that did not happen, and the branch delete is where the work is lost.

The warnings do not close this. They name the consequence as merging a branch
into itself, which git declines, so a reader who checks the claim finds a
non-event and learns to discount the instruction guarding it (G-0619).

The pattern reached three rituals by being copied as the fix for a different
defect: G-0609 and G-0615 corrected rituals that told the operator to check out
the merge target, which fails under the worktree convention. The correction was
sound about what not to do and untested about what it replaced it with.
