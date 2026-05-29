---
id: M-0150
title: Embed + materialize ritual agents (.claude/agents/) and templates
status: draft
parent: E-0038
depends_on:
    - M-0149
tdd: required
---
## Goal

Extend the embed+materialize pipeline to the rituals' agents (→ `.claude/agents/`) and templates (→ their referenced locations), with manifest ownership and gitignore coverage, treating agents exactly like skills.

## Context

M2 delivers ritual skills. The rituals also ship four agents (`planner` / `builder` / `reviewer` / `deployer`) and a set of templates. This milestone completes artifact coverage so `aiwf init` delivers the full ritual set. Hooks are explicitly *not* part of the rituals (ADR-0014 §3), so no hook surface is added.

## Acceptance criteria

## Constraints

- Agents are materialized like skills — same manifest ownership and gitignore discipline; user-authored agents are never clobbered.
- No new hook installation (ADR-0014 §3). The only managed hooks remain aiwf's existing git hooks.

## Design notes

- ADR-0014 §3 (artifact coverage; agents-as-skills; hooks-not-rituals).

## Surfaces touched

- `internal/skills/` (embed + materialize for agents/templates), `internal/initrepo/`, the manifest.

## Out of scope

- Per-target agent handling for non-Claude agents — M4.
- The marketplace sunset — M5.

## Dependencies

- M2 — the materializer extended for skills, which this milestone extends further.

## References

- **ADR-0014** (§3), **E-0038**.
