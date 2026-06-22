---
id: G-0274
title: No batch resolution for id collisions; reallocate is one-at-a-time
status: open
---
## Problem

`aiwf reallocate <id>` cures one id collision per invocation: it renumbers a
single entity and rewrites that entity's cross-references in one commit
(`internal/cli/reallocate/`). When a merge surfaces several `ids-unique`
collisions at once — two branches that each allocated a run of ids before either
published — the operator must read each finding, pick which side renumbers, and
run `aiwf reallocate` once per colliding id. The cure is correct but manual and
one-at-a-time, exactly when the operator is mid-merge and least wants ceremony.

This addresses collision class 3 from `G-0272`'s catalogue — genuinely
concurrent cross-machine allocation, which *cannot* be prevented with local
information. The design stance there is explicit: do not try to eliminate class 3;
make its cure cheap. `G-0272` (worktree heads) and `G-0273` (fetch) shrink the
window down to this residual; this gap makes resolving the residual a reflex
rather than a chore.

## Direction (to converge at the milestone)

- A batch resolution path — e.g. `aiwf reallocate --auto` — that reads the
  `ids-unique` findings against the configured trunk ref, renumbers every
  working-tree id that collides, and rewrites all cross-references, ideally in a
  single commit (or one commit per id with a clear summary).
- The non-trivial decisions to settle at the milestone:
  - **Which side renumbers** when both trees carry the colliding id. Default to
    renumbering the *local working-tree* side (trunk is the published authority),
    with an override.
  - **Atomicity** — one commit for the whole batch vs. one per id. One commit is
    cleaner for history; per-id keeps `aiwf history` granular. Reconcile with the
    "every mutating verb produces exactly one commit" commitment.
  - **Trailers** — each renumber must still stamp `aiwf-verb: reallocate` +
    `aiwf-prior-entity:` so `aiwf history` and any trailer-keyed check see the
    event (the discipline CLAUDE.md pins against the `git mv` shortcut).

## Why not just keep one-at-a-time

The per-id verb is the right *primitive*; this gap is about the *batch
ergonomics* on top of it. Nothing here weakens the existing single-id path — it
adds a sweep that calls the same machinery, so the audit trail and reference
rewriting stay identical.

## Provenance

Emerged from a design discussion on reducing merge-time id-collision risk
(2026-06-22). Third gap of a three-gap set headed by `G-0272`; sibling of
`G-0273` (fetch-before-allocate). Cheapens the cure for collision class 3, the
unpreventable residual after `G-0272` and `G-0273` shrink the window.
