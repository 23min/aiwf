---
id: ADR-0024
title: Shared ritual content lives as a referenced reference-skill
status: rejected
---

## Context

Several rituals repeat the same operational how-to verbatim. The trailered-commit
/ trailered-merge block (`git commit --trailer "aiwf-verb: …" --trailer
"aiwf-entity: …" --trailer "aiwf-actor: …"`, plus the "variant casings fail the
trailer-keys policy" caveat and the "resolve identity from `git config
user.email`" note) appears in `aiwfx-wrap-epic` (twice), `aiwfx-wrap-milestone`
(twice), and partially in `aiwfx-release`. The "why an `aiwf-verb` trailer on a
`git merge` commit" explanation is duplicated between the two wrap rituals. This
duplication has already drifted: `aiwfx-wrap-milestone` lacked the trailer
prescription its sibling had until G-0219 caught it via a same-cycle operator
error.

Two costs: (1) drift — a fix in one copy silently misses the others; (2) tokens —
the duplicated prose is loaded in every ritual body on every invocation, whereas
the content is only needed at one step.

It cannot live in `CLAUDE.md`: that is the kernel repo's own file and is not
materialized into consumer repos, where the rituals run. The shared content must
ship through the same channel as the rituals themselves.

## Decision

Shared, reusable ritual content lives as a **dedicated reference skill** in the
embedded-rituals tree (e.g. `wf-commit-trailers`), materialized into the
consumer's `.claude/skills/` by `aiwf init` / `aiwf update` exactly like every
other ritual. Rituals stop inlining the shared block and instead reference the
skill by name ("compose the trailered commit per `wf-commit-trailers`").

Rejected alternative: a materialized fragment `@`-imported inside a `SKILL.md`
(mirroring `CLAUDE.md`'s `@.claude/aiwf-guidance.md` import). It is unproven that
Claude Code resolves `@` imports *inside* a skill body; the reference-skill route
uses only the proven skill-materialization and skill-invocation mechanisms.

## Consequences

- **Downstream-available:** the reference skill ships in the same materialization
  as the rituals that cite it; no consumer-side setup.
- **Token-positive:** a referenced skill loads on demand at the step that needs
  it, instead of duplicated prose loaded in every ritual body on every turn.
  Deduplication here reduces per-invocation context.
- **Drift-policed:** the existing `skill_coverage` policy validates that backticked
  `` `aiwf <verb>` `` and skill cross-references resolve, so a dangling reference
  fails CI; the single source means a fix lands once.
- **New authoring rule:** future ritual authors extract a shared block into a
  reference skill rather than copy-pasting it. The reference skill is the single
  source of truth for that content.
- **Cost:** one extra skill-invocation step at the point of use, and one more
  materialized skill in `.claude/skills/`.
