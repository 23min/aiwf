# aiwf status — 2026-05-11

_249 entities · 0 errors · 3 warnings · run `aiwf check` for details_

## In flight

_(no active epics)_

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

### E-0029 — Glanceable governance HTML render: layout, sidebar, chips (closes G-0114) _(proposed)_

_(no milestones)_

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
| G-0059 | Branch model: no canonical entity-hierarchy-to-git-branches mapping | M-0069 |
| G-0060 | Patch ritual is loosely defined; no kernel-level rules for shape, scope, branch, or audit trail |  |
| G-0067 | wf-tdd-cycle is LLM-honor-system advisory; no mechanical RED-first guard | M-0066 |
| G-0068 | Discoverability policy misses dynamic finding subcodes | M-0066 |
| G-0069 | aiwf init's plugins nudge hardcodes user-scope CLI install form | M-0070 |
| G-0070 | aiwf doctor lacks --format=json envelope | M-0070 |
| G-0073 | depends_on restricted to milestone→milestone; cross-kind blocking via body prose | E-0021 |
| G-0074 | docs/pocv3/ body prose still uses PoC framing; needs sweep |  |
| G-0075 | docs/pocv3/ directory naming is now historical; rename or accept |  |
| G-0076 | CONTRIBUTING.md describes PR-based workflow at odds with trunk-based model on main |  |
| G-0077 | Post-promotion working paper (aiwf's thesis) not yet written |  |
| G-0078 | No priority field on entities; backlog isn't filterable or sortable by importance |  |
| G-0079 | aiwfx-plan-milestones plugin skill needs --depends-on documentation | M-0076 |
| G-0080 | Wide-table verbs wrap mid-row; no TTY-aware sizing or truncation | M-0076 |
| G-0082 | Planning closure should default-merge to main before implementation begins | E-0021 |
| G-0083 | aiwf retitle does not sync entity body H1 with frontmatter title | E-0021 |
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

## Warnings

| Code | Entity | Path | Message |
|------|--------|------|---------|
| entity-body-empty | E-0029 | work/epics/E-0029-glanceable-governance-html-render-layout-sidebar-chips-closes-g-0114/epic.md | E-0029 body section \`## Goal\` is empty |
| entity-body-empty | E-0029 | work/epics/E-0029-glanceable-governance-html-render-layout-sidebar-chips-closes-g-0114/epic.md | E-0029 body section \`## Scope\` is empty |
| entity-body-empty | E-0029 | work/epics/E-0029-glanceable-governance-html-render-layout-sidebar-chips-closes-g-0114/epic.md | E-0029 body section \`## Out of scope\` is empty |

## Recent activity

| Date | Actor | Verb | Detail |
|------|-------|------|--------|
| 2026-05-11 | human/peter | archive | aiwf archive: sweep 6 entities into archive/ (1 epic, 1 adr, 4 gap) |
| 2026-05-11 | human/peter | promote | aiwf promote G-0081 open -> addressed |
| 2026-05-11 | human/peter | edit-body | aiwf edit-body G-0081 |
| 2026-05-11 | human/peter | promote | aiwf promote G-0084 open -> wontfix |
| 2026-05-11 | human/peter | edit-body | aiwf edit-body G-0084 |

