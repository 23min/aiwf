---
id: M-0108
title: Author legal-workflows.md spec enumerating every blessed workflow
status: draft
parent: E-0031
tdd: none
---
## Goal

Author `docs/pocv3/design/legal-workflows.md` enumerating every blessed workflow currently encoded in skill bodies under `.claude/skills/aiwfx-*` and `wf-rituals:*`, with each workflow's entry condition, sequenced verb calls, branch each step runs from, post-conditions, and tree-level invariants. Settles the three open questions from E-0031 (workflow granularity, ADR-vs-design-doc, where branch choreography sits in the spec).

## Context

Skill bodies today encode workflow recipes prose-only and inconsistently. There is no canonical artifact a contributor (human or LLM) can cite for "what's the legal sequence for X." This milestone produces that artifact — the foundation every subsequent milestone in E-0031 depends on. No prior milestones; this is the entry point of the epic.

## Approach

Walk the current skill body set (aiwf-add, aiwf-promote, aiwf-rename, aiwfx-plan-epic, aiwfx-plan-milestones, aiwfx-start-milestone, aiwfx-wrap-milestone, aiwfx-wrap-epic, aiwfx-release, aiwf-authorize, aiwf-archive, aiwf-reallocate, wf-rituals:wf-patch, wf-rituals:wf-tdd-cycle, etc.). For each, distill: entry condition (pre-conditions), sequenced verb calls (each step's verb + key flags), branch each step expects (main vs feature branch), post-conditions (tree state after), tree-level invariants the workflow preserves. Lock the workflow granularity decision at "one workflow per skill" (skills are the LLM/human entry point) and the spec-ratification decision at "design doc, not ADR" (the test layer is the actual chokepoint).

## Acceptance criteria

<!-- ACs are added at aiwfx-start-milestone via `aiwf add ac M-0108 --title "..."`. -->

## Surfaces touched

- `docs/pocv3/design/legal-workflows.md` (new)
- `internal/policies/` (structural-assertion test for the spec's required per-workflow sections)

## Out of scope

- Integration tests against the spec (M-0109/M-0110)
- Skill citations back to the spec (M-0111)
- Fuzz harness (M-0112)

## Dependencies

- None — entry milestone of E-0031.
