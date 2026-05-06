# aiwf status — 2026-05-06

_126 entities · 0 errors · 0 warnings_

## In flight

### E-15 — Reduce planning-verb commit cardinality _(active)_

- ✓ **M-056** — Add --body-file to aiwf add variants _(done)_ — ACs 5/5 met
- → **M-057** — Batched --title on aiwf add ac _(in_progress)_ — ACs 2/5 met (3 open)
- **M-058** — Add aiwf edit-body verb and reconcile skill _(draft)_ — ACs 0/4 met (4 open)
- ✓ **M-059** — Add resolver-pointer flags to status-transition verbs _(done)_ — ACs 4/4 met

```mermaid
flowchart LR
  E_15["E-15<br/>Reduce planning-verb commit cardinality"]:::epic_active
  M_056["M-056 (5/5)<br/>Add --body-file to aiwf add variants"]:::ms_done
  E_15 --> M_056
  M_057["M-057 (2/5)<br/>Batched --title on aiwf add ac"]:::ms_in_progress
  E_15 --> M_057
  M_058["M-058 (0/4)<br/>Add aiwf edit-body verb and reconcile skill"]:::ms_draft
  E_15 --> M_058
  M_059["M-059 (4/4)<br/>Add resolver-pointer flags to status-transition verbs"]:::ms_done
  E_15 --> M_059
  classDef epic_active fill:#d6eaff,stroke:#1a73e8,color:#000
  classDef epic_proposed fill:#f4f4f4,stroke:#888,color:#000
  classDef ms_done fill:#d8f5d8,stroke:#2a8a2a,color:#000
  classDef ms_in_progress fill:#fff3c4,stroke:#caa400,color:#000
  classDef ms_draft fill:#f4f4f4,stroke:#888,color:#000
  classDef ms_cancelled fill:#fbeaea,stroke:#c33,color:#000
```

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

### E-14 — Cobra and completion _(proposed)_

- **M-049** — Bootstrap Cobra and migrate version _(draft)_ — ACs 0/6 met (6 open)
- **M-050** — Migrate read-only verbs _(draft)_ — ACs 0/4 met (4 open)
- **M-051** — Migrate mutating verbs _(draft)_ — ACs 0/6 met (6 open)
- **M-052** — Migrate setup verbs _(draft)_ — ACs 0/4 met (4 open)
- **M-053** — Completion verb and static completion _(draft)_ — ACs 0/5 met (5 open)
- **M-054** — Dynamic id completion and drift test _(draft)_ — ACs 0/4 met (4 open)
- **M-055** — Documentation pass _(draft)_ — ACs 0/4 met (4 open)

```mermaid
flowchart LR
  E_14["E-14<br/>Cobra and completion"]:::epic_proposed
  M_049["M-049 (0/6)<br/>Bootstrap Cobra and migrate version"]:::ms_draft
  E_14 --> M_049
  M_050["M-050 (0/4)<br/>Migrate read-only verbs"]:::ms_draft
  E_14 --> M_050
  M_051["M-051 (0/6)<br/>Migrate mutating verbs"]:::ms_draft
  E_14 --> M_051
  M_052["M-052 (0/4)<br/>Migrate setup verbs"]:::ms_draft
  E_14 --> M_052
  M_053["M-053 (0/5)<br/>Completion verb and static completion"]:::ms_draft
  E_14 --> M_053
  M_054["M-054 (0/4)<br/>Dynamic id completion and drift test"]:::ms_draft
  E_14 --> M_054
  M_055["M-055 (0/4)<br/>Documentation pass"]:::ms_draft
  E_14 --> M_055
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
| G-052 | Plain-git body edits trigger warnings despite skill permitting them | E-14 |

## Warnings

_(none)_

## Recent activity

| Date | Actor | Verb | Detail |
|------|-------|------|--------|
| 2026-05-06 | human/peter | promote | aiwf promote M-057/AC-1 open -> met |
| 2026-05-06 | human/peter | promote | aiwf promote M-057 draft -> in_progress |
| 2026-05-06 | human/peter | promote | aiwf promote G-051 open -> addressed |
| 2026-05-06 | human/peter | promote | aiwf promote M-056 in_progress -> done |
| 2026-05-06 | human/peter | promote | aiwf promote M-056/AC-5 open -> met |

