---
id: E-0056
title: Extend the id chokepoint across shipped surfaces; strip provenance prose
status: proposed
---
## Goal

Every surface aiwf materializes into a consumer's `.claude/` — verb and ritual
skills, role-agent cards, entity templates, the always-on guidance fragment, and
the statusline — reads as imperative, consumer-scoped instruction: no
aiwf-internal entity ids, no development history, no provenance tags, no
rationale or war-stories, and no dead references to artifacts that do not ship.
The id chokepoint covers every shipped surface, so the leak class cannot recur.

## Context

Successor to E-0048 (*Skill & ritual content integrity*), which built the
`skill-body-id` check (G-0299) but scoped it to `SKILL.md` bodies only. A
downstream audit (G-0348) found the check has four blind spots — it skips the
`description:` frontmatter, non-`SKILL.md` files (templates, agent cards),
`embedded-guidance/`, and `embedded-statusline/` — and that beyond ids, the same
surfaces carry aiwf's own history and rationale. The trigger: a consumer AI was
told epic activation is "human-sovereign per `M-0095`", an id meaningless in its
repo. The statusline is the largest offender, its comments narrating aiwf's own
gap history as inline provenance tags.

This epic closes the coverage gaps in the mechanical chokepoint and removes the
existing prose that violates the consumer-scoping rule. Both halves land under
one goal because extending the check forces the id cleanup in the same change,
and the surfaces needing a prose sweep are the same ones the extended check will
police.

## Scope

### In scope

- Extend the id chokepoint to scan the `description:` field, all materialized
  `*.md` under the ritual tree (templates, agent cards), `embedded-guidance/`,
  and `embedded-statusline/` comments — with a firing fixture per surface. Code
  spans and link destinations stay exempt (runnable examples and ADR doc-links
  remain legal).
- Clean every existing real-id leak the extended check would flag (statusline
  comments, `aiwfx-start-epic` description, the `epic-spec.md` template example).
- Strip development history and rationale from shipped prose: the statusline's
  provenance comments, the "v1 ... is gone" asides, and the "why date is in the
  body" argumentation blocks in the ADR / decision templates.
- Extend this repo's `CLAUDE.md` § "Skills policy" id-reference paragraph to the
  full surface list and the broader content class (history / provenance /
  rationale), as the human-review backstop for what the check cannot mechanize.
- Resolve the dead ADR doc-links in ritual skills (G-0315): they use the wrong
  relative depth (dead even in this repo) and target a `docs/adr/` tree a
  consumer does not have. Encode the reference rule in the `aiwfx-record-decision`
  skill body — a behavioral skill states its behavior directly and does not link
  to a non-shipping decision record — and rework the existing dead links to
  match. No decision entity is created.

### Out of scope

- Code examples and their embedded ids — kept exempt by decision; a runnable
  `aiwf <verb> <id>` example is legitimate.
- Status-value vocabulary (`superseded`, `deprecated`, `retired`, `rejected`) in
  FSM tables and enums — correct domain language, not history.
- The stale `branch-not-found` code reference in the start rituals (G-0224),
  owned by lifecycle epic E-0049; and new kernel verbs beyond the chokepoint.

## Constraints

- The extended check stays inert in a consumer repo: it scans only the
  authoring-source trees under `internal/skills/`, which a consumer does not
  have.
- Extending the check and cleaning the leaks land together — a check that fires
  on un-cleaned surfaces would fail its own pre-push.
- The authoring principle lives in `CLAUDE.md` (develop-aiwf guidance), never in
  the shipped `aiwf-guidance.md`: a consumer never authors skill bodies, and a
  "shipped prose must be consumer-scoped" rule shipped to consumers would violate
  itself.
- Dropping the ritual-skill ADR doc-links interacts with the prior
  discoverability ACs that required an ADR reference (`M-0104/AC-2` and
  siblings): the rework must reconcile those ACs, not silently contradict them.

## Success criteria

- The id chokepoint fires on a real id planted in any newly-covered surface —
  description, template, agent card, guidance, statusline comment — proven by a
  firing fixture for each.
- No shipped surface under `internal/skills/embedded{,-rituals,-guidance,
  -statusline}/` carries a real aiwf-internal id in prose, a provenance tag, or a
  development-history aside (the code-example carve-out excepted).
- This repo's `CLAUDE.md` § "Skills policy" states the broadened authoring
  principle over the full surface list and content class.
- No shipped ritual skill links to a non-shipping `docs/` path, and the
  `aiwfx-record-decision` skill body encodes the reference rule — so no dead
  "read more" link ships (G-0315).

## Milestones

Refined via `aiwfx-plan-milestones`:

- Extend the chokepoint coverage + forced id cleanup + firing fixtures.
- Strip the history/rationale axis + broaden the `CLAUDE.md` authoring principle.
- Drop the dead ADR doc-links (G-0315): encode the reference rule in the
  `aiwfx-record-decision` skill, rework the links. No decision entity.
