---
id: M-0215
title: Profile aiwf check and the policies suite to a per-rule wall-time baseline
status: draft
parent: E-0053
tdd: none
acs:
    - id: AC-1
      title: Record aiwf check CPU profile and git-subprocess attribution
      status: open
    - id: AC-2
      title: Record policies-suite per-test timing ranking floor-gating tests
      status: open
---
## Goal

Establish a per-rule wall-time baseline for `aiwf check` and the
`internal/policies` test suite, so every later optimization in this epic is
measured against a recorded before/after rather than guessed.

Deliverable: a CPU profile (pprof) and subprocess attribution of one full
`aiwf check` over a representative tree, plus a per-test timing snapshot of
the policies suite, recorded in this milestone's validation. Confirms the
diagnosis (one check spawns ~895 git subprocesses, 683 of them
`git merge-base --is-ancestor` from the orphaned-AI-commit walk) with a
clean second-by-second budget — the strace count is direction; the profile
is the budget.

## Notes

Profile before optimizing. No production code changes land beyond temporary,
reverted profiling instrumentation. Acceptance criteria are authored when
the milestone starts.

### AC-1 — Record aiwf check CPU profile and git-subprocess attribution

### AC-2 — Record policies-suite per-test timing ranking floor-gating tests

