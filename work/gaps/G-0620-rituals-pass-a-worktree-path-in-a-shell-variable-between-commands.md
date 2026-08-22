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
and in an operator's terminal alike. Measured the same way: a `cd` issued in one
call is still in effect in the next.

That survival has a bound, and the bound is a second route to the same failure.
Measured the same way: a `cd` to a path outside the repository is undone — the
harness restores the repository root and reports it at the tail of that command's
output, where a reader waiting on the next command's result does not look. Every
relative path issued afterwards resolves against the main checkout. Observed cost
on 2026-08-22: three consecutive file edits intended for a worktree were written
to the main checkout, undetected until an unrelated notice named the wrong path.

A guard conditioned on suspicion does not close either route. The operator is not
uncertain about the working directory; they are unaware it changed. What closes it
is asserting the location in the same command as the action, and aborting on
mismatch rather than reporting one.

## Why it matters

Only the first step is silent; the rest fail in ways that read like success. The
merge reports `Already up to date.` at exit 0 and the target receives nothing.
The commit after it fails at exit 1 behind `nothing to commit, working tree
clean`. Between them the promote and the roadmap regen succeed, landing their
commits on whichever branch the session stands on, since an empty `--root`
resolves to the current worktree too. The branch delete then fails at exit 1,
`not fully merged`.

No work is lost — `-d` refuses, and the commits survive on the branch. What is
lost is the correspondence between the planning tree and the code: the entity is
marked addressed or done, and that status-flip commit sits on a branch that never
merged. A reader of the tree sees closed work that did not land.

The warnings do not close this, even now that they describe the failure
accurately (G-0619). They present an empty variable as the exception to guard
against, and for an assistant consumer it is not an exception — it is every step
after the one that sets it.

The pattern reached three rituals by being copied as the fix for a different
defect: G-0609 and G-0615 corrected rituals that told the operator to check out
the merge target, which fails under the worktree convention. That correction was
sound about what not to do and untested about what it replaced it with. It also
carried a conflation. Checking out the target moves a branch into a worktree and
fails when another holds it; changing directory into the worktree that already
holds it moves nothing and cannot fail that way. Only the first was ever the
defect, and both were avoided.

One ritual carries a second defect from that same correction. Where the target is
checked out nowhere, all three fall back to checking it out in the current worktree.
That is right in two of them, whose later steps need no branch of their own. In
`aiwfx-wrap-epic` it is not: the fallback sits at step 5, and steps 6 through 8
commit on the epic branch, so taking it moves the session off the branch those
commits belong to. Measured on 2026-08-22 against a scratch repository with the
target held by no worktree — the fallback succeeded, the step 6 assertion reported
"not on the epic branch" and continued rather than stopping, and the wrap commit
landed on the target, which then carried a wrap for work it did not contain while
the epic branch held that work without its wrap.
