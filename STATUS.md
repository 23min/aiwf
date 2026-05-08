# aiwf status — 2026-05-08

_174 entities · 0 errors · 29 warnings · run `aiwf check` for details_

## In flight

_(no active epics)_

## Roadmap

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

### E-19 — Parallel TDD subagents with finding-gated AC closure _(proposed)_

_(no milestones)_

### E-20 — Add list verb (closes G-061) _(proposed)_

- **M-072** — aiwf list verb, status filter-helper refactor, contract-skill drift fix _(draft)_ — ACs 0/9 met (9 open) — tdd: required
- **M-073** — aiwf-list skill, aiwf-status skill tightening _(draft)_ — ACs 0/5 met (5 open) — tdd: advisory
- **M-074** — skill-coverage policy, judgment ADR, CLAUDE.md skills section, G-061 closure _(draft)_ — ACs 0/10 met (10 open) — tdd: required

```mermaid
flowchart LR
  E_20["E-20<br/>Add list verb (closes G-061)"]:::epic_proposed
  M_072["M-072 (0/9)<br/>aiwf list verb, status filter-helper refactor, contract-skill drift fix"]:::ms_draft
  E_20 --> M_072
  M_073["M-073 (0/5)<br/>aiwf-list skill, aiwf-status skill tightening"]:::ms_draft
  E_20 --> M_073
  M_074["M-074 (0/10)<br/>skill-coverage policy, judgment ADR, CLAUDE.md skills section, G-061 closure"]:::ms_draft
  E_20 --> M_074
  classDef epic_active fill:#d6eaff,stroke:#1a73e8,color:#000
  classDef epic_proposed fill:#f4f4f4,stroke:#888,color:#000
  classDef ms_done fill:#d8f5d8,stroke:#2a8a2a,color:#000
  classDef ms_in_progress fill:#fff3c4,stroke:#caa400,color:#000
  classDef ms_draft fill:#f4f4f4,stroke:#888,color:#000
  classDef ms_cancelled fill:#fbeaea,stroke:#c33,color:#000
```

### E-21 — Open-work synthesis: recommended-sequence skill (replaces critical-path.md) _(proposed)_

_(no milestones)_

### E-22 — Planning toolchain fixes (closes G-071, G-072, G-065) _(proposed)_

- **M-075** — Lifecycle-gate entity-body-empty rule (closes G-071) _(draft)_ — tdd: required

