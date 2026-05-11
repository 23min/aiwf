# aiwf status — 2026-05-11

_245 entities · 0 errors · 7 warnings · run `aiwf check` for details_

> Sweep pending: 6 terminal entities not yet archived (run `aiwf archive --dry-run` to preview)

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

## Open decisions

| ID | Kind | Title | Status |
|----|------|-------|--------|
| ADR-0001 | adr | Mint entity ids at trunk integration via per-kind inbox state | proposed |
| ADR-0005 | adr | Verb hygiene contract: complete, consistent, pre-flighted aiwf verbs | proposed |
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
| G-0081 | aiwf rename does not pre-flight trunk-collision check | E-0021 |
| G-0082 | Planning closure should default-merge to main before implementation begins | E-0021 |
| G-0083 | aiwf retitle does not sync entity body H1 with frontmatter title | E-0021 |
| G-0084 | Verb hygiene contract is undocumented; G-0081/G-0082/G-0083 lack umbrella | E-0021 |
| G-0087 | No aiwf-show embedded skill | M-0074 |
| G-0088 | Skill-coverage policy doesn't police plugin skills under aiwf-extensions/ | M-0079 |
| G-0090 | M-0079 AC-8 drift-check has untested branches; refactor for hermetic tests | M-0079 |
| G-0091 | No preventive check for body-prose path-form refs to archive-moved entities |  |
| G-0092 | No documented hierarchy of doc authority across docs/ |  |
| G-0097 | Test-suite wall time dominated by serial execution and per-test setup |  |
| G-0099 | Worktree isolation must be a parent-side precondition, not an Agent kwarg honor |  |
| G-0103 | absolute-path leak lint | M-0089 |
| G-0104 | Test-parallelism discipline: ship to consumers via wf-rituals or BYO? | E-0025 |
| G-0107 | reorganize cmd/aiwf into idiomatic per-verb packages |  |
| G-0110 | gremlins --diff <ref> filter excludes new files entirely; manual mutation review needed for M-0094/95/96 | M-0097 |
| G-0111 | Wrap-side ritual: scope ends before done, human-only on done, wrap-epic update | M-0096 |

## Warnings

| Code | Entity | Path | Message |
|------|--------|------|---------|
| terminal-entity-not-archived | M-0094 | work/epics/E-0028-start-epic-ritual-sovereign-activation-with-preflight-branch-worktree-choice-and-optional-delegation-closes-g-0063-start-side/M-0094-add-aiwf-check-finding-epic-active-no-drafted-milestones.md | entity M-0094 has terminal status 'done' but file is still in the active tree; awaiting \`aiwf archive --apply\` sweep |
| terminal-entity-not-archived | M-0095 | work/epics/E-0028-start-epic-ritual-sovereign-activation-with-preflight-branch-worktree-choice-and-optional-delegation-closes-g-0063-start-side/M-0095-enforce-human-only-actor-on-aiwf-promote-e-nn-active.md | entity M-0095 has terminal status 'done' but file is still in the active tree; awaiting \`aiwf archive --apply\` sweep |
| terminal-entity-not-archived | M-0096 | work/epics/E-0028-start-epic-ritual-sovereign-activation-with-preflight-branch-worktree-choice-and-optional-delegation-closes-g-0063-start-side/M-0096-ship-aiwfx-start-epic-skill-with-worktree-and-branch-preflight-prompts.md | entity M-0096 has terminal status 'done' but file is still in the active tree; awaiting \`aiwf archive --apply\` sweep |
| terminal-entity-not-archived | M-0097 | work/epics/E-0028-start-epic-ritual-sovereign-activation-with-preflight-branch-worktree-choice-and-optional-delegation-closes-g-0063-start-side/M-0097-close-m-0094-95-96-verification-seams-m-0095-automation-audit-chokepoint-and-ac-5-drift-comparator.md | entity M-0097 has terminal status 'done' but file is still in the active tree; awaiting \`aiwf archive --apply\` sweep |
| terminal-entity-not-archived | E-0028 | work/epics/E-0028-start-epic-ritual-sovereign-activation-with-preflight-branch-worktree-choice-and-optional-delegation-closes-g-0063-start-side/epic.md | entity E-0028 has terminal status 'done' but file is still in the active tree; awaiting \`aiwf archive --apply\` sweep |
| terminal-entity-not-archived | G-0063 | work/gaps/G-0063-no-start-epic-ritual.md | entity G-0063 has terminal status 'addressed' but file is still in the active tree; awaiting \`aiwf archive --apply\` sweep |

## Recent activity

| Date | Actor | Verb | Detail |
|------|-------|------|--------|
| 2026-05-11 | human/peter | wrap-epic | chore(E-0028): wrap epic — start-epic ritual + verification seams closed |
| 2026-05-11 | human/peter | wrap-epic | chore(epic): wrap E-0028 — start-epic ritual: sovereign activation with preflight + delegation |
| 2026-05-11 | human/peter | archive | aiwf archive: sweep 1 entity into archive/ (1 gap) |
| 2026-05-11 | human/peter | promote | aiwf promote G-0108 open -> addressed |
| 2026-05-11 | human/peter | render-roadmap | aiwf render roadmap |

