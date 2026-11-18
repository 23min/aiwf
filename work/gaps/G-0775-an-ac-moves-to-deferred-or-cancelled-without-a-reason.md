---
id: G-0775
title: An AC moves to deferred or cancelled without a --reason
status: open
---
## What's missing

D-0110 requires `--reason` on every move of an AC to `deferred` or `cancelled`, from `open` or
from `met`, through `aiwf promote` or `aiwf cancel`. `promoteAC` and `cancelAC` in
`internal/verb/ac.go` take the reason as optional and commit the move without one.

Measured on Linux 7.0.14 in the devcontainer, with `aiwf` built from a `git archive` of main at
bd818e14d (no Go change since, up to ce4e6d89c), in three copies of a scratch repository: E-0001
`active`, M-0001 `in_progress` with `tdd: none` and AC-1 `open` with body prose, on the epic
branch. E-0001 and M-0001 here are the scratch repository's own; in this tree those ids name
unrelated archived entities.

```
$ aiwf promote M-0001/AC-1 deferred
aiwf promote M-0001/AC-1 open -> deferred
exit 0
$ git log -1 --format=%B
aiwf promote M-0001/AC-1 open -> deferred

aiwf-verb: promote
aiwf-entity: M-0001/AC-1
aiwf-actor: human/peter
aiwf-to: deferred
```

On the second copy:

```
$ aiwf cancel M-0001/AC-1
aiwf cancel M-0001/AC-1 -> cancelled
exit 0
$ git log -1 --format=%B
aiwf cancel M-0001/AC-1 -> cancelled

aiwf-verb: cancel
aiwf-entity: M-0001/AC-1
aiwf-actor: human/peter
```

On the third, after `aiwf promote M-0001/AC-1 met` (exit 0):

```
$ aiwf promote M-0001/AC-1 cancelled
aiwf promote M-0001/AC-1 met -> cancelled
exit 0
$ git log -1 --format=%B
aiwf promote M-0001/AC-1 met -> cancelled

aiwf-verb: promote
aiwf-entity: M-0001/AC-1
aiwf-actor: human/peter
aiwf-to: cancelled
```

## Why it matters

A `deferred` or `cancelled` AC leaves the milestone's contract without claiming it succeeded,
and the commit carries no record of why. `aiwf history` shows that a criterion was deferred or
cancelled, but not why, where deferred work went, or why a criterion recorded as `met` was
withdrawn. Anything else is prose someone chose to write, in the milestone body, where the wrap
ritual asks for a note pointing a deferred AC at the milestone or gap that receives it, or in
the receiving entity, and nothing checks it.
