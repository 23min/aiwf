---
id: G-0273
title: No fetch-before-allocate option to refresh trunk view before id add
status: open
---
## Problem

`aiwf add` allocates against `refs/remotes/origin/main` — the *local*
remote-tracking ref, which is only as fresh as the last `git fetch`
(`internal/trunk/trunk.go`, `config.DefaultAllocateTrunk`). A session that has
not fetched recently allocates against a stale trunk view: ids that landed on
the real trunk since the last fetch are invisible, so the allocator can hand back
an id that already exists upstream. The collision surfaces only at push time via
the `ids-unique` check.

This is the second of the three id-collision classes catalogued in `G-0272`
(same machine, sequential work, trunk-tracking ref stale). It is distinct from
class 1 (sibling worktree heads, fixed by `G-0272`) and class 3 (genuinely
concurrent cross-machine, unpreventable locally and cured by `G-0274`).

## Direction (to converge at the milestone)

- An opt-in refresh of the trunk-tracking ref immediately before allocation —
  e.g. `aiwf add --fetch`, or a narrower fetch of just the configured trunk ref —
  so `max` is computed against the freshest published trunk.
- Network-touching and best-effort: a fetch failure (offline, no remote) must
  degrade to today's behaviour, not block the add. The fetch *narrows* the
  window; it does not close it (another machine can still publish between the
  fetch and the commit — that residual is `G-0274`'s to cure cheaply).
- Consider whether a `doctor` nudge ("trunk-tracking ref is N commits / H hours
  behind origin") is a lower-friction surface than a per-`add` flag, or a
  complement to it.

## Why opt-in, not automatic

An implicit fetch on every `aiwf add` would put a network round-trip on the
critical path of a verb that is otherwise purely local and fast, and would fail
noisily offline. The cost/benefit only favours the fetch when the operator
suspects trunk has moved — hence a flag (or a nudge), not a default.

## Provenance

Emerged from a design discussion on reducing merge-time id-collision risk
(2026-06-22). Secondary gap of a three-gap set headed by `G-0272`; sibling of
`G-0274` (batch reallocate). Narrows collision class 2 from that catalogue.