```mermaid
flowchart LR
  E_22["E-22<br/>Planning toolchain fixes (closes G-071, G-072, G-065)"]:::epic_proposed
  M_075["M-075<br/>Lifecycle-gate entity-body-empty rule (closes G-071)"]:::ms_draft
  E_22 --> M_075
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
| ADR-0004 | adr | Uniform archive convention for terminal-status entities | proposed |

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
| G-063 | No defined start-epic ritual: epic activation is a deliberate sovereign act with preflight + optional delegation, but kernel treats it as a one-line FSM flip |  |
| G-065 | No aiwf retitle verb: scope refactors that change an entity's or AC's intent leave frontmatter title fields permanently misleading; only slug rename is supported |  |
| G-067 | wf-tdd-cycle is LLM-honor-system advisory; under load the LLM bypasses RED-first and the branch-coverage HARD RULE without anything mechanical catching it (M-066/AC-1 cycle wrote ~165 lines of impl before any test existed) | M-066 |
| G-068 | Discoverability policy misses dynamic finding subcodes | M-066 |
| G-069 | aiwf init's printRitualsSuggestion hardcodes the CLI install form, which defaults to user scope and won't satisfy doctor.recommended_plugins; nudge silently steers fresh operators away from project-scope outcome | M-070 |
| G-070 | aiwf doctor has no --format=json envelope; M-070's recommended-plugin-not-installed finding-code surfaces only as human text. Add JSON envelope when a JSON-consuming caller appears | M-070 |
| G-071 | entity-body-empty/ac fires on freshly-allocated ACs in draft milestones; conflicts with plan-milestones 'shape now, detail later' discipline | E-20 |
| G-072 | milestone depends_on has six kernel read sites and zero writer verbs; populating it requires a hand-edit aiwf edit-body refuses, and neither aiwf-add nor aiwfx-plan-milestones tells the full story | E-20 |
| G-073 | depends_on is restricted to milestone→milestone edges; cross-kind blocking lives in body prose only; subsumes G-072 in scope | E-21 |

## Warnings

| Code | Entity | Path | Message |
|------|--------|------|---------|
| entity-body-empty | ADR-0002 | docs/adr/ADR-0002-test-dry-run-delete-me.md | ADR-0002 body section \`## Context\` is empty |
| entity-body-empty | ADR-0002 | docs/adr/ADR-0002-test-dry-run-delete-me.md | ADR-0002 body section \`## Decision\` is empty |
| entity-body-empty | ADR-0002 | docs/adr/ADR-0002-test-dry-run-delete-me.md | ADR-0002 body section \`## Consequences\` is empty |
| entity-body-empty | M-072/AC-1 | work/epics/E-20-add-list-verb-closes-g-061/M-072-aiwf-list-verb-status-filter-helper-refactor-contract-skill-drift-fix.md | M-072/AC-1 body under \`### AC-1\` is empty |
| entity-body-empty | M-072/AC-2 | work/epics/E-20-add-list-verb-closes-g-061/M-072-aiwf-list-verb-status-filter-helper-refactor-contract-skill-drift-fix.md | M-072/AC-2 body under \`### AC-2\` is empty |
| entity-body-empty | M-072/AC-3 | work/epics/E-20-add-list-verb-closes-g-061/M-072-aiwf-list-verb-status-filter-helper-refactor-contract-skill-drift-fix.md | M-072/AC-3 body under \`### AC-3\` is empty |
| entity-body-empty | M-072/AC-4 | work/epics/E-20-add-list-verb-closes-g-061/M-072-aiwf-list-verb-status-filter-helper-refactor-contract-skill-drift-fix.md | M-072/AC-4 body under \`### AC-4\` is empty |
| entity-body-empty | M-072/AC-5 | work/epics/E-20-add-list-verb-closes-g-061/M-072-aiwf-list-verb-status-filter-helper-refactor-contract-skill-drift-fix.md | M-072/AC-5 body under \`### AC-5\` is empty |
| entity-body-empty | M-072/AC-6 | work/epics/E-20-add-list-verb-closes-g-061/M-072-aiwf-list-verb-status-filter-helper-refactor-contract-skill-drift-fix.md | M-072/AC-6 body under \`### AC-6\` is empty |
| entity-body-empty | M-072/AC-7 | work/epics/E-20-add-list-verb-closes-g-061/M-072-aiwf-list-verb-status-filter-helper-refactor-contract-skill-drift-fix.md | M-072/AC-7 body under \`### AC-7\` is empty |
| entity-body-empty | M-072/AC-8 | work/epics/E-20-add-list-verb-closes-g-061/M-072-aiwf-list-verb-status-filter-helper-refactor-contract-skill-drift-fix.md | M-072/AC-8 body under \`### AC-8\` is empty |
| entity-body-empty | M-072/AC-9 | work/epics/E-20-add-list-verb-closes-g-061/M-072-aiwf-list-verb-status-filter-helper-refactor-contract-skill-drift-fix.md | M-072/AC-9 body under \`### AC-9\` is empty |
| entity-body-empty | M-073/AC-1 | work/epics/E-20-add-list-verb-closes-g-061/M-073-aiwf-list-skill-aiwf-status-skill-tightening.md | M-073/AC-1 body under \`### AC-1\` is empty |
| entity-body-empty | M-073/AC-2 | work/epics/E-20-add-list-verb-closes-g-061/M-073-aiwf-list-skill-aiwf-status-skill-tightening.md | M-073/AC-2 body under \`### AC-2\` is empty |
| entity-body-empty | M-073/AC-3 | work/epics/E-20-add-list-verb-closes-g-061/M-073-aiwf-list-skill-aiwf-status-skill-tightening.md | M-073/AC-3 body under \`### AC-3\` is empty |
| entity-body-empty | M-073/AC-4 | work/epics/E-20-add-list-verb-closes-g-061/M-073-aiwf-list-skill-aiwf-status-skill-tightening.md | M-073/AC-4 body under \`### AC-4\` is empty |
| entity-body-empty | M-073/AC-5 | work/epics/E-20-add-list-verb-closes-g-061/M-073-aiwf-list-skill-aiwf-status-skill-tightening.md | M-073/AC-5 body under \`### AC-5\` is empty |
| entity-body-empty | M-074/AC-1 | work/epics/E-20-add-list-verb-closes-g-061/M-074-skill-coverage-policy-judgment-adr-claude-md-skills-section-g-061-closure.md | M-074/AC-1 body under \`### AC-1\` is empty |
| entity-body-empty | M-074/AC-2 | work/epics/E-20-add-list-verb-closes-g-061/M-074-skill-coverage-policy-judgment-adr-claude-md-skills-section-g-061-closure.md | M-074/AC-2 body under \`### AC-2\` is empty |
| entity-body-empty | M-074/AC-3 | work/epics/E-20-add-list-verb-closes-g-061/M-074-skill-coverage-policy-judgment-adr-claude-md-skills-section-g-061-closure.md | M-074/AC-3 body under \`### AC-3\` is empty |
| entity-body-empty | M-074/AC-4 | work/epics/E-20-add-list-verb-closes-g-061/M-074-skill-coverage-policy-judgment-adr-claude-md-skills-section-g-061-closure.md | M-074/AC-4 body under \`### AC-4\` is empty |
| entity-body-empty | M-074/AC-5 | work/epics/E-20-add-list-verb-closes-g-061/M-074-skill-coverage-policy-judgment-adr-claude-md-skills-section-g-061-closure.md | M-074/AC-5 body under \`### AC-5\` is empty |
| entity-body-empty | M-074/AC-6 | work/epics/E-20-add-list-verb-closes-g-061/M-074-skill-coverage-policy-judgment-adr-claude-md-skills-section-g-061-closure.md | M-074/AC-6 body under \`### AC-6\` is empty |
| entity-body-empty | M-074/AC-7 | work/epics/E-20-add-list-verb-closes-g-061/M-074-skill-coverage-policy-judgment-adr-claude-md-skills-section-g-061-closure.md | M-074/AC-7 body under \`### AC-7\` is empty |
| entity-body-empty | M-074/AC-8 | work/epics/E-20-add-list-verb-closes-g-061/M-074-skill-coverage-policy-judgment-adr-claude-md-skills-section-g-061-closure.md | M-074/AC-8 body under \`### AC-8\` is empty |
| entity-body-empty | M-074/AC-9 | work/epics/E-20-add-list-verb-closes-g-061/M-074-skill-coverage-policy-judgment-adr-claude-md-skills-section-g-061-closure.md | M-074/AC-9 body under \`### AC-9\` is empty |
| entity-body-empty | M-074/AC-10 | work/epics/E-20-add-list-verb-closes-g-061/M-074-skill-coverage-policy-judgment-adr-claude-md-skills-section-g-061-closure.md | M-074/AC-10 body under \`### AC-10\` is empty |
| entity-body-empty | M-075 | work/epics/E-22-planning-toolchain-fixes-closes-g-071-g-072-g-065/M-075-lifecycle-gate-entity-body-empty-rule-closes-g-071.md | M-075 body section \`## Goal\` is empty |
| entity-body-empty | M-075 | work/epics/E-22-planning-toolchain-fixes-closes-g-071-g-072-g-065/M-075-lifecycle-gate-entity-body-empty-rule-closes-g-071.md | M-075 body section \`## Acceptance criteria\` is empty |

## Recent activity

| Date | Actor | Verb | Detail |
|------|-------|------|--------|
| 2026-05-08 | human/peter | edit-body | aiwf edit-body E-22 |
| 2026-05-08 | human/peter | render-roadmap | aiwf render roadmap |
| 2026-05-08 | human/peter | edit-body | aiwf edit-body E-22 |
| 2026-05-08 | human/peter | add | aiwf add epic E-22 'Planning toolchain fixes (closes G-071, G-072, G-065)' |
| 2026-05-08 | human/peter | add | aiwf add gap G-073 'depends_on is restricted to milestone→milestone edges; cross-kind blocking lives in body prose only; subsumes G-072 in scope' |

