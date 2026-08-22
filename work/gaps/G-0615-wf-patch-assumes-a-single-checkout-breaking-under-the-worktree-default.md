---
id: G-0615
title: wf-patch assumes a single checkout, breaking under the worktree default
status: addressed
priority: medium
addressed_by_commit:
    - 5ef655b1e758aa809fe770aafb68a8d3ce9faf77
---
## What's missing

`wf-patch` ends by telling the operator to check out mainline. Step 11 does it to
fast-forward it, step 12 to run the merge:

```bash
git checkout main
git merge --no-ff --no-commit <branch>
```

Git allows one checkout per branch. Under this repo's worktree default the patch
branch lives in its own worktree and mainline is already checked out in the
primary one, so the command is refused and the documented step cannot be
followed as written.

[G-0609](G-0609-wrap-rituals-assume-a-single-checkout-breaking-under-the-worktree-default.md)
found and fixed the same assumption in `aiwfx-wrap-epic` and
`aiwfx-wrap-milestone`. `wf-patch` carries it too and was not part of that
change, so the class survives in the third ritual that merges to mainline.

## Why it matters

The refusal lands at the end of the ritual — after the work, after the commit
gate, partway through a sequence the human has already approved. Whoever hits it
is improvising while finishing, and the readiest improvisation is to close the
worktree and switch the primary checkout in place, which is what the worktree
default exists to avoid. An instruction that cannot be followed teaches the
operator to stop trusting the surrounding steps, and the steps around this one
are the gates.

`wf-patch` also materializes into consumer repos, where the same default applies
and the same step fails.

## Resolution shape

Point git at the mainline checkout rather than moving the session into it —
`git -C <mainline-worktree> merge --no-ff --no-commit <branch>`, and the same
for step 11's fast-forward. This is the shape the wrap rituals now use, so the
fix is an application of a resolved decision rather than a new one.

Two details worth keeping when the text is rewritten:

- The merge target is resolved rather than assumed. A ritual cannot hardcode the
  primary checkout's path; `git worktree list` names which worktree holds the
  target branch.
- The ordering constraint stays as it is. This changes where the merge runs, not
  when it runs relative to the promote.

## Prior threads

- [G-0609](G-0609-wrap-rituals-assume-a-single-checkout-breaking-under-the-worktree-default.md)
  is the same defect in the two wrap rituals, already fixed. Its resolution is
  the template for this one; what it did not do was sweep the remaining ritual.
