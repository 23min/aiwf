---
id: M-0218
title: Drive the internal/policies test suite below its ~9s floor
status: draft
parent: E-0053
tdd: advisory
---
## Goal

Drive the `internal/policies` test suite below its residual ~9s
parallel-bound floor (gap `G-0321`), left after the `G-0320` fixture fix
removed the 82s outlier.

Deliverable: an investigation and, where the leverage justifies the rewrite
risk, optimization of the handful of heavyweight tests that gate the suite
wall-clock — the git-fixture verb subtests (each builds its own repo) and
the source-tree-walking auditors. Targets a ~5s suite without faking the
integration assertions (the build-tag and golangci-config firing tests
legitimately compile/lint and stay as-is).

## Notes

Lower leverage than the `aiwf check` milestones — the next tier is ~4s, not
82s. May conclude "not worth it" after measurement; that is an acceptable
outcome recorded in validation. Acceptance criteria authored when the
milestone starts.
