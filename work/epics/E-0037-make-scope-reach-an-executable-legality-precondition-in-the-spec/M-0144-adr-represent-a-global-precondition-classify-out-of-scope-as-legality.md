---
id: M-0144
title: 'ADR: represent a global precondition; classify out-of-scope as legality'
status: draft
parent: E-0037
tdd: none
---
## Goal

Author and ratify an ADR that resolves **how a global precondition is represented** in the spec `Rule` table and **classifies out-of-scope as legality**, and that **sizes the cellcoverage extension** — the design decisions the implementation milestones (M-0145/M-0146/M-0147) depend on.

## Context

D-0014 / G-0171 set the *directions* during E-0037 planning: a single marked global/cross-cutting `Rule` (single source of truth); `provenance-authorization-out-of-scope` as `codes.ClassLegality`; full cellcoverage integration with a documented fallback. This milestone formalizes the exact mechanism against the *real* code — `internal/workflows/spec` (Rule shape, key-uniqueness), the AC-5 fourth arm (`m0123_ac5_drift_test.go`), and `internal/cellcoverage` — before any of it is built. **Reviewed reconcile**: read those surfaces and surface divergence from the directions before ratifying.

## Acceptance criteria

(ACs allocated separately via `aiwf add ac` after milestone creation; bodies seeded at allocation time.)

## Constraints

- **Decision is decision** (CLAUDE.md): ratify via `aiwf promote ADR-NNNN accepted`; no gate language in the ADR body.
- Reviewed reconcile before ratifying.
- AC promotion requires mechanical evidence (structural section assertions, scoped not flat-grep).

## Out of scope

The implementation (M-0145/M-0146/M-0147). This milestone produces the decision, not the code.

## Dependencies

None — keystone. M-0145/M-0146/M-0147 depend on it.
