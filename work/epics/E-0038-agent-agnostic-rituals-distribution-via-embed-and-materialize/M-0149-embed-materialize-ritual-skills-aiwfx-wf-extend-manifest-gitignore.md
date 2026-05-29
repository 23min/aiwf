---
id: M-0149
title: Embed + materialize ritual skills (aiwfx-/wf-); extend manifest + gitignore
status: in_progress
parent: E-0038
depends_on:
    - M-0148
tdd: required
acs:
    - id: AC-1
      title: aiwf init writes embedded ritual skills to .claude/skills/{aiwfx,wf}-*
      status: met
      tdd_phase: done
    - id: AC-2
      title: Manifest and gitignore own the skill dirs; update refreshes, no clobber
      status: open
      tdd_phase: green
    - id: AC-3
      title: aiwf check is clean against a repo materialized with ritual skills
      status: open
      tdd_phase: red
---
## Goal

Embed the vendored ritual skills (`aiwfx-*`, `wf-*`) into the engine binary via `go:embed` and extend the `init`/`update` materializer (plus the `.aiwf-owned` manifest and `.gitignore` patterns) so they are written into the consumer repo's `.claude/skills/` alongside the existing verb skills.

## Context

M1 vendored the snapshot. This milestone makes `aiwf init` / `aiwf update` actually deliver the ritual *skills* — the largest and most-used slice of the rituals — through the same marker-managed pipeline that already ships the 16 verb skills. After it lands, an operator gets the planning, lifecycle, and engineering skills with no `/plugin` step.

## Acceptance criteria

## Constraints

- Reuse the existing materializer / manifest / gitignore mechanism; do not fork a parallel path.
- Never clobber user-authored skills under `.claude/skills/` (the existing guarantee is preserved).
- Writes the Claude location directly — the target seam is M4, not this milestone.

## Design notes

- ADR-0014 §1 (build-time embed) and §3 (artifact coverage).
- CLAUDE.md commitment #5 (marker-managed artifacts regenerated on `init`/`update`) extended to ritual skills.

## Surfaces touched

- `internal/skills/` (embed directive + `Materialize`), `internal/initrepo/` (gitignore patterns), the `.aiwf-owned` manifest.

## Out of scope

- Agents and templates — M3.
- The agent-target abstraction — M4 (this milestone writes the Claude location directly).

## Dependencies

- M1 — the vendored snapshot to embed.

## References

- **ADR-0014** (§1, §3), **G-0177**, **E-0038**.

### AC-1 — aiwf init writes embedded ritual skills to .claude/skills/{aiwfx,wf}-*

### AC-2 — Manifest and gitignore own the skill dirs; update refreshes, no clobber

### AC-3 — aiwf check is clean against a repo materialized with ritual skills

