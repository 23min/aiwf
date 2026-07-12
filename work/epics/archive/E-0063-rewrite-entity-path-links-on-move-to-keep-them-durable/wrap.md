# Epic wrap — E-0063

**Date:** 2026-07-12
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0063-rewrite-entity-path-links-on-move-to-keep-them-durable
**Merge commit:** aa57a1b6

## Milestones delivered

- M-0245 — Shared link-destination rewrite primitive (merged 0ee645aa)
- M-0246 — Wire archive to rewrite link destinations on sweep (merged 6e422090)
- M-0247 — Wire rename and retitle to rewrite link destinations (merged 56e73e88)
- M-0248 — Unify reallocate onto the shared rewrite primitive (merged d2ea378f)
- M-0251 — Handle #fragment / ?query suffixes in link-destination rewrite (merged 7343849d)

## Summary

Markdown path-links between entity files now survive every file-moving verb.
A shared, pure, idempotent link-destination rewrite primitive
(`internal/verb/linkrewrite.go`, generalized from `rewidth`'s region-splitter
to handle relative destinations and `#fragment`/`?query` suffixes) is wired
into `archive`, `rename`, `retitle`, and `reallocate`, so a link that pointed
at an entity's old path is rewritten to its new path in the same commit that
moves the entity — instead of rotting silently. `reallocate`'s unification
(originally flagged optional/droppable in the epic scope) shipped alongside
the rest: it replaced a blind id-token substring replace that happened to
land the right path but wasn't link-region-scoped, fixing a real corruption
case (a URL-shaped destination containing the id as a substring) along the
way.

## ADRs ratified

- ADR-0033 — Entity path-links are first-class and rewritten on move

## Decisions captured

- none

## Follow-ups carried forward

- G-0396 — `addressed_by_commit` SHA is merge-fragile; derive closure from git
  history (discovered during this epic; explicitly deferred in its own body —
  "recorded but not scheduled," symptomless today, a future milestone-sized
  change if a read surface ever needs it)

## Handoff

The link-rewrite primitive and its four verb integrations are the durable
deliverable — any future move-emitting verb (or a hand-written `git mv`
alternative) has the shared primitive to reach for rather than reinventing
region-aware rewriting. Nothing is deliberately left half-done: the epic's
one explicitly-optional milestone (M-0248) shipped, and the primitive's one
known gap at epic-open time (fragment/query suffixes, G-0409) closed via
M-0251. Non-entity narrative (`docs/*.md`, `README`) remains
detection-only via `wf-doc-lint`'s existing markdown-link-integrity check,
per the epic's explicit out-of-scope boundary — unchanged by this epic.
