# aiwf status — 2026-05-06

_118 entities · 0 errors · 0 warnings_

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

### E-14 — Cobra and completion _(proposed)_

- **M-049** — Bootstrap Cobra and migrate version _(draft)_ — ACs 0/5 met (5 open)
- **M-050** — Migrate read-only verbs _(draft)_
- **M-051** — Migrate mutating verbs _(draft)_
- **M-052** — Migrate setup verbs _(draft)_
- **M-053** — Completion verb and static completion _(draft)_
- **M-054** — Dynamic id completion and drift test _(draft)_
- **M-055** — Documentation pass _(draft)_

```mermaid
flowchart LR
  E_14["E-14<br/>Cobra and completion"]:::epic_proposed
  M_049["M-049 (0/5)<br/>Bootstrap Cobra and migrate version"]:::ms_draft
  E_14 --> M_049
  M_050["M-050<br/>Migrate read-only verbs"]:::ms_draft
  E_14 --> M_050
  M_051["M-051<br/>Migrate mutating verbs"]:::ms_draft
  E_14 --> M_051
  M_052["M-052<br/>Migrate setup verbs"]:::ms_draft
  E_14 --> M_052
  M_053["M-053<br/>Completion verb and static completion"]:::ms_draft
  E_14 --> M_053
  M_054["M-054<br/>Dynamic id completion and drift test"]:::ms_draft
  E_14 --> M_054
  M_055["M-055<br/>Documentation pass"]:::ms_draft
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

## Warnings

_(none)_

## Recent activity

| Date | Actor | Verb | Detail |
|------|-------|------|--------|
| 2026-05-06 | human/peter | add | aiwf add ac M-049/AC-4 'Exit codes 0/1/2/3 preserved end-to-end through Cobra dispatch' |
| 2026-05-06 | human/peter | add | aiwf add ac M-049/AC-3 'version verb migrated; --format=json envelope shape preserved byte-exact' |
| 2026-05-06 | human/peter | add | aiwf add ac M-049/AC-2 'Cobra root command and subcommand routing structure in cmd/aiwf' |
| 2026-05-06 | human/peter | add | aiwf add ac M-049/AC-1 'Cobra dependency added to go.mod with one-line justification in commit message' |
| 2026-05-06 | human/peter | add | aiwf add milestone M-055 'Documentation pass' |

