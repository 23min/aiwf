---
id: M-0229
title: Drop dead doc-links; encode reference discipline in record-decision
status: in_progress
parent: E-0056
depends_on:
    - M-0227
tdd: advisory
acs:
    - id: AC-1
      title: aiwfx-record-decision encodes the self-contained reference rule
      status: open
---
## Goal

The rule for how a decision is referenced — so a reference never becomes a dead
link in a consumer's materialized `.claude/` — is encoded in the
`aiwfx-record-decision` skill, the ritual that authors decisions. The existing
ritual-skill doc-links that violate it are reworked to match. No decision entity
is created; the behavior lives in the skill body.

## Approach

The ritual-skill ADR doc-links are dead twice over (G-0315): six `../` segments
where the `SKILL.md` sits seven levels down (broken even in this repo), and they
target a `docs/adr/` tree a consumer's materialized `.claude/` does not contain.
Fixing the depth alone still leaves the consumer a dead link, so the resolution
is to drop the non-shipping references and state the behavior directly.

- **Encode the reference behavior in the `aiwfx-record-decision` skill body.**
  The skill that records decisions is where the rule lives: a behavioral skill
  states its behavior directly and does not link to a decision record or design
  doc under `docs/` (which does not materialize into a consumer's tree); a
  decision's rationale lives in its own entry, authored via this skill, not in a
  link from a behavioral skill. This is skill guidance, **not** a decision
  entity — do not create an ADR or `D-` entity for it.
- **Rework the existing dead links** across the `aiwfx-*` / `wf-*` ritual skills:
  drop the `docs/adr/` doc-links, rewording to self-contained imperative
  instruction that conveys the same behavioral fact. Reconcile the prior
  discoverability ACs that required an ADR reference (`M-0104/AC-2` and siblings)
  so they no longer mandate a now-removed link.

## Acceptance criteria

Sketch — formalized at start-milestone:

1. The `aiwfx-record-decision` skill body encodes the reference-handling rule,
   pinned by a structural test on the named section. No decision entity is
   created by this milestone.
2. A structural assertion over the shipped ritual skills holds: no `docs/adr/`
   doc-link (or other path into the repo's non-shipping `docs/` tree) remains.
3. The prior discoverability ACs that required an ADR reference no longer mandate
   one (reconciled, not silently contradicted).

### AC-1 — aiwfx-record-decision encodes the self-contained reference rule

