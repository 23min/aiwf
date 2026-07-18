---
id: ADR-0035
title: Cross-branch collision detection is scoped to the locally-absent id set
status: accepted
---
# ADR-0035 — Cross-branch collision detection is scoped to the locally-absent id set

> **Date:** 2026-07-18 · **Decided by:** human/peter

## Context

ADR-0030 extended the cross-branch view (ADR-0025) to two read-only consumers:
`aiwf check`'s reference resolution (`refs-resolve`, `body-prose-id`) and the
`aiwf show` / `aiwf list` read path. Each composes the union of every id's hits
across the local and remote-tracking refs, then runs a collision pass — one `git
cat-file` blob-stat per id that appears on two or more refs — to tell a genuine
cross-branch divergence (the same id minted independently on two branches, content
in conflict) from one coherent entity merely known on several refs.

That collision pass ran eagerly over the whole ref union, duplicated at three call
sites. At a real repository's scale — on the order of hundreds of entities across
roughly ten refs, thousands of hits — it cost the corresponding number of blob-stat
round-trips on every filtered `aiwf list` and inside every `cliutil.LoadTreeWithTrunk`
(hence every `aiwf check`), and produced nothing: a collision result is read only
after the local working tree fails to resolve the id. When every id is present
locally — the common, all-merged state — the entire pass is computed and discarded.

The controlling fact is a property of the consumers, not of the scan. Every surface
that reads a collision result guards on a local-tree miss first: `refs-resolve` and
`body-prose-id` consult the cross-branch view only when the id misses the loaded
tree; `list` and `show` render a cross-branch result only for an id absent locally.
A collision entry for a locally-present id can never reach an output.

## Decision

Cross-branch collision detection is scoped to the locally-absent id set. A single
`internal/trunk` helper composes the local + remote ref-hit union once and hands the
collision pass only the hits whose canonical id is absent from the local working
tree; locally-present ids are never blob-stat'd. All three consumers route through
that one helper, so the "hits handed to collision detection equal the union that was
scanned" coupling has one home rather than three.

This is behavior-preserving by the **miss-guard subset invariant**: the set of ids
the filter treats as present locally is a subset of every consumer's own local index,
so an id whose collision the filter skips is never an id a consumer would read a
collision for. Cross-branch rows and check findings are identical before and after,
while collision-stat cost drops from O(entities × refs) to O(locally-absent ids) —
zero work, and sub-second wall clock, in the common all-merged state.

The invariant is load-bearing and imposes a standing obligation: **any new consumer
of the cross-branch collision set must read a collision result only after a
local-tree miss.** A consumer that reads a collision for a locally-present id would
observe an empty result where the eager scan once produced one — a silent
correctness regression that the lazy scoping introduces precisely because that path
is assumed unreachable. Extending collision consumption to a surface that is not
miss-guarded is a new decision, not a quiet change.

Distinguishing a genuine duplicate-mint collision from an unmerged edit of the same
entity (G-0416) is out of scope and unaffected; this helper is the seam that makes
it a cheap successor. The coarse, non-blocking collision severity of D-0036 is
unchanged.

## Consequences

**Positive:**

- Filtered `aiwf list` and the cross-branch half of `aiwf check` shed the
  O(entities × refs) collision-scan cost; work now tracks the locally-absent id set,
  which is empty whenever the tree is fully merged.
- One composition point instead of three: the union/collision coupling is no longer
  copied across `cliutil.LoadTreeWithTrunk`, `list`, and `show`.

**Negative:**

- The miss-guard subset invariant is now a correctness precondition, not merely a
  performance nicety. A future cross-branch collision consumer that reads a collision
  without a local-tree miss first silently gets an empty result instead of the real
  one; every new consumer must honor the guard, and that is the maintenance cost of
  the lazy scoping.
- `aiwf show`'s cross-branch path reads its single id's collision status from the
  whole locally-absent set rather than scanning that id alone. The result is
  identical (that path is reached only on a local-tree miss) and bounded by the
  absent set — accepted in exchange for the single composition point.

## Validation

The decision holds as long as (1) a behavior-preservation test keeps cross-branch
rows and findings identical whether or not a locally-present id carries a recorded
collision, and (2) no cross-branch collision consumer reads a collision result
without a local-tree miss first. If a consumer ever needs an un-guarded collision
read, that is a new decision, not a quiet extension of this one.

## References

- Related ADRs: `ADR-0025` (the cross-branch view this scopes), `ADR-0030` (the
  read-side consumers whose miss-guard makes the scoping safe).
- aiwf decisions: `D-0036` (collision severity is non-blocking).
- Epic: `E-0067`. Milestone: `M-0265`. Follow-up: `G-0416`.
