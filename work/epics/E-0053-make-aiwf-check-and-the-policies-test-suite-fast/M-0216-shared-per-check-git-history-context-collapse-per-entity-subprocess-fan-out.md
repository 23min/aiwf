---
id: M-0216
title: Shared per-check git-history context; collapse per-entity subprocess fan-out
status: in_progress
parent: E-0053
tdd: required
acs:
    - id: AC-1
      title: Orphaned-AI-commit walk uses in-memory DAG ancestry, no per-pair merge-base
      status: met
      tdd_phase: done
    - id: AC-2
      title: Shared per-check git-history context consumed by the history-walking rules
      status: met
      tdd_phase: done
    - id: AC-3
      title: aiwf check findings byte-identical before and after the refactor
      status: open
      tdd_phase: red
    - id: AC-4
      title: Measured check wall-time delta recorded in Validation
      status: open
      tdd_phase: red
---
## Goal

Eliminate the per-entity git-subprocess fan-out in `aiwf check` by loading
git history once per check and sharing it across every history-walking rule.

Deliverable: (1) a shared per-check history context — commits, trailers, and
the commit DAG read in a single pass — consumed by the rules that today each
re-walk history independently; (2) the orphaned-AI-commit walk rewritten to
answer ancestry from the in-memory DAG, collapsing its 683
`git merge-base --is-ancestor` subprocess spawns to a single bulk read.
Findings must be byte-identical before and after, pinned by the existing
rule fixtures, with a measured wall-time delta against the baseline.

## Notes

Behavior-preserving refactor; the ancestry and FSM-history semantics are the
correctness surface. Acceptance criteria authored when the milestone starts.

### AC-1 — Orphaned-AI-commit walk uses in-memory DAG ancestry, no per-pair merge-base

### AC-2 — Shared per-check git-history context consumed by the history-walking rules

### AC-3 — aiwf check findings byte-identical before and after the refactor

### AC-4 — Measured check wall-time delta recorded in Validation

