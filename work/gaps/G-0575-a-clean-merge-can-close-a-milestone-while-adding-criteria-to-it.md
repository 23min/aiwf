---
id: G-0575
title: A clean merge can close a milestone while adding criteria to it
status: open
priority: medium
discovered_in: M-0300
---
## What's missing

A clean merge of two individually-legal branches can close a milestone while
adding criteria to it, leaving a `done` milestone with open acceptance criteria.
Reproduced with no hand-edited frontmatter and no `--force`:

```
# on the epic branch: milestone in_progress, AC-1 met
git checkout -b branch-a
aiwf add ac <milestone> --title "..."      # twice; both legal
git checkout <epic-branch> && git checkout -b branch-b
aiwf promote <milestone> done              # legal: its only AC was met
git checkout <epic-branch>
git merge --no-ff branch-a                 # clean
git merge --no-ff branch-b                 # clean
```

`aiwf check` then reports at error severity:

```
milestone-done-incomplete-acs: milestone <milestone> is done but 2 AC(s) still
open: AC-2, AC-3
```

The finding's hint offers two remedies. Both fail, and they fail differently:

| Attempt | Outcome |
|---|---|
| `promote <m>/AC-2 <met\|cancelled>` — the hint's first remedy | refused by the projection guard |
| `cancel <m>/AC-2` | refused by the projection guard |
| `cancel <m>/AC-2 --force --reason ...` | refused; `--force` does not reach the projection guard |
| `promote <m>/AC-2 cancelled --force --reason ...` | refused, likewise |
| `promote <m> done --force --reason ...` — the hint's second remedy | exit 0, `already done; nothing to change` — converges to a NoOp and the error stands |
| `promote <m> in_progress` | refused: `done` is terminal |
| `promote <m> in_progress --force --reason ...` | **the only exit** — force a terminal milestone backwards |

## Why it matters

This is a strictly harder deadlock than the acceptance-criterion trap. There,
one sovereign act on the criterion restores an honest state. Here every act on
the criterion is refused — including the sovereign ones, because `--force`
relaxes the FSM check and never reaches the projection guard — so the only way
out is to force the *milestone* backwards out of a terminal status, which is the
transition the kernel is most explicit about not permitting.

The hint makes it worse rather than better. Its first remedy is refused on the
first criterion an operator tries. Its second reports success while changing
nothing, because a NoOp converges on the milestone's status without noticing the
finding it was invoked to clear. An operator following the error message is told
one move is illegal and another succeeded, and the tree is unchanged either way.

Two merge-reachable traps now share a shape: individually-legal verbs on
separate branches compose, through a conflict-free merge, into a state whose
repair the kernel refuses. Neither is reachable by any single-branch sequence,
which is why no walker has found them — the harness composes on one branch. That
is the composition class G-0121 names, and these are its measured instances.

The projection-guard refusals here have the same root as the criterion trap's:
finding identity includes the message text, and this rule interpolates the list
of still-open criteria, so cancelling one changes the message and the standing
finding reads as newly introduced. That defect is tracked separately; it is the
reason the per-criterion route is closed rather than merely slow.
