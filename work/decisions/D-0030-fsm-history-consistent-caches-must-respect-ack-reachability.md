---
id: D-0030
title: fsm-history-consistent caches must respect ack reachability
status: proposed
relates_to:
    - G-0372
    - E-0058
---
> **Date:** 2026-07-05 · **Decided by:** human/peter

## Question

Can `fsm-history-consistent`'s per-commit findings be memoized or cached
across `aiwf check` invocations — and shared across worktrees — the way two
attempted designs for G-0372 assumed, or does something about the rule's
actual semantics make that unsafe?

## Decision

Any future design that memoizes or caches `fsm-history-consistent` (or
another rule that consults `WalkAcknowledgedSHAs` / `computeAckedObservations`
-style HEAD-relative exemption data) results — across checks, or shared
across worktrees — must make the memo's validity a function of the
reachable acknowledgment set, not just the commit or ref range walked. A
design that assumes a commit's finding-verdict is a pure function of its
own content, ignoring which acknowledgments are reachable from the current
HEAD, is not adopted.

## Reasoning

`fsm-history-consistent`'s illegal-transition / manual-edit /
forced-untrailered predicates exempt a commit when it's covered by an
`aiwf acknowledge illegal` (or audit-only) commit — and that exemption set
(`ackedSHAs` / `ackedObs`) is computed by walking `HEAD`-reachable history
only, not `--all`. This is deliberate, DAG-aware scoping: a cherry-picked
acknowledgment landing on a branch that never saw the original offense must
not suppress the finding there. The consequence is that a commit's
finding-verdict is not a pure function of its own content — it also depends
on which acknowledgment commits happen to be reachable from wherever the
check is run.

Two designs assumed otherwise. The exact-per-commit-cache epic (E-0058,
scoped from `docs/initiatives/check-performance-incremental-revwalk-cache.md`,
cancelled after a four-way adversarial review) and a simpler ref-tip-watermark
alternative (independently reviewed three ways, also set aside) both treated
"this commit range has been verified clean" as a durable, shareable fact.
Both broke on the same concrete counterexample: worktree A acknowledges an
illegal commit X, its check goes clean, and a shared watermark/cache
advances past X; worktree B, on a different branch that never merged A's
acknowledgment, later reuses that memoized "clean" state — but a full walk
from B's HEAD would still find X, since B's HEAD doesn't reach the
acknowledgment that exempted it there. No history rewrite is required to
reproduce this.

Alternatives considered and rejected:

- **Ignore the interaction and ship anyway.** Rejected — it produces a
  silent false negative on the exact rule that guarantees entity-status
  history integrity, the one class of failure this repo is built to
  prevent.
- **Restrict any future cache to a single, non-shared worktree**, to
  sidestep cross-worktree acknowledgment-visibility differences. Not
  rejected outright, but insufficient alone: a single worktree's HEAD can
  also move between branches with differing ack-reachability (a checkout
  switch, a rebase), so worktree-scoping narrows the hazard without
  eliminating it.

## Consequences

Any future attempt to solve G-0372's remaining structural cause must either
(a) make cache/watermark validity a function of the reachable acknowledgment
set as well as the commit range, or (b) never memoize a region whose "clean"
verdict depended on an acknowledgment, re-verifying such regions on every
check regardless of range. G-0372 stays open until one of those is designed
and shipped.
