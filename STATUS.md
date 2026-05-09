# aiwf status — 2026-05-09

_197 entities · 0 errors · 10 warnings · run `aiwf check` for details_

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

### E-21 — Open-work synthesis: aiwfx-whiteboard skill replaces critical-path.md _(proposed)_

- ✓ **M-078** — Planning-conversation skills design ADR (placement, tiering, name rationale) _(done)_ — ACs 5/5 met — tdd: none
- → **M-079** — aiwfx-whiteboard skill: classification rubric, output template, Q&A gate _(in_progress)_ — ACs 8/8 met — tdd: advisory
- **M-080** — Whiteboard skill fixture validation; retire critical-path.md; close E-21 _(draft)_ — ACs 0/7 met (7 open) — tdd: required

```mermaid
flowchart LR
  E_21["E-21<br/>Open-work synthesis: aiwfx-whiteboard skill replaces critical-path.md"]:::epic_proposed
  M_078["M-078 (5/5)<br/>Planning-conversation skills design ADR (placement, tiering, name rationale)"]:::ms_done
  E_21 --> M_078
  M_079["M-079 (8/8)<br/>aiwfx-whiteboard skill: classification rubric, output template, Q&A gate"]:::ms_in_progress
  E_21 --> M_079
  M_080["M-080 (0/7)<br/>Whiteboard skill fixture validation; retire critical-path.md; close E-21"]:::ms_draft
  E_21 --> M_080
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
| ADR-0005 | adr | Verb hygiene contract: complete, consistent, pre-flighted aiwf verbs | proposed |
| ADR-0006 | adr | Skills policy: per-verb default; topical multi-verb when concept-shaped; no skill when --help suffices | proposed |
| ADR-0007 | adr | Planning-conversation skills: rituals-plugin placement; pure-skill first, kernel verb only if usage demands it | proposed |

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
| G-063 | No defined start-epic ritual: epic activation is a deliberate sovereign act with preflight + optional delegation, but kernel treats it as a one-line FSM flip |  |
| G-065 | No aiwf retitle verb: scope refactors that change an entity's or AC's intent leave frontmatter title fields permanently misleading; only slug rename is supported |  |
| G-067 | wf-tdd-cycle is LLM-honor-system advisory; under load the LLM bypasses RED-first and the branch-coverage HARD RULE without anything mechanical catching it (M-066/AC-1 cycle wrote ~165 lines of impl before any test existed) | M-066 |
| G-068 | Discoverability policy misses dynamic finding subcodes | M-066 |
| G-069 | aiwf init's printRitualsSuggestion hardcodes the CLI install form, which defaults to user scope and won't satisfy doctor.recommended_plugins; nudge silently steers fresh operators away from project-scope outcome | M-070 |
| G-070 | aiwf doctor has no --format=json envelope; M-070's recommended-plugin-not-installed finding-code surfaces only as human text. Add JSON envelope when a JSON-consuming caller appears | M-070 |
| G-073 | depends_on is restricted to milestone→milestone edges; cross-kind blocking lives in body prose only; subsumes G-072 in scope | E-21 |
| G-074 | docs/pocv3/ body prose still uses PoC framing; needs sweep |  |
| G-075 | docs/pocv3/ directory naming is now historical; rename or accept |  |
| G-076 | CONTRIBUTING.md describes PR-based workflow at odds with trunk-based model on main |  |
| G-077 | Post-promotion working paper (aiwf's thesis) not yet written |  |
| G-078 | No priority field on entities; backlog isn't filterable or sortable by importance |  |
| G-079 | aiwfx-plan-milestones plugin skill needs --depends-on documentation; M-076 added the verb but the plugin lives in ai-workflow-rituals upstream | M-076 |
| G-080 | Wide-table verbs wrap mid-row and break column scan; no TTY-aware sizing, glyph palette, or truncation surface | M-076 |
| G-081 | aiwf rename does not pre-flight trunk-collision check | E-21 |
| G-082 | Planning closure should default-merge to main before implementation begins | E-21 |
| G-083 | aiwf retitle does not sync entity body H1 with frontmatter title | E-21 |
| G-084 | Verb hygiene contract is undocumented; G-081/G-082/G-083 lack umbrella | E-21 |
| G-086 | docs/pocv3/contracts.md still references non-existent aiwf list contracts (lines 98, 114-117); same drift class as G-061/G-085, different file | M-072 |
| G-087 | no aiwf-show embedded skill; show is the per-entity inspection verb every AI reaches for, but --help-only coverage misses body-rendering and composite-id discovery | M-074 |
| G-088 | Skill-coverage policy walks internal/skills/embedded/ only; plugin skills (aiwf-extensions/skills/aiwfx-*) are not policed by the kernel — equivalent invariants must be re-applied per-skill in test code as M-079 did | M-079 |

## Warnings

| Code | Entity | Path | Message |
|------|--------|------|---------|
| acs-tdd-audit | M-079/AC-1 | work/epics/E-21-open-work-synthesis-recommended-sequence-skill-replaces-critical-path-md/M-079-aiwfx-whiteboard-skill-classification-rubric-output-template-q-a-gate.md | M-079/AC-1 status: met under tdd: advisory but tdd_phase is red (expected done) |
| acs-tdd-audit | M-079/AC-2 | work/epics/E-21-open-work-synthesis-recommended-sequence-skill-replaces-critical-path-md/M-079-aiwfx-whiteboard-skill-classification-rubric-output-template-q-a-gate.md | M-079/AC-2 status: met under tdd: advisory but tdd_phase is (absent) (expected done) |
| acs-tdd-audit | M-079/AC-3 | work/epics/E-21-open-work-synthesis-recommended-sequence-skill-replaces-critical-path-md/M-079-aiwfx-whiteboard-skill-classification-rubric-output-template-q-a-gate.md | M-079/AC-3 status: met under tdd: advisory but tdd_phase is (absent) (expected done) |
| acs-tdd-audit | M-079/AC-4 | work/epics/E-21-open-work-synthesis-recommended-sequence-skill-replaces-critical-path-md/M-079-aiwfx-whiteboard-skill-classification-rubric-output-template-q-a-gate.md | M-079/AC-4 status: met under tdd: advisory but tdd_phase is (absent) (expected done) |
| acs-tdd-audit | M-079/AC-5 | work/epics/E-21-open-work-synthesis-recommended-sequence-skill-replaces-critical-path-md/M-079-aiwfx-whiteboard-skill-classification-rubric-output-template-q-a-gate.md | M-079/AC-5 status: met under tdd: advisory but tdd_phase is (absent) (expected done) |
| acs-tdd-audit | M-079/AC-6 | work/epics/E-21-open-work-synthesis-recommended-sequence-skill-replaces-critical-path-md/M-079-aiwfx-whiteboard-skill-classification-rubric-output-template-q-a-gate.md | M-079/AC-6 status: met under tdd: advisory but tdd_phase is (absent) (expected done) |
| acs-tdd-audit | M-079/AC-7 | work/epics/E-21-open-work-synthesis-recommended-sequence-skill-replaces-critical-path-md/M-079-aiwfx-whiteboard-skill-classification-rubric-output-template-q-a-gate.md | M-079/AC-7 status: met under tdd: advisory but tdd_phase is (absent) (expected done) |
| acs-tdd-audit | M-079/AC-8 | work/epics/E-21-open-work-synthesis-recommended-sequence-skill-replaces-critical-path-md/M-079-aiwfx-whiteboard-skill-classification-rubric-output-template-q-a-gate.md | M-079/AC-8 status: met under tdd: advisory but tdd_phase is (absent) (expected done) |
| entity-body-empty | G-088 | work/gaps/G-088-skill-coverage-policy-walks-internal-skills-embedded-only-plugin-skills-aiwf-extensions-skills-aiwfx-are-not-policed-by-the-kernel-equivalent-invariants-must-be-re-applied-per-skill-in-test-code-as-m-079-did.md | G-088 body section \`## What's missing\` is empty |
| entity-body-empty | G-088 | work/gaps/G-088-skill-coverage-policy-walks-internal-skills-embedded-only-plugin-skills-aiwf-extensions-skills-aiwfx-are-not-policed-by-the-kernel-equivalent-invariants-must-be-re-applied-per-skill-in-test-code-as-m-079-did.md | G-088 body section \`## Why it matters\` is empty |

## Recent activity

| Date | Actor | Verb | Detail |
|------|-------|------|--------|
| 2026-05-09 | human/peter | promote | aiwf promote M-079/AC-8 open -> met |
| 2026-05-09 | human/peter | promote | aiwf promote M-079/AC-7 open -> met |
| 2026-05-09 | human/peter | add | aiwf add gap G-088 'Skill-coverage policy walks internal/skills/embedded/ only; plugin skills (aiwf-extensions/skills/aiwfx-*) are not policed by the kernel — equivalent invariants must be re-applied per-skill in test code as M-079 did' |
| 2026-05-09 | human/peter | promote | aiwf promote M-079/AC-6 open -> met |
| 2026-05-09 | human/peter | promote | aiwf promote M-079/AC-5 open -> met |

