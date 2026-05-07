# aiwf status — 2026-05-07

_160 entities · 0 errors · 3 warnings · run `aiwf check` for details_

## In flight

_(no active epics)_

## Roadmap

### E-13 — Status report _(proposed)_

- **M-048** — Status report: cross-entity summaries + dashboard + time-window views _(draft)_

```mermaid
flowchart LR
  E_13["E-13<br/>Status report"]:::epic_proposed
  M_048["M-048<br/>Status report: cross-entity summaries + dashboard + time-window views"]:::ms_draft
  E_13 --> M_048
  classDef epic_active fill:#d6eaff,stroke:#1a73e8,color:#000
  classDef epic_proposed fill:#f4f4f4,stroke:#888,color:#000
  classDef ms_done fill:#d8f5d8,stroke:#2a8a2a,color:#000
  classDef ms_in_progress fill:#fff3c4,stroke:#caa400,color:#000
  classDef ms_draft fill:#f4f4f4,stroke:#888,color:#000
  classDef ms_cancelled fill:#fbeaea,stroke:#c33,color:#000
```

### E-16 — TDD policy declaration chokepoint (closes G-055) _(proposed)_

- **M-062** — tdd flag on aiwf add milestone with project-default fallback _(draft)_ — ACs 0/8 met (8 open) — tdd: required
- **M-063** — aiwf.yaml tdd.default schema and aiwf init seeding _(draft)_ — ACs 0/7 met (7 open) — tdd: required
- **M-064** — aiwf update migration for existing aiwf.yaml with loud output _(draft)_ — ACs 0/8 met (8 open) — tdd: required
- **M-065** — aiwf check finding milestone-tdd-undeclared as defense-in-depth _(draft)_ — ACs 0/5 met (5 open) — tdd: required

```mermaid
flowchart LR
  E_16["E-16<br/>TDD policy declaration chokepoint (closes G-055)"]:::epic_proposed
  M_062["M-062 (0/8)<br/>tdd flag on aiwf add milestone with project-default fallback"]:::ms_draft
  E_16 --> M_062
  M_063["M-063 (0/7)<br/>aiwf.yaml tdd.default schema and aiwf init seeding"]:::ms_draft
  E_16 --> M_063
  M_064["M-064 (0/8)<br/>aiwf update migration for existing aiwf.yaml with loud output"]:::ms_draft
  E_16 --> M_064
  M_065["M-065 (0/5)<br/>aiwf check finding milestone-tdd-undeclared as defense-in-depth"]:::ms_draft
  E_16 --> M_065
  classDef epic_active fill:#d6eaff,stroke:#1a73e8,color:#000
  classDef epic_proposed fill:#f4f4f4,stroke:#888,color:#000
  classDef ms_done fill:#d8f5d8,stroke:#2a8a2a,color:#000
  classDef ms_in_progress fill:#fff3c4,stroke:#caa400,color:#000
  classDef ms_draft fill:#f4f4f4,stroke:#888,color:#000
  classDef ms_cancelled fill:#fbeaea,stroke:#c33,color:#000
```

### E-18 — Operator-side dogfooding completion (closes G-062, G-064) _(proposed)_

- **M-070** — aiwf doctor warning for missing recommended plugins _(draft)_ — ACs 0/7 met (7 open) — tdd: required
- **M-071** — Install ritual plugins in kernel repo + document operator setup path _(draft)_ — ACs 0/4 met (4 open) — tdd: required

```mermaid
flowchart LR
  E_18["E-18<br/>Operator-side dogfooding completion (closes G-062, G-064)"]:::epic_proposed
  M_070["M-070 (0/7)<br/>aiwf doctor warning for missing recommended plugins"]:::ms_draft
  E_18 --> M_070
  M_071["M-071 (0/4)<br/>Install ritual plugins in kernel repo + document operator setup path"]:::ms_draft
  E_18 --> M_071
  classDef epic_active fill:#d6eaff,stroke:#1a73e8,color:#000
  classDef epic_proposed fill:#f4f4f4,stroke:#888,color:#000
  classDef ms_done fill:#d8f5d8,stroke:#2a8a2a,color:#000
  classDef ms_in_progress fill:#fff3c4,stroke:#caa400,color:#000
  classDef ms_draft fill:#f4f4f4,stroke:#888,color:#000
  classDef ms_cancelled fill:#fbeaea,stroke:#c33,color:#000
```

