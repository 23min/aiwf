---
id: G-0619
title: The wrap rituals warn about a harmless failure and miss the real ones
status: addressed
priority: medium
addressed_by_commit:
    - c0a4f13d2d1a2796698a0114dc362d582c890f93
---
## What's missing

Both wrap rituals resolve the worktree holding their merge target into a shell
variable and drive it with `git -C`. Both warn against leaving that variable
empty, and both name a consequence that cannot happen.

`aiwfx-wrap-milestone` says `git -C ""` "would merge the milestone branch into
itself". Measured, standing on the milestone branch and running the ritual's own
command with the variable empty:

```text
$ git -C "" merge --no-ff --no-commit milestone/M-0001-demo
Already up to date.
exit=0            # HEAD unchanged
```

Merging a branch into itself is not a hazard; git declines. `aiwfx-wrap-epic`
says "fast-forward or merge the epic branch into itself", which is half true —
its first use of the variable is `merge --ff-only`, and that one really can move
the branch you are standing on — but the merge half is the same impossible
consequence.

The failure that does occur goes unwarned. The merge is the silent step: the
target receives nothing, at exit 0. What follows is not silent but is easy to
misread: the commit fails at exit 1 behind `nothing to commit, working tree
clean`, which reads like success, and the branch delete fails at exit 1 too,
though that one says `error:` outright. Between them the promote and the
roadmap regen do succeed, landing their commits on the current branch, because
`--root ""` resolves to the current worktree.

A second route to the same silence is unwarned in both: the variable is shell
state and these rituals span many commands, so a new shell, a sequence aborted
and re-gated, or a context compaction loses it.

## Why it matters

A warning whose stated consequence is harmless teaches the reader to discount
it. Anyone who knows git can see that merging a branch into itself does nothing,
and the sentence reads as caution about a non-event — so the instruction it
guards gets less attention, not more, exactly where the real failure mode is
invisible in every command that would otherwise reveal it.

The class also propagates by copying. This text was carried into a third ritual
verbatim on the assumption that the reasoning transferred, where it was wrong in
a worse way: that ritual's first use of the variable is `merge --ff-only`, which
fast-forwards the branch the operator is standing on.

## Prior threads

- [G-0609](archive/G-0609-wrap-rituals-assume-a-single-checkout-breaking-under-the-worktree-default.md)
  introduced the resolved-worktree variable in these two rituals; the warning
  arrived with it.
- [G-0615](archive/G-0615-wf-patch-assumes-a-single-checkout-breaking-under-the-worktree-default.md)
  carried the same text into the third ritual and corrected it there. This gap is
  the same correction applied back to the two it came from.
