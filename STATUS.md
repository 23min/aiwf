# aiwf status — 2026-05-11

_259 entities · 0 errors · 6 warnings · run `aiwf check` for details_

> Sweep pending: 1 terminal entity not yet archived (run `aiwf archive --dry-run` to preview)

## In flight

### E-0029 — Glanceable governance HTML render: layout, sidebar, chips (closes G-0114) _(active)_

- **M-0098** — Render-site layout overhaul: viewport-fill body, flush-left sidebar, prose cap _(draft)_ — tdd: required
- **M-0099** — Kind-index chip filter: single emitted file per kind with :target chips _(draft)_ — tdd: required
- **M-0100** — Sidebar surfaces gaps with active count _(draft)_ — tdd: required
- **M-0101** — In-page status hierarchy in gaps.html _(draft)_ — tdd: required

```mermaid
flowchart LR
  E_0029["E-0029<br/>Glanceable governance HTML render: layout, sidebar, chips (closes G-0114)"]:::epic_active
  M_0098["M-0098<br/>Render-site layout overhaul: viewport-fill body, flush-left sidebar, prose cap"]:::ms_draft
  E_0029 --> M_0098
  M_0099["M-0099<br/>Kind-index chip filter: single emitted file per kind with :target chips"]:::ms_draft
  E_0029 --> M_0099
  M_0100["M-0100<br/>Sidebar surfaces gaps with active count"]:::ms_draft
  E_0029 --> M_0100
  M_0101["M-0101<br/>In-page status hierarchy in gaps.html"]:::ms_draft
  E_0029 --> M_0101
  classDef epic_active fill:#d6eaff,stroke:#1a73e8,color:#000
  classDef epic_proposed fill:#f4f4f4,stroke:#888,color:#000
  classDef ms_done fill:#d8f5d8,stroke:#2a8a2a,color:#000
  classDef ms_in_progress fill:#fff3c4,stroke:#caa400,color:#000
  classDef ms_draft fill:#f4f4f4,stroke:#888,color:#000
  classDef ms_cancelled fill:#fbeaea,stroke:#c33,color:#000
```

## Roadmap

### E-0016 — TDD policy declaration chokepoint (closes G-0055) _(proposed)_

- **M-0062** — tdd flag on aiwf add milestone with project-default fallback _(draft)_ — ACs 0/8 met (8 open) — tdd: required
- **M-0063** — aiwf.yaml tdd.default schema and aiwf init seeding _(draft)_ — ACs 0/7 met (7 open) — tdd: required
- **M-0064** — aiwf update migration for existing aiwf.yaml with loud output _(draft)_ — ACs 0/8 met (8 open) — tdd: required
- **M-0065** — aiwf check finding milestone-tdd-undeclared as defense-in-depth _(draft)_ — ACs 0/5 met (5 open) — tdd: required

```mermaid
flowchart LR
  E_0016["E-0016<br/>TDD policy declaration chokepoint (closes G-0055)"]:::epic_proposed
  M_0062["M-0062 (0/8)<br/>tdd flag on aiwf add milestone with project-default fallback"]:::ms_draft
  E_0016 --> M_0062
  M_0063["M-0063 (0/7)<br/>aiwf.yaml tdd.default schema and aiwf init seeding"]:::ms_draft
  E_0016 --> M_0063
  M_0064["M-0064 (0/8)<br/>aiwf update migration for existing aiwf.yaml with loud output"]:::ms_draft
  E_0016 --> M_0064
  M_0065["M-0065 (0/5)<br/>aiwf check finding milestone-tdd-undeclared as defense-in-depth"]:::ms_draft
  E_0016 --> M_0065
  classDef epic_active fill:#d6eaff,stroke:#1a73e8,color:#000
  classDef epic_proposed fill:#f4f4f4,stroke:#888,color:#000
  classDef ms_done fill:#d8f5d8,stroke:#2a8a2a,color:#000
  classDef ms_in_progress fill:#fff3c4,stroke:#caa400,color:#000
  classDef ms_draft fill:#f4f4f4,stroke:#888,color:#000
  classDef ms_cancelled fill:#fbeaea,stroke:#c33,color:#000
```

