---
id: ADR-0049
title: A create must be complete; an edit or a push must not regress
status: accepted
supersedes:
    - ADR-0048
---
> **Date:** 2026-09-14 · **Decided by:** human/peter

## Context

ADR-0048 kept ADR-0043's push seam, which treats every required section absent at
HEAD, in a body the range changed, as a violation. That is the wrong question for
two reasons. It refuses the commit `aiwf edit-body` has just made, since that verb
permits an edit keeping an existing omission and neither path has `--force`. And
it blocks authors over omissions they did not introduce: measured 2026-09-14 at
`09d2058cc`, 55 active entities omit a required section, and 29 of them have had
their body changed since creation — each file's history read with
`git log --follow --format=%H --name-only`, the path resolved at every commit
through its renames, and the bytes after the frontmatter compared between
consecutive versions. A walk that does not follow renames counts 27.

Judging the push commit by commit along the first-parent line was rejected: it
credits a `--no-ff` drop to the merge rather than to the commit that made it,
sees no write at all for a merge that adopts an older copy, and refuses a push
whose section came and went. A warning for inherited omissions was rejected as a
second copy of a debt nobody must act on, and a baseline ledger as a mandate with
no owner. This decision replaces D-0092, which answered the push-seam question
first.

## Decision

Each seam holds a body to where it started.

- `aiwf add` requires a complete body; `--force --reason` overrides it.
  `aiwf import` is excluded from the verb seams on its deprecation (G-0667).
- `aiwf edit-body` refuses a write that drops a section the committed body
  carries. It has no `--force`; a deliberate removal is recorded with
  `aiwf acknowledge illegal <sha>`.
- The push compares each entity, by id, at its starting point and at HEAD, and
  refuses a required section present at the start and absent at HEAD, reported as
  `entity-body-section-dropped` at error severity against the commit that removed
  it. The starting point is where the branch left its base, not wherever the base
  has since reached. An entity present there begins from that body, matched
  through its prior ids after a reallocation; one the range creates begins from
  the body an `aiwf import` or a forced `aiwf add` commit wrote, and is otherwise
  held to every required section, since an unforced `aiwf add` cannot write an
  incomplete body. A section the entity already lacks on the configured trunk,
  under its own id, is exempt: trunk lacks it whether or not the push lands, so
  refusing recovers nothing.

A violation is what ADR-0043 defined and this carries forward: a required section
not present as a top-level `## ` heading, with sections beyond the set legal and
order not enforced. Emptiness is a separate property at a separate seam. No rule
joins `check.Run`.

## Consequences

- Entities that omitted a required section before these seams held never
  converge: no seam judges a body that a write or a push did not change.
- A forced `aiwf add` is its entity's starting point, so what it left out is never
  asked for again.
- The create exemption reads the commit's own trailers, so a hand-written create
  stamped `aiwf-verb: import`, or `aiwf-verb: add` with `aiwf-force`, is taken at
  its word. It rests on `aiwf add` refusing an incomplete unforced body: loosen
  that gate and the push refuses creates the verb permitted.
- The trunk exemption keys on trunk's state, not on who removed the section. An
  author who drops one trunk has already lost is not refused, even from a branch
  that never merged trunk.
- The commit a finding names is best-effort. It can be the wrong one where history
  discards a removal and a later merge publishes an older copy anyway, and where
  two entities hold one id across a collision; the refusal and the acknowledgment
  do not depend on it, and only the report is pinned.
- The push judges only ranges the provenance audit resolves, so a branch started
  from a local ref is judged by nothing at its first push (G-0679).
