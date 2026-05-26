---
id: M-0144
title: 'ADR: represent a global precondition; classify out-of-scope as legality'
status: draft
parent: E-0037
tdd: none
acs:
    - id: AC-1
      title: ADR ratifies global-rule representation and its meta-test composition
      status: open
    - id: AC-2
      title: ADR classifies out-of-scope as ClassLegality with dual-emission rationale
      status: open
    - id: AC-3
      title: ADR sizes cellcoverage extension and states explicit fallback condition
      status: open
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

### AC-1 — ADR ratifies global-rule representation and its meta-test composition

The ADR ratifies the global-rule representation mechanism (e.g. a `KindAny` sentinel vs. a `Global` flag on `Rule`) and states how it composes with the `Rule` key-uniqueness + coverage meta-tests (`m0123_ac2/ac4`, `m0124/m0125`) and how the AC-5 fourth arm recognizes it.

*Evidence:* a structural policy assertion that the ADR resolves via the loader, is `accepted`, and its Decision section names the chosen mechanism.

### AC-2 — ADR classifies out-of-scope as ClassLegality with dual-emission rationale

The ADR records out-of-scope as `ClassLegality` with the dual-emission rationale (verb-time refusal + check-time audit are one violation at two surfaces) and the `codes.go` carve-out note.

*Evidence:* structural assertion on the named section.

### AC-3 — ADR sizes cellcoverage extension and states explicit fallback condition

The ADR sizes the cellcoverage extension and states the explicit fallback condition (dedicated test + recorded exemption only if the extension proves its own epic).

*Evidence:* structural assertion the sizing + fallback are present.

