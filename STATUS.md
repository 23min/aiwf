# aiwf status — 2026-05-07

_143 entities · 0 errors · 0 warnings_

## In flight

### E-14 — Cobra and completion _(active)_

- ✓ **M-049** — Bootstrap Cobra and migrate version _(done)_ — ACs 6/6 met
- ✓ **M-050** — Migrate read-only verbs _(done)_ — ACs 4/4 met
- ✓ **M-051** — Migrate mutating verbs _(done)_ — ACs 6/6 met
- ✓ **M-052** — Migrate setup verbs _(done)_ — ACs 4/4 met
- ✓ **M-053** — Completion verb and static completion _(done)_ — ACs 5/5 met
- ✓ **M-054** — Dynamic id completion and drift test _(done)_ — ACs 4/4 met
- ✓ **M-055** — Documentation pass _(done)_ — ACs 4/4 met
- ✓ **M-061** — Contract family migration + changelog retrofill + help-recursion test _(done)_ — ACs 5/5 met
- **M-069** — Retrofit TDD-shaped tests for E-14 _(draft)_ — ACs 0/7 met (7 open) — tdd: required

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
  M_061["M-061 (5/5)<br/>Contract family migration + changelog retrofill + help-recursion test"]:::ms_done
  E_14 --> M_061
  M_069["M-069 (0/7)<br/>Retrofit TDD-shaped tests for E-14"]:::ms_draft
  E_14 --> M_069
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

### E-17 — AC body prose chokepoint (closes G-058) _(proposed)_

- **M-066** — aiwf check finding acs-body-empty _(draft)_ — ACs 0/6 met (6 open) — tdd: required
- **M-067** — aiwf add ac --body-file flag for in-verb body scaffolding _(draft)_ — ACs 0/8 met (8 open) — tdd: required
- **M-068** — aiwf-add skill names fill-in-body as required next step _(draft)_ — ACs 0/5 met (5 open) — tdd: required

```mermaid
flowchart LR
  E_17["E-17<br/>AC body prose chokepoint (closes G-058)"]:::epic_proposed
  M_066["M-066 (0/6)<br/>aiwf check finding acs-body-empty"]:::ms_draft
  E_17 --> M_066
  M_067["M-067 (0/8)<br/>aiwf add ac --body-file flag for in-verb body scaffolding"]:::ms_draft
  E_17 --> M_067
  M_068["M-068 (0/5)<br/>aiwf-add skill names fill-in-body as required next step"]:::ms_draft
  E_17 --> M_068
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
| G-056 | aiwf render output (site/) is not gitignored; pollutes consumer working tree | E-14 |
| G-057 | Stray aiwf binary in repo root from local builds is not gitignored |  |
| G-058 | AC body sections ship empty; no chokepoint enforces prose intent | E-16 |

## Warnings

_(none)_

## Recent activity

| Date | Actor | Verb | Detail |
|------|-------|------|--------|
| 2026-05-07 | human/peter | promote | aiwf promote M-069/AC-6 --phase red -> green |
| 2026-05-07 | human/peter | edit-body | aiwf edit-body M-069 |
| 2026-05-07 | human/peter | promote | aiwf promote M-069/AC-5 --phase green -> done |
| 2026-05-07 | human/peter | promote | aiwf promote M-069/AC-5 --phase red -> green |
| 2026-05-07 | human/peter | edit-body | aiwf edit-body M-069 |

