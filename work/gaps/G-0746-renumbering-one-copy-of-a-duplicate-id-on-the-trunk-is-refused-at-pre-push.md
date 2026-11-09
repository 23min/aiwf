---
id: G-0746
title: Renumbering one copy of a duplicate id on the trunk is refused at pre-push
status: open
priority: medium
---
## What's missing

When two entities with the same id are on the trunk, repairing the duplicate the way the finding's hint suggests, by renumbering one copy, cannot get past the pre-push hook. The refusal lasts for as long as the trunk ref still holds the duplicate.

`idsUnique` (`internal/check/check.go:344`) checks each disputed trunk id in a loop (line 376). It exempts the id only when `TrunkCollisionRenames` maps the trunk path to the branch's path *for that same id* (line 387). A renumber is in that map: `renameVerbs` (`internal/gitops/refs.go:494`) lists `reallocate`. But its entry maps the renumbered copy's old trunk path to that copy's new path, under the new id. Meanwhile the disputed id's branch-side path belongs to the copy that kept the id, so the exemption never matches. The copy that kept the id is then reported as colliding with the other copy's old path on the trunk.

Reproduced 2026-10-26 with aiwf built from `1e8e17e6c` (it reports `v0.42.0+dirty`), git 2.51.1, on Linux, by `docs/research/id-allocation/experiments/scripts/s0_repair_branch.sh`:

1. Two clones each allocate a gap and an epic from the same base and push their branches.
2. A host merges both into `main` with `git merge --no-ff` and runs no check, so `main` holds two `G-0002` and two `E-0002`. `aiwf check` there reports `7 findings (4 errors, 3 warnings)`.
3. In clones with every hook installed, the script tries the pushes shown below.

The hint's repair on a branch renumbers the second clone's copies:

```
$ aiwf reallocate work/gaps/G-0002-beta-gap.md
aiwf reallocate G-0002 -> G-0004
$ aiwf reallocate work/epics/E-0002-beta-epic/epic.md
aiwf reallocate E-0002 -> E-0003
$ git push origin fix/dup-g0002
work/epics/E-0002-alpha-epic/epic.md:2: error ids-unique/trunk-collision: id "E-0002" is allocated on this branch (work/epics/E-0002-alpha-epic/epic.md) and on trunk (work/epics/E-0002-beta-epic/epic.md) for different entities …
work/gaps/G-0002-alpha-gap.md:2: error ids-unique/trunk-collision: id "G-0002" is allocated on this branch (work/gaps/G-0002-alpha-gap.md) and on trunk (work/gaps/G-0002-beta-gap.md) for different entities …
6 findings (2 errors, 4 warnings)
error: failed to push some refs to '…/host.git'
```

The same two reallocations made on `main` and pushed with hooks are refused with the same two errors (`5 findings (2 errors, 3 warnings)`).

Renumbering *both* copies of each id does pass (`4 findings (0 errors, 4 warnings)`, `* [new branch] fix/both`), but only in one order. Renumbering the first clone's copy first is refused by the verb itself:

```
$ aiwf reallocate work/gaps/G-0002-alpha-gap.md
work/gaps/G-0002-beta-gap.md:2: error ids-unique/trunk-collision: id "G-0002" is allocated on this branch (work/gaps/G-0002-beta-gap.md) and on trunk (work/gaps/G-0002-alpha-gap.md) for different entities …
1 findings (1 errors, 0 warnings)
```

## Why it matters

A duplicate can reach the trunk when a host merges without running `aiwf check` ([G-0536](G-0536-aiwf-check-has-no-ci-backstop-the-primary-oracle-is-per-clone-opt-in.md)). Repairing it afterwards is what `aiwf reallocate` exists for. The repair the hint leads to, renumbering one copy, cannot pass the pre-push check, whether it goes to the trunk or to a branch for a merge request. Only two routes get through:

- renumbering both copies, which also changes the id of an entity that did not need to move and that others may already cite;
- `git push --no-verify`, which skips the whole pre-push check.

Until one of these lands, every branch that contains the trunk's tip carries both copies, and its hooked pushes are refused. In the same run, an unrelated `aiwf add decision` on a branch cut from the duplicated `main` was refused at push with `8 findings (4 errors, 4 warnings)`.
