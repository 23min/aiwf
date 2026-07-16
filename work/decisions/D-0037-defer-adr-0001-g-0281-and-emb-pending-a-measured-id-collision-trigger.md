---
id: D-0037
title: Defer ADR-0001, G-0281, and EMB pending a measured id-collision trigger
status: accepted
relates_to:
    - ADR-0001
    - G-0281
    - ADR-0025
    - E-0052
---
# D-0037 — Defer ADR-0001, G-0281, and EMB pending a measured id-collision trigger

> **Date:** 2026-07-16 · **Decided by:** human/peter

## Question

Three overlapping, unratified designs answer "how does an entity get a stable
id under concurrent branches, agents, or machines": `ADR-0001` (mint-at-trunk-
integration via slug pre-mint), `G-0281` (eager allocation via a coordination
ref, built via `M-0186`/`M-0187`), and "EMB" (an ephemeral-mutation-branch
variant surfaced while pressure-testing `G-0281`). `docs/initiatives/id-lifecycle.md`
reconciled all three against each other and against the already-shipped cheap
mechanism (`ADR-0025`/`E-0052`). Should any of the three be built now, or is
the friction they'd solve still small enough to defer — and if deferred, on
what checkable basis would that call get revisited?

## Decision

Defer `ADR-0001`, `G-0281`, and EMB. Build none of them now. `M-0187`
(`G-0281`'s implementation milestone) is cancelled on this reasoning; `G-0281`
itself stays `open` (the design is not rejected, only not being built yet) and
`ADR-0001` stays `proposed` (the design was never found wrong, only
unnecessary at today's measured friction).

Revisit when either: the `aiwf-verb: reallocate` rate climbs meaningfully
above the measured ~3.4% of `aiwf-verb: add` events, or the reallocate bursts
stop clustering into identifiable concurrent-work episodes and become steady
background noise across ordinary activity. Re-run
`git log --all --grep="^aiwf-verb: reallocate$"` against
`git log --all --grep="^aiwf-verb: add$"` periodically (e.g., at each epic
wrap) to check the trigger, rather than relying on memory.

## Reasoning

`docs/initiatives/id-lifecycle.md` measured this repo's own history: 34
reallocate events against 986 add events (~3.4%) as of 2026-07-04, clustered
into five or six identifiable multi-day episodes rather than spread evenly —
every one resolved by a single `aiwf reallocate` call. Re-measured at decision
time (2026-07-16): 35 reallocate / 1129 add (~3.1%), with only one new
reallocate event since the prior measurement and no new burst — the trigger
has not fired.

Each of the three deferred mechanisms is real, non-trivial engineering:
`ADR-0001` touches all six entity kinds' minting model; `G-0281` adds a
dedicated coordination ref plus an import verb and carries an unbounded
discoverability-blindness window (a pending entity in the side ref is
invisible to grep, editor search, and LLM cold-reads until imported); EMB has
two open, unresolved design gaps (nested-invocation detection; the "checked
out in place" property colliding with its own retry-recompute step).
Building any of them now would be engineering against friction the data says
is small and already absorbed by the cheap, shipped mechanism
(`ADR-0025`/`E-0052`) plus `aiwf reallocate`.

Alternatives considered: accepting `ADR-0001` now (rejected — the structural
design is sound but not yet justified by measured friction); rejecting
`ADR-0001` outright (rejected — nothing about the design was found wrong, and
the FSM has no path back from `rejected` without a new ADR superseding it,
which overstates what's actually being said here, which is "not yet").

## Consequences

- `ADR-0001` and `G-0281` remain visible, findable designs (`aiwf show
  ADR-0001`, `aiwf show G-0281`) rather than silently abandoned — a future
  reopening starts from them, not from scratch.
- The trigger condition is now recorded as a decision, not just prose in an
  initiative doc — the next person or agent evaluating "should we build
  ADR-0001 yet" has a checkable, dated answer instead of reconstructing one
  from `docs/initiatives/id-lifecycle.md`.
- No code changes; no follow-on migration.
