---
id: M-0151
title: Agent-target seam in the materializer (Claude writer behind the seam)
status: in_progress
parent: E-0038
depends_on:
    - M-0149
    - M-0150
tdd: required
acs:
    - id: AC-1
      title: Materializer takes a target param; Claude target preserves M2/M3 behavior
      status: met
      tdd_phase: done
    - id: AC-2
      title: Seam contract test asserts target-to-output mapping; accepts a 2nd target
      status: open
      tdd_phase: red
---
## Goal

Refactor the materializer to be parameterized by an agent target, with the Claude writer (`.claude/skills/`, `.claude/agents/`) implemented behind the seam, so additional targets (Codex `.agents/skills/`, down-converted Cursor/Copilot rules) become new writers rather than a rewrite.

## Context

M2/M3 wrote the Claude locations concretely. This milestone extracts the target seam now that there is a concrete writer to abstract over — per CLAUDE.md KISS/YAGNI, abstract on the second case, not speculatively. It unblocks the agent-agnostic future without building every target. The seam may be pulled forward into M2 if that proves cheaper at implementation time; that is a just-in-time call, not a planning-time one.

## Acceptance criteria

## Constraints

- Behavior-preserving for the Claude target — M2/M3 tests stay green with no observable change for Claude consumers.
- Do not build out non-Claude target writers here (deferred per the epic scope and the M6-deferral gap).

## Design notes

- ADR-0014 §4 (agent-target abstraction). SKILL.md is a cross-vendor open standard (agentskills.io; OpenAI Codex reads the identical frontmatter from `.agents/skills/`), so the first non-Claude target is near-verbatim.
- CLAUDE.md KISS/YAGNI — the seam is *extracted* from a concrete writer, not speculated ahead of one.

## Surfaces touched

- `internal/skills/` (materializer target parameterization), the materialize call sites in `init`/`update`.

## Out of scope

- Implementing Codex / Cursor / Copilot writers — deferred to a follow-up gap.

## Dependencies

- M2 and M3 — the concrete Claude writers this milestone abstracts over.

## References

- **ADR-0014** (§4), **E-0038**.

### AC-1 — Materializer takes a target param; Claude target preserves M2/M3 behavior

### AC-2 — Seam contract test asserts target-to-output mapping; accepts a 2nd target

