---
id: M-0110
title: Per-workflow integration test coverage including G-0118 regression
status: draft
parent: E-0031
depends_on:
    - M-0109
tdd: required
---
## Goal

Add an integration test for every workflow listed in the spec from M-0108. Include G-0118's composition pattern (`reallocate` → downstream verb → provenance audit) as an explicit regression test. Coverage is the success criterion of this milestone.

## Context

M-0109 produced the harness and one seam test; this milestone fills in coverage. Container-shaped — split mid-flight via `aiwfx-record-decision` if the per-workflow test count grows past 3 days of work.

## Approach

For each workflow in the spec: write an integration test under `internal/workflows/` that exercises the workflow end-to-end and asserts the post-conditions named in the spec. Multi-branch workflows (start-epic, start-milestone, wrap-milestone, wrap-epic) use the multi-branch fixture support from M-0109. The G-0118 regression test composes `add epic → add milestone → reallocate milestone → promote downstream` and asserts the provenance audit passes (the bug class was `reallocate` not populating `prior_ids`).

## Acceptance criteria

<!-- ACs are added at aiwfx-start-milestone via `aiwf add ac M-0110 --title "..."`. -->

## Surfaces touched

- `internal/workflows/` (existing package; new test files per workflow)

## Out of scope

- Drift-prevention test (M-0111)
- Fuzz harness (M-0112)

## Dependencies

- M-0109 (harness must exist)
