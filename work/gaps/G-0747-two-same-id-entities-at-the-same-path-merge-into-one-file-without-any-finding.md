---
id: G-0747
title: Two same-id entities at the same path merge into one file without any finding
status: open
---
## What's missing

Two branches that allocate the same id to entities with the same title produce the same path, because a file's name is its id followed by a slug of the title. aiwf reports the collision nowhere:

- **On the branch.** `DisputedTrunkIDs` (`internal/check/check.go:288`) drops a trunk id whose path matches the branch's file (`if branchPath == tid.Path {`, line 317), whatever either file holds. The pre-push therefore passes.
- **On the trunk.** When the contents match, git merges the two additions into one file without a conflict. The trunk then holds one file standing for two entities, and `aiwf check` there finds nothing.

Reproduced 2026-10-26 with aiwf built from `1e8e17e6c` (it reports `v0.42.0+dirty`), git 2.51.1, on Linux, by `docs/research/id-allocation/experiments/scripts/s0_samepath.sh`:

1. Two clones with hooks each add a gap titled "Login fails" with the same body, and each cites its own gap.
2. The first clone's branch merges to `main`.
3. The second clone fetches and pushes its branch, and a host merges it.

```
alpha: G-0002
beta:  G-0002
== beta's pre-push after A has landed on main and beta fetched
2 findings (0 errors, 2 warnings)
blob A: dd8962ecc0253ba88e0423b7867a961d93945fd9  blob B: dd8962ecc0253ba88e0423b7867a961d93945fd9
== host merges feature/B
Merge made by the 'ort' strategy.
gaps on main: G-0001-seed-gap.md G-0002-login-fails.md
1 findings (0 errors, 1 warnings)
$ aiwf history G-0002
… human/alpha  add  aiwf add gap G-0002 "Login fails"  160892d
… human/beta   add  aiwf add gap G-0002 "Login fails"  b427681
```

When the bodies differ, the second clone's pre-push still passes, and the host merge stops at `CONFLICT (add/add): Merge conflict in work/gaps/G-0002-login-fails.md`.

Two runs of `aiwf add epic --title "Q3 planning"` with no `--body` behave the same way, because the scaffold body matches too (`s0_samepath_epic.sh`). Blob `e0cbe579b884…` is on both branches, and in a fresh clone of `main` after both merges:

```
epics: E-0001-seed-epic E-0002-q3-planning
5 findings (0 errors, 5 warnings)
$ aiwf history E-0002
… human/alpha  add  aiwf add epic E-0002 "Q3 planning"  927190f
… human/beta   add  aiwf add epic E-0002 "Q3 planning"  c12f4c4
```

## Why it matters

Two entities that two people or agents created and cited separately silently become one. Both sides' citations name the same entity, and because there is one file, every later edit, status change and acceptance criterion from either side changes that one entity. Neither author is told. The only trace is two `add` events for one id in `aiwf history`.

When the contents differ, the collision first appears as a git add/add conflict on the host, not as an aiwf finding. Whoever resolves it is invited to merge two entities' text into one file instead of renumbering one of them.

## Relates to

[G-0416](G-0416-cross-branch-collision-can-t-tell-an-edit-from-a-genuine-collision.md) and [D-0036](../decisions/D-0036-cross-branch-collision-severity-is-non-blocking-not-error.md) keep the cross-branch-collision finding non-blocking. They rely on a genuine collision being "still caught … just later" by `ids-unique`/`trunk-collision`. A collision at the same path is not caught by either.
