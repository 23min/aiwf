# aiwf status — 2026-05-07

_133 entities · 0 errors · 0 warnings_

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

_(no milestones)_

## Open decisions

_(none)_

## Open gaps

| ID | Title | Discovered in |
|----|-------|---------------|
| G-022 | Provenance model extension surface |  |
| G-023 | Delegated \`--force\` via \`aiwf authorize --allow-force\` |  |
| G-055 | Milestone creation does not require a TDD policy declaration | E-14 |
| G-056 | aiwf render output (site/) is not gitignored; pollutes consumer working tree | E-14 |
| G-057 | Stray aiwf binary in repo root from local builds is not gitignored |  |

## Warnings

_(none)_

## Recent activity

| Date | Actor | Verb | Detail |
|------|-------|------|--------|
| 2026-05-07 | human/peter | add | aiwf add gap G-057 'Stray aiwf binary in repo root from local builds is not gitignored' |
| 2026-05-07 | human/peter | add | aiwf add gap G-056 'aiwf render output (site/) is not gitignored; pollutes consumer working tree' |
| 2026-05-07 | human/peter | add | aiwf add gap G-055 'Milestone creation does not require a TDD policy declaration' |
| 2026-05-07 | human/peter | promote | aiwf promote E-14 active -> done |
| 2026-05-07 | human/peter | promote | aiwf promote M-061 in_progress -> done |

