# Epic wrap — E-0066

**Date:** 2026-07-17
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0066-add-a-priority-field-to-gaps-and-decisions-for-a-filterable-backlog
**Merge commit:** eb831de3

## Milestones delivered

- M-0261 — Add the priority field, its validation, and drift chokepoints (merged d370939a)
- M-0262 — Add the priority write surface: set-priority verb and add --priority (merged b8b4f71c)
- M-0263 — Add the priority read surface: list/status filter, envelope, show (merged c192254c)
- M-0264 — Render a priority badge in the HTML site (merged 75b7986d)

## Summary

Gaps and decisions can now carry a `priority` (urgent/high/medium/low), the closed-set field G-0078 designed to replace the ad-hoc inline `Severity:` prose nothing could query. The field, its validation, and drift chokepoints landed first (M-0261); the write surface (`aiwf add --priority`, `aiwf set-priority`) and read surface (`aiwf list`/`aiwf status --priority`, the JSON envelope, `aiwf show`) followed independently, both depending only on M-0261; the HTML render badge (M-0264) closed the loop with a visual surface, human-verified against a real snapshot of this repo's own tree. Every surface the epic's Success criteria named is met. Sort-by-priority ordering was deliberately deferred to G-0420 from the start, not discovered mid-flight.

## ADRs ratified

- ADR-0034 — Enforce per-kind field applicability via a presence-scope check rule (accepted)

## Decisions captured

- none

## Follow-ups carried forward

- G-0420 — the deferred sort-by-priority tiebreaker (pre-existing, named in the epic spec's own Out-of-scope section from planning; not a mid-epic discovery)

## Doc findings

Scoped sweep across every markdown file touched on this branch since it diverged from main (the epic's own wrap.md and ADR-0034, and the SKILL.md files touched across all four milestones: aiwf-add, aiwf-check, aiwf-list, aiwf-render, aiwf-set-priority, aiwf-show, aiwf-status). No findings — 8 files checked: no TODO/FIXME markers (one incidental match inside `aiwf-check`'s own prose describing the `entity-body-empty` rule's HTML-comment behavior, not an outstanding marker), no heading-hierarchy jumps, no markdown links to verify, and every backticked `aiwf <verb> --flag` invocation resolves against the current binary's `--help` output.

## Handoff

Every surface named in E-0066's Success criteria is done: creation-time and post-creation writes, both filter reads, the JSON envelope, and the HTML badge. Nothing is left half-built. G-0420 (sort ordering) is the one deliberately-deferred follow-up, tracked as its own gap rather than left implicit. No open design questions remain — both items in the epic's own `## Open questions` table resolved during implementation (severity landed as warning, consistent with `area_unknown`'s posture; the render/list contract question resolved by direct inspection — this repo's own Go struct shapes are unrelated to the `contract` entity kind's consumer-facing schema-binding feature).