### E-0019 — Parallel TDD subagents with finding-gated AC closure _(proposed)_

_(no milestones)_

### E-0025 — Test-suite parallelism and fixture-sharing pass — closes G-0097 _(proposed)_

- **M-0091** — Roll out TestMain + t.Parallel across internal/* test packages _(draft)_ — ACs 0/6 met (6 open) — tdd: none
- **M-0092** — Roll out TestMain + t.Parallel + no-ldflags dedup to cmd/aiwf/ _(draft)_ — ACs 0/4 met (4 open) — tdd: none
- **M-0093** — Document test-discipline convention and lock its chokepoint _(draft)_ — ACs 0/3 met (3 open) — tdd: none

```mermaid
flowchart LR
  E_0025["E-0025<br/>Test-suite parallelism and fixture-sharing pass — closes G-0097"]:::epic_proposed
  M_0091["M-0091 (0/6)<br/>Roll out TestMain + t.Parallel across internal/* test packages"]:::ms_draft
  E_0025 --> M_0091
  M_0092["M-0092 (0/4)<br/>Roll out TestMain + t.Parallel + no-ldflags dedup to cmd/aiwf/"]:::ms_draft
  E_0025 --> M_0092
  M_0093["M-0093 (0/3)<br/>Document test-discipline convention and lock its chokepoint"]:::ms_draft
  E_0025 --> M_0093
  classDef epic_active fill:#d6eaff,stroke:#1a73e8,color:#000
  classDef epic_proposed fill:#f4f4f4,stroke:#888,color:#000
  classDef ms_done fill:#d8f5d8,stroke:#2a8a2a,color:#000
  classDef ms_in_progress fill:#fff3c4,stroke:#caa400,color:#000
  classDef ms_draft fill:#f4f4f4,stroke:#888,color:#000
  classDef ms_cancelled fill:#fbeaea,stroke:#c33,color:#000
```

### E-0030 — Branch model chokepoint: --branch flag, sequencing, isolation-escape finding _(proposed)_

- **M-0102** — aiwf authorize --branch flag + scope-branch trailer coupling _(draft)_ — tdd: required
- **M-0103** — AI-side preflight: aiwf authorize refuses without ritual branch context _(draft)_ — tdd: required

```mermaid
flowchart LR
  E_0030["E-0030<br/>Branch model chokepoint: --branch flag, sequencing, isolation-escape finding"]:::epic_proposed
  M_0102["M-0102<br/>aiwf authorize --branch flag + scope-branch trailer coupling"]:::ms_draft
  E_0030 --> M_0102
  M_0103["M-0103<br/>AI-side preflight: aiwf authorize refuses without ritual branch context"]:::ms_draft
  E_0030 --> M_0103
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
| ADR-0009 | adr | Orchestration substrate: substrate-vs-driver split with trailer-only events | proposed |

## Open gaps

| ID | Title | Discovered in |
|----|-------|---------------|
| G-0022 | Provenance model extension surface |  |
| G-0023 | Delegated \`--force\` via \`aiwf authorize --allow-force\` |  |
| G-0060 | Patch ritual is loosely defined; no kernel-level rules for shape, scope, branch, or audit trail |  |
| G-0067 | wf-tdd-cycle is LLM-honor-system advisory; no mechanical RED-first guard | M-0066 |
| G-0068 | Discoverability policy misses dynamic finding subcodes | M-0066 |
| G-0070 | aiwf doctor lacks --format=json envelope | M-0070 |
| G-0073 | depends_on restricted to milestone→milestone; cross-kind blocking via body prose | E-0021 |
| G-0074 | docs/pocv3/ body prose still uses PoC framing; needs sweep |  |
| G-0075 | docs/pocv3/ directory naming is now historical; rename or accept |  |
| G-0077 | Post-promotion working paper (aiwf's thesis) not yet written |  |
| G-0078 | No priority field on entities; backlog isn't filterable or sortable by importance |  |
| G-0080 | Wide-table verbs wrap mid-row; no TTY-aware sizing or truncation | M-0076 |
| G-0087 | No aiwf-show embedded skill | M-0074 |
| G-0088 | Skill-coverage policy doesn't police plugin skills under aiwf-extensions/ | M-0079 |
| G-0090 | M-0079 AC-8 drift-check has untested branches; refactor for hermetic tests | M-0079 |
| G-0092 | No documented hierarchy of doc authority across docs/ |  |
| G-0097 | Test-suite wall time dominated by serial execution and per-test setup |  |
| G-0099 | Worktree isolation must be a parent-side precondition, not an Agent kwarg honor |  |
| G-0103 | absolute-path leak lint | M-0089 |
| G-0104 | Test-parallelism discipline: ship to consumers via wf-rituals or BYO? | E-0025 |
| G-0107 | reorganize cmd/aiwf into idiomatic per-verb packages |  |
| G-0110 | gremlins --diff <ref> filter excludes new files entirely; manual mutation review needed for M-0094/95/96 | M-0097 |
| G-0111 | Wrap-side ritual: scope ends before done, human-only on done, wrap-epic update | M-0096 |
| G-0112 | STATUS.md pre-commit regen produces merge conflicts on a derived artifact |  |
| G-0113 | rendered HTML site has no publish path; only viewable via local aiwf render |  |
| G-0114 | HTML render gap surface: status and archive state not glanceable from sidebar |  |
| G-0115 | aiwf render roadmap --write rewrites entity refs in epic prose to broken paths |  |
| G-0116 | aiwfx-start-epic creates worktree before promote/authorize on trunk-based repos | E-0029 |

## Warnings

| Code | Entity | Path | Message |
|------|--------|------|---------|
| entity-body-empty | M-0102 | work/epics/E-0030-branch-model-chokepoint-branch-flag-sequencing-isolation-escape-finding/M-0102-aiwf-authorize-branch-flag-scope-branch-trailer-coupling.md | M-0102 body section \`## Goal\` is empty |
| entity-body-empty | M-0102 | work/epics/E-0030-branch-model-chokepoint-branch-flag-sequencing-isolation-escape-finding/M-0102-aiwf-authorize-branch-flag-scope-branch-trailer-coupling.md | M-0102 body section \`## Acceptance criteria\` is empty |
| entity-body-empty | M-0103 | work/epics/E-0030-branch-model-chokepoint-branch-flag-sequencing-isolation-escape-finding/M-0103-ai-side-preflight-aiwf-authorize-refuses-without-ritual-branch-context.md | M-0103 body section \`## Goal\` is empty |
| entity-body-empty | M-0103 | work/epics/E-0030-branch-model-chokepoint-branch-flag-sequencing-isolation-escape-finding/M-0103-ai-side-preflight-aiwf-authorize-refuses-without-ritual-branch-context.md | M-0103 body section \`## Acceptance criteria\` is empty |
| terminal-entity-not-archived | G-0059 | work/gaps/G-0059-branch-model-no-canonical-hierarchy-mapping.md | entity G-0059 has terminal status 'addressed' but file is still in the active tree; awaiting \`aiwf archive --apply\` sweep |

## Recent activity

| Date | Actor | Verb | Detail |
|------|-------|------|--------|
| 2026-05-12 | human/peter | add | aiwf add milestone M-0102 'aiwf authorize --branch flag + scope-branch trailer coupling' |
| 2026-05-12 | human/peter | edit-body | aiwf edit-body E-0030 |
| 2026-05-12 | human/peter | add | aiwf add epic E-0030 'Branch model chokepoint: --branch flag, sequencing, isolation-escape finding' |
| 2026-05-12 | human/peter | promote | aiwf promote G-0059 open -> addressed |
| 2026-05-12 | human/peter | promote | aiwf promote ADR-0010 proposed -> accepted |

