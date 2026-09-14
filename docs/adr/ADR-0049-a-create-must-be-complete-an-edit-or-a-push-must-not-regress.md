---
id: ADR-0049
title: A create must be complete; an edit or a push must not regress
status: accepted
supersedes:
    - ADR-0048
---
> **Date:** 2026-09-14 · **Decided by:** human/peter

## Context

ADR-0048 kept ADR-0043's push seam, which treats every required section absent
at HEAD, in a body the range changed, as a violation. That is the wrong question
for two reasons. It refuses the commit `aiwf edit-body` has just made, since that
verb permits an edit keeping an existing omission and neither path has `--force`.
And it blocks authors over omissions they did not introduce: measured 2026-09-14,
55 active entities omit a required section, and 29 of them have had their body
changed since creation, following each file through renames and reallocations
with `git log --follow`.

Judging the push commit by commit along the first-parent line was rejected: it
misses a section removed on a branch merged with `--no-ff`, and one carried off
by a later rename. A warning for inherited omissions was rejected as a second
copy of a debt nobody must act on, and a baseline ledger as a mandate with no
owner. This decision replaces D-0092, which answered the push-seam question first.

## Decision

Each seam holds a body to where it started.

- `aiwf add` requires a complete body; `--force --reason` overrides it.
  `aiwf import` is excluded.
- `aiwf edit-body` refuses a write that drops a section the committed body carries.
- The push compares each entity, by id, at its starting point and at HEAD, and
  refuses a required section present at the start and absent at HEAD, reported
  as `entity-body-section-dropped` against the commit that removed it. An entity
  present when the range starts begins from that body, matched through its prior
  ids if reallocated. One the range creates begins from the body an `aiwf import`
  or a forced `aiwf add` commit wrote, and from nothing otherwise: an unforced
  `aiwf add` cannot write an incomplete body, so a commit claiming one passed no
  verb. A section the entity already lacks on the configured trunk is exempt,
  since that removal is not the pushing author's.

ADR-0043's definition of a violation stands, and no rule joins `check.Run`. A
deliberate removal is recorded with `aiwf acknowledge illegal <sha>`.

## Consequences

- Entities that omitted a required section before these seams held never
  converge: no seam judges a body that a write or a push did not change.
- A forced `aiwf add` is its entity's starting point, so what it left out is
  never asked for again.
- The push judges only ranges the provenance audit resolves, so a branch pushed
  once and merged on the server is judged by nothing (G-0679).
