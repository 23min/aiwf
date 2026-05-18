---
id: M-0123
title: Pass C reconcile to canonical Go spec table + drift policy
status: draft
parent: E-0033
depends_on:
    - M-0121
    - M-0122
tdd: required
---
## Goal

Reconcile M-0121's audit catalog with M-0122's first-principles catalog into the canonical **Go spec table** that downstream tests (M-0124, M-0125) drive against the binary, plus the **drift policy** that ensures the spec stays closed-set against the impl.

This is where the epic's deliverable lands. The two catalogs are evidence; this milestone produces the actual spec.

## Reconciliation classes

Each rule from either catalog falls into one of:

| Class | M-0121 says | M-0122 says | Resolution |
|---|---|---|---|
| **Agreement** | legal | legal | Spec entry, marked agreed |
| **Agreement** | illegal | illegal | Spec entry, marked agreed |
| **Audit-only** | rule X | silent | Spec entry citing audit source |
| **FP-only** | silent | rule X | Decision needed: do we ratify? |
| **Conflict** | rule X | rule Y (different) | Decision needed: which wins, why |
| **Undefined-by-both** | silent | silent | Surface as known-undecided; M-γ decides posture (see "negative-of-undefined" discussion) |

Each conflict and FP-only entry surfaces a decision. Capture each via `aiwf add decision --relates-to E-0033`, with body explaining the choice and the rationale.

## Canonical Go spec

Under `internal/workflows/spec/` (exact package name decided during this milestone):

- A `Rule` struct capturing one legality cell (kind / state / verb / preconditions / expected outcome / severity).
- A `Rules() []Rule` function exposing the closed-set table.
- A `LookupRule(kind, state, verb)` helper for tests to call.

The schema is designed once the catalogs are in front of us — not before. Likely fields: `Kind`, `FromState`, `Verb`, `Preconditions []Predicate`, `Expected Outcome`, `ExpectedErrorCode` (for illegal cells), `RuleSource` (citation back to the audit or FP doc).

## Drift policy

A test under `internal/policies/` that asserts:

- Every kind/state pair the impl's FSM tables recognize has at least one corresponding rule in the spec.
- Every top-level Cobra verb is referenced by at least one rule.
- Every `aiwf check` finding code that pertains to verb legality is referenced by at least one illegal-cell rule (advisory codes are exempt).

Failure of any of these is a hard CI block — the impl cannot grow a new verb/state/finding without the spec growing too.

## Acceptance criteria

(Added via `aiwf add ac` after the reconciliation pass produces a draft schema.)

## Approach

- Sit with both catalogs open simultaneously. Walk rule-by-rule.
- For Agreements → directly add to spec.
- For Audit-only / FP-only / Conflicts → open a decision entity per case, write the rationale, then add the spec entry the decision pinned.
- Design the Go schema only after the reconciliation walk — the catalog shape informs the schema, not the other way round.
- Land the drift policy in the same milestone as the spec table.

## Open question to settle here

The "negative-of-undefined" posture (cells the spec deliberately leaves silent) is decided in this milestone, based on whether reconciliation actually surfaces any genuinely undecidable cells. Default lean: closed spec (every cell decided one way or the other). Decide otherwise only if forced by reality.
