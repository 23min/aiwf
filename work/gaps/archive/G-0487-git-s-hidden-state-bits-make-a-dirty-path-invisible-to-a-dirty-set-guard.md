---
id: G-0487
title: Git's hidden-state bits make a dirty path invisible to a dirty-set guard
status: addressed
priority: medium
discovered_in: M-0282
addressed_by_commit:
    - 9c6ebe566
---
## What's missing

Git can be told to stop reporting a path, and when it has been, every
working-tree query aiwf makes about that path answers "clean" while the file on
disk says otherwise. A verb then commits the disk content as its own.

Measured, with `git update-index --assume-unchanged` set on a nested milestone,
its `tdd:` field hand-edited from `none` to `required`, and `aiwf rename` run on
the parent epic:

- `git status --porcelain` — nothing for that path
- `git diff --name-only HEAD` — nothing
- `git ls-files --others --exclude-standard` — nothing
- `git ls-files -v` — `h`, the only primitive that shows it

The rename commit carried `tdd: required`, and `aiwf history` on the milestone
shows only its creation. That is the laundering vector E-0075 calls its worst —
a policy field that decides whether `acs-tdd-audit` fires, changed under another
entity's trailer with no event of its own.

`skip-worktree` behaves the same way and has a second shape: under a sparse
checkout the file is absent from disk while present in HEAD and the index, and
git still calls the tree clean.

## Why it matters

The precondition E-0075 is building derives its dirty set from
`gitops.DirtyPaths`, which is `git diff --name-only HEAD` unioned with
`git ls-files --others --exclude-standard`. Neither half sees a path carrying
`assume-unchanged` or `skip-worktree`. So the guard has a blind spot that is
exactly the vector it exists to close, and nothing in its design would surface
that.

This is not an exotic configuration. `skip-worktree` is how sparse checkouts
work, and sparse checkouts are ordinary in large repositories. `assume-unchanged`
is a documented performance workaround operators reach for on slow filesystems.

The honest framing is that this bounds the guard rather than defeating it: a
precondition over the working tree can only be as truthful as the working tree's
own reporting. What it must not do is claim a completeness it cannot have.

## Scope

Two things, and they are separable:

- What `aiwf` should do when a path it is about to commit carries one of these
  bits — detect and refuse, detect and warn, or document the limit.
- The sparse-checkout variant, where a directory move leaves behind the entries
  git is not materializing. That is a correctness question about the move itself,
  not about a guard.

Out of scope: the guard's design as a whole, which is E-0075's.

## Resolution options

1. **Detect the bits and refuse.** `git ls-files -v` over the paths a plan
   touches, refusing on `h`, `S`, or the lowercase variants. One extra
   subprocess, scoped to the plan's paths. Closes the laundering route and makes
   the sparse case explicit rather than silent.
2. **Document the limit and do nothing.** Cheapest, and defensible for
   `assume-unchanged`, which git's own documentation describes as a promise the
   operator makes and breaks at their own risk. Not defensible for sparse
   checkout, where the operator promised nothing.
3. **Refuse to run directory moves under a sparse checkout at all.** Narrow and
   blunt, but the split-directory outcome is a corrupted record rather than a
   missed guard, so a refusal is proportionate there.

Option 1 for the detection, with option 3's stance for the sparse case, is the
lean. Whatever is chosen, E-0075's precondition should state this limit rather
than imply a completeness it does not have.
