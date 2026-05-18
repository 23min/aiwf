---
id: M-0121
title: 'Pass A audit: catalog legal-workflow rules from existing surfaces'
status: in_progress
parent: E-0033
depends_on:
    - M-0120
tdd: advisory
acs:
    - id: AC-1
      title: Audit catalog exists at canonical path with per-source sections in spec order
      status: open
    - id: AC-2
      title: All nine audit sources covered with at least one rule or explicit no-rules note
      status: open
    - id: AC-3
      title: Each rule row has the six-column schema with non-empty fields
      status: open
    - id: AC-4
      title: 'Catalog schema internally consistent: unique sequential ids, totals match'
      status: open
---
## Goal

Walk every source of pre-existing legality statements in the repo and extract them into a single draft catalog with citations. The output is **evidence**, not a spec — Pass C (M-0123) reconciles this catalog with M-0122's first-principles catalog into the canonical Go spec.

## Sources to audit

In order of mechanical authority (most rigorous first):

| Source | Format | What to extract |
|---|---|---|
| `internal/entity/transition.go` | Go maps | Per-kind FSM tables; (state, event, next-state) triples |
| `internal/policies/*.go` | Go tests | Each policy = a legality rule already mechanized in CI |
| `internal/check/*.go` | Go code | Each finding code = a class of illegal state |
| Cobra verb definitions under `cmd/aiwf/` and `internal/cli/<verb>/` | Go code | Per-verb pre/postconditions surfaced in flag validation and RunE bodies |
| ADRs under `docs/adr/` | Markdown | Cross-cutting workflow constraints (ADR-0004 archive, ADR-0008 ids, etc.) |
| `docs/pocv3/design/design-decisions.md` | Markdown | The 10 kernel commitments |
| `CLAUDE.md` | Markdown | Workflow constraints (trunk-based, AC mechanical evidence, etc.) |
| Skills under `.claude/skills/` + rituals plugin | Markdown | Narrative workflows (start-milestone, wrap-milestone, etc.) — lower authority but useful for cross-checking |
| `aiwf <verb> --help` | CLI text | Per-verb terse pre/postconditions |

## Output

A markdown file under `docs/pocv3/design/legal-workflows-audit.md` with one row per legality rule:

```
| Rule id | Source | Citation | Scope | Statement | Severity if violated |
```

The rule ids are sequential within the audit doc (R-AUDIT-001..N) — they are *not* an aiwf entity kind. They're internal references that Pass C will map onto spec cells.

## Acceptance criteria

(Added via `aiwf add ac` once M-0120's ADR ratifies the methodology and the catalog schema is settled.)

## Approach

- Walk sources top-down (most-mechanical first), so the lower-authority sources can cross-reference the higher.
- For each source, produce a per-source section in the audit doc with its extracted rules.
- Mark rules that are *implicit* (we believe the source intends X but doesn't state it directly) as such, so Pass C can flag them for explicit decision.
- No spec authorship in this milestone — only extraction.

## What this milestone does *not* do

- Does not invent new legality rules (that's M-0122).
- Does not reconcile contradictions between sources (that's M-0123).
- Does not produce Go code.

### AC-1 — Audit catalog exists at canonical path with per-source sections in spec order

### AC-2 — All nine audit sources covered with at least one rule or explicit no-rules note

### AC-3 — Each rule row has the six-column schema with non-empty fields

### AC-4 — Catalog schema internally consistent: unique sequential ids, totals match

