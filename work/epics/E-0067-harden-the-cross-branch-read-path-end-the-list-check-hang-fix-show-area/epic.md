---
id: E-0067
title: 'Harden the cross-branch read path: end the list/check hang, fix show --area'
status: proposed
---

# E-0067 — Harden the cross-branch read path: end the list/check hang, fix show --area

## Goal

Filtered `aiwf list` (~10s) and `aiwf check` (~15–20s) are slow at this repository's
scale because the cross-branch scan runs collision blob-stats over every entity on
every ref, then discards nearly all of it. Make that scan lazy — collision detection
only for ids absent from the local working tree — so read verbs stay fast as the tree
and branch count grow, and fix the one cross-branch correctness bug that lives in the
same code.

## Context

E-0060 shipped the cross-branch read path: `trunk.LocalRefHits` / `trunk.RemoteRefHits`
union every local and remote-tracking ref, then `trunk.DetectCollisions` compares blob
content for every id that appears on two or more refs. Each comparison is a `git
cat-file` round-trip that resolves `<commit>:<path>` — a full tree walk.

At this repository's current shape — 860 entity files on `main`, 10 refs (seven stale
local epic branches plus three remote-tracking), ~8300 ref-hits — the scan issues on
the order of 8300 tree-walk round-trips, and produces zero useful rows: 818 distinct
ids across all refs equals 818 ids in the local tree, so nothing is actually absent.
The round-trips' unit cost depends on environment: with a packed object store and a
fresh commit-graph (`maintenance.auto` now keeps both current), the scan accounts for
~10s of the filtered-list wall clock on this repository's bind-mounted devcontainer
filesystem, and sub-second on native fs. The epic targets the algorithmic term — work
proportional to entities × refs that is then discarded — which grows with both factors
regardless of environment. `DetectCollisions`' result is read only on a local-tree
*miss*, so all the work spent on locally-present ids is discarded.

The cost is not `--priority`-specific; it lands on every filtered `aiwf list` and inside
every `cliutil.LoadTreeWithTrunk` (so `aiwf check` too). It surfaced during E-0066's
priority-backlog triage, which made filtered listing a frequent operation. This epic
builds on E-0052 (the allocator's cross-branch view) and on E-0053 / E-0054 (the prior
fast-check and fast-read-path work).

## Scope

### In scope

- A single lazy cross-branch scan helper in `internal/trunk` that runs `DetectCollisions`
  only for ids absent from the local working tree, collapsing the three duplicated call
  sites into one (also closing G-0418's duplication/coupling concern).
- Rewiring the three consumers to it: `cliutil.LoadTreeWithTrunk` (the check and
  allocation path), `list`'s `crossBranchListRows`, and `show`'s
  `buildCrossBranchShowView`.
- A behavior-preservation proof — cross-branch rows and findings are unchanged before and
  after — plus a scale assertion that collision-stats scale with the locally-absent id
  set, not with entities × refs.
- G-0419: `aiwf show <id> --area X` on a cross-branch-resolved id honors the entity's real
  `area:` field instead of the local-only lookup that always returns untagged.

### Out of scope

- G-0416 (distinguishing an unmerged edit from a genuine duplicate-mint collision via `git
  merge-base`). D-0036's coarse severity stands until practice shows it insufficient; the
  helper introduced here is the seam that makes G-0416 a cheap successor if it does.
- The structural incremental-revwalk cache for `aiwf check`'s full-history walks (scoped
  and cancelled as E-0058, twice-rejected; tracked by G-0372). That is check's *other*,
  history-revwalk cost center; this epic removes only the collision-scan half. Check gets
  faster, but the revwalk half is untouched.
- G-0157's remaining slices (per-worktree and per-scope `git log` fan-out in `status` and
  `show`) — a different surface.
- G-0324 (pruning merged ritual branches) — reduces the scan's input size, not its
  algorithm; independent.

## Constraints

- **Behavior preservation is load-bearing.** The lazy filter skips computing collision
  entries for locally-present ids; this is safe only because every consumer consults a
  collision result exclusively after a local-tree miss (the `refs-resolve` rule's
  cross-branch branch, `body_prose_id`'s cross-branch branch, `crossBranchListRows`, and
  `buildCrossBranchShowView` all guard on a miss first). A test must fail if a
  locally-present id's collision result ever reaches an output.
- The union of hits passed to `DetectCollisions` must stay identical to the union that was
  scanned — one place, not three (G-0418's coupling point).
- `internal/trunk` stays read-only and best-effort (never errors, degrades to nil on odd
  repo state); no new package-level mutable state.
- The allocator's cross-branch view (`LocalRefIDs` / `RemoteRefIDs`, feeding allocation)
  must not regress.

## Success criteria

- [ ] On this repository, a filtered `aiwf list` (any filter) returns in the same
  sub-second order of magnitude as no-args `aiwf list`, not the ~10s it costs today.
- [ ] `aiwf check`'s cross-branch scan no longer exhibits the entities × refs blow-up; a
  scale assertion pins collision-stats to the locally-absent id set.
- [ ] The cross-branch scan composition exists in exactly one place, consumed by all three
  sites.
- [ ] `aiwf show <cross-branch-id> --area X` reports the entity's real area.
- [ ] Cross-branch rows and findings are unchanged for every pre-existing scenario.
- [ ] Every gap named as addressed in the *References* section is closed.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Lazy filter inside `DetectCollisions` (via a `presentLocally` predicate) or in the new scan wrapper (filter hits before calling it)? | no | Milestone design review; lean toward the wrapper, keeping `DetectCollisions` a pure function over the hits it is given |
| Scale proof as an `internal/policies` synthetic-fixture test, or a verb-package scale test? | no | Milestone planning; lean toward a deterministic assertion that `DetectCollisions` receives only absent-id hits, not a wall-clock budget |
| Any consumer of the collision set not guarded by a local-tree miss? | no | Confirmed during investigation — all four guard on a miss first; the behavior-preservation test pins it |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| The lazy filter silently changes a check finding (a real duplicate-mint collision going undetected) | high | Behavior-preservation test on a fixture that includes an absent-locally colliding id; the miss-guard invariant is the correctness basis |
| The ls-tree scan floor (~0.3s) still scales with refs × entities | low | Out of scope here — this epic removes only the collision amplifier; G-0324 and further caching are the next levers if the floor grows |

## Milestones

- `M-0265` — the lazy `trunk` cross-branch scan helper; rewire `LoadTreeWithTrunk`,
  `crossBranchListRows`, and `buildCrossBranchShowView` so `DetectCollisions` runs only for
  locally-absent ids; behavior-preserving and scale-asserted (closes G-0418) · depends on: —
- `M-0266` — `aiwf show --area` on a cross-branch id honors the resolved entity's real area
  (closes G-0419) · depends on: `M-0265`

## ADRs produced

- ADR candidate, to be decided at wrap per the ADR harvest: cross-branch collision
  detection is scoped to the locally-absent id set — the durable design principle
  mirroring the miss-guard invariant.

## References

- Gaps addressed: G-0418, G-0419. Related and deferred: G-0416, G-0372, G-0157, G-0324.
- Epics: E-0060 (origin), E-0052, E-0053, E-0054, E-0058 (cancelled — check-performance
  history).
- Decisions: D-0036 (collision severity), ADR-0030 (cross-branch read-side extension
  point).
- Source: `internal/trunk/trunk.go`, `internal/cli/cliutil/treeload.go`,
  `internal/cli/list/list.go`, `internal/cli/show/show.go`, `internal/gitops/catfile.go`.
