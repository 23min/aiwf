---
id: M-0168
title: Corpus-wide mutate-hunt sweep over the kernel packages
status: draft
parent: E-0042
tdd: none
---
## Deliverable

A `mutate-hunt` sweep over the load-bearing kernel packages, with every
survivor dispositioned: either a new or strengthened assertion that now kills
it, or a documented, justified exclusion (equivalent mutant, unreachable
branch).

## Scope

Per-package mutation runs, prioritized by blast radius (G-0262):

- `internal/entity` (the FSM and id allocator), `internal/gitops`,
  `internal/verb`, `internal/check` first — the kernel.
- Renderers and CLI surfaces second, as budget allows.

Use the repo's tuning: `--workers 1`, `--timeout-coefficient 15`. Read
survivors carefully — equivalent-mutant and unreachable-branch noise are common
false positives and are not chased; real survivors are concrete file:line
entries that warrant a test or a refactor.

## Outcome

A survivor-disposition record per swept package: the objective floor for the
strength of the kernel's test assertions.

*Draft stub — acceptance criteria pinned when the milestone starts.*
