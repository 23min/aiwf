# aiwf status — 2026-05-07

_129 entities · 0 errors · 0 warnings_

## In flight

### E-14 — Cobra and completion _(active)_

- ✓ **M-049** — Bootstrap Cobra and migrate version _(done)_ — ACs 6/6 met
- ✓ **M-050** — Migrate read-only verbs _(done)_ — ACs 4/4 met
- ✓ **M-051** — Migrate mutating verbs _(done)_ — ACs 6/6 met
- ✓ **M-052** — Migrate setup verbs _(done)_ — ACs 4/4 met
- ✓ **M-053** — Completion verb and static completion _(done)_ — ACs 5/5 met
- ✓ **M-054** — Dynamic id completion and drift test _(done)_ — ACs 4/4 met
- ✓ **M-055** — Documentation pass _(done)_ — ACs 4/4 met
- → **M-061** — Contract family migration + changelog retrofill + help-recursion test _(in_progress)_ — ACs 4/5 met (1 open)

```mermaid
flowchart LR
  E_14["E-14<br/>Cobra and completion"]:::epic_active
  M_049["M-049 (6/6)<br/>Bootstrap Cobra and migrate version"]:::ms_done
  E_14 --> M_049
  M_050["M-050 (4/4)<br/>Migrate read-only verbs"]:::ms_done
  E_14 --> M_050
  M_051["M-051 (6/6)<br/>Migrate mutating verbs"]:::ms_done
  E_14 --> M_051
  M_052["M-052 (4/4)<br/>Migrate setup verbs"]:::ms_done
  E_14 --> M_052
  M_053["M-053 (5/5)<br/>Completion verb and static completion"]:::ms_done
  E_14 --> M_053
  M_054["M-054 (4/4)<br/>Dynamic id completion and drift test"]:::ms_done
  E_14 --> M_054
  M_055["M-055 (4/4)<br/>Documentation pass"]:::ms_done
  E_14 --> M_055
  M_061["M-061 (4/5)<br/>Contract family migration + changelog retrofill + help-recursion test"]:::ms_in_progress
  E_14 --> M_061
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
| 2026-05-07 | human/peter | promote | aiwf promote M-061/AC-3 open -> met |
| 2026-05-07 | human/peter | promote | aiwf promote M-061/AC-2 open -> met |
| 2026-05-07 | human/peter | promote | aiwf promote M-061/AC-1 open -> met |
| 2026-05-07 | human/peter | promote | aiwf promote M-061 draft -> in_progress |
| 2026-05-07 | human/peter | add | aiwf add ac M-061 AC-1..AC-5 (5 criteria) |

