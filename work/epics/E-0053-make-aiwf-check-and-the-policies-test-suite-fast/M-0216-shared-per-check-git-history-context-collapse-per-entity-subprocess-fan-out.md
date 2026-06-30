---
id: M-0216
title: Shared per-check git-history context; collapse per-entity subprocess fan-out
status: draft
parent: E-0053
tdd: required
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
