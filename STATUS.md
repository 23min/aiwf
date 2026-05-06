# aiwf status — 2026-05-06

_110 entities · 0 errors · 0 warnings_

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

## Open decisions

_(none)_

## Open gaps

| ID | Title | Discovered in |
|----|-------|---------------|
| G-022 | Provenance model extension surface |  |
| G-023 | Delegated \`--force\` via \`aiwf authorize --allow-force\` |  |

## Warnings

_(none)_

## Recent activity

| Date | Actor | Verb | Detail |
|------|-------|------|--------|
| 2026-05-06 | human/peter | promote | aiwf promote G-049 addressed [audit-only] |
| 2026-05-06 | human/peter | promote | aiwf promote G-048 open -> addressed |
| 2026-05-06 | human/peter | promote | aiwf promote G-050 open -> addressed |
| 2026-05-06 | human/peter | add | aiwf add gap G-050 'Pre-commit hook aborts when STATUS.md is gitignored — violates 'tolerant by design' contract, orphans .git/index.lock' |
| 2026-05-06 | human/peter | promote | aiwf promote G-049 addressed [audit-only] |