## Open decisions

| ID | Kind | Title | Status |
|----|------|-------|--------|
| ADR-0001 | adr | Mint entity ids at trunk integration via per-kind inbox state | proposed |
| ADR-0003 | adr | Add finding (F-NNN) as a seventh entity kind | proposed |

## Open gaps

| ID | Title | Discovered in |
|----|-------|---------------|
| G-022 | Provenance model extension surface |  |
| G-023 | Delegated \`--force\` via \`aiwf authorize --allow-force\` |  |
| G-056 | aiwf render output (site/) is not gitignored; pollutes consumer working tree | E-14 |
| G-057 | Stray aiwf binary in repo root from local builds is not gitignored |  |
| G-058 | AC body sections ship empty; no chokepoint enforces prose intent | E-16 |
| G-059 | Branch model: no canonical mapping from entity hierarchy to git branches; epic/milestone work lands on whichever branch is current | M-069 |
| G-060 | Patch ritual is loosely defined; no kernel-level rules for shape, scope, branch, or audit trail |  |
| G-061 | Generic \`aiwf list <kind>\` verb referenced as canonical in contracts plan and shipped contract skill, but never implemented; AI assistants are instructed to invoke a non-existent verb |  |
| G-062 | aiwf doctor does not surface missing recommended plugins; ritual skills (aiwf-extensions, wf-rituals) can be silently absent from a consumer repo with no signal to operator or AI assistant |  |
| G-063 | No defined start-epic ritual: epic activation is a deliberate sovereign act with preflight + optional delegation, but kernel treats it as a one-line FSM flip |  |
| G-064 | Kernel repo dogfooding closed partial (G-038) without installing the ritual plugins (aiwf-extensions, wf-rituals); operator-side surface incomplete here despite framework design assuming rituals are present |  |
| G-065 | No aiwf retitle verb: scope refactors that change an entity's or AC's intent leave frontmatter title fields permanently misleading; only slug rename is supported |  |
| G-067 | wf-tdd-cycle is LLM-honor-system advisory; under load the LLM bypasses RED-first and the branch-coverage HARD RULE without anything mechanical catching it (M-066/AC-1 cycle wrote ~165 lines of impl before any test existed) | M-066 |
| G-068 | Discoverability policy misses dynamic finding subcodes | M-066 |

## Warnings

| Code | Entity | Path | Message |
|------|--------|------|---------|
| entity-body-empty | ADR-0002 | docs/adr/ADR-0002-test-dry-run-delete-me.md | ADR-0002 body section \`## Context\` is empty |
| entity-body-empty | ADR-0002 | docs/adr/ADR-0002-test-dry-run-delete-me.md | ADR-0002 body section \`## Decision\` is empty |
| entity-body-empty | ADR-0002 | docs/adr/ADR-0002-test-dry-run-delete-me.md | ADR-0002 body section \`## Consequences\` is empty |

## Recent activity

| Date | Actor | Verb | Detail |
|------|-------|------|--------|
| 2026-05-07 | human/peter | cancel | aiwf cancel ADR-0002 -> rejected |
| 2026-05-07 | human/peter | add | aiwf add adr ADR-0002 'TEST-DRY-RUN-DELETE-ME' |
| 2026-05-07 | human/peter | render-roadmap | aiwf render roadmap |
| 2026-05-07 | human/peter | promote | aiwf promote E-17 active -> done |
| 2026-05-07 | human/peter | promote | aiwf promote G-066 open -> addressed |

