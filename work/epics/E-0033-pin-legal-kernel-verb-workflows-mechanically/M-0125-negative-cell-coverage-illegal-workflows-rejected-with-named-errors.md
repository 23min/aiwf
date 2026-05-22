---
id: M-0125
title: 'Negative cell coverage: illegal workflows rejected with named errors'
status: draft
parent: E-0033
depends_on:
    - M-0123
    - M-0130
    - M-0131
tdd: required
acs:
    - id: AC-1
      title: Negative-precondition fixture helpers + self-verification
      status: open
      tdd_phase: red
    - id: AC-2
      title: 'Per-cell negative driver: verb-time rejection (exit-code + rollback)'
      status: open
      tdd_phase: red
    - id: AC-3
      title: 'Per-cell negative driver: check-time rejection (finding-code present)'
      status: open
      tdd_phase: red
---
## Goal

For every **illegal** cell in M-0123's spec table, write a test under `internal/policies/` that drives the real `aiwf` binary against a fixture tree, attempts the cell's verb, and asserts:

- Non-zero exit code (matching the rule's expected severity), or
- The expected finding code appears in `aiwf check --format=json` output (for check-time cells)
- The pre-state is unchanged (or the planning tree is otherwise in the rule's `ExpectedRollbackState`)

This is the closing chokepoint: if the impl ever silently *allows* an illegal workflow, the corresponding negative test fails and CI blocks.

## Severity model

The spec table's illegal cells carry a two-axis severity model (decided in M-0123): `RejectionLayer` (verb-time | check-time) + `BlockingStrict` (bool). The test shape derives from the layer:

- **Verb-time rejection** — verb returns non-zero exit, no commit, no side effect. Tested via subprocess invocation + exit-code assertion + `git status` rollback verification.
- **Check-time rejection (blocking)** — verb succeeds; `aiwf check --strict` returns non-zero with a specific finding code. Tested via verb → check → finding-code inspection.
- **Check-time advisory** — verb succeeds; `aiwf check` returns 0 but emits a warning-severity finding. Tested via finding-code inspection without exit-code assertion.

If M-0123's reconciliation walk surfaces too few `(check-time, advisory)` cells to justify the second axis and the model collapses to a single `Severity` enum (`HardReject | CheckError | CheckWarning`), the three test shapes above remain — only the field they key off changes. M-0125 consumes whichever shape M-0123 lands on.

## Test shape

Per illegal cell:

1. Build a fixture tree that satisfies the cell's preconditions.
2. Attempt the verb.
3. Assert the expected severity behavior (verb exit code, check exit code, or finding code presence per the cell's `RejectionLayer` / `BlockingStrict`).
4. Confirm rollback / no-side-effect for verb-time-rejection cells.
5. Confirm finding code via `aiwf check --format=json` for check-time cells.

## Coverage commitment

Every illegal cell in `Rules()` has at least one negative test. The meta-test from M-0124 extends to require negative coverage as well — `Outcome = illegal` rules without a matching test fail CI.

## Acceptance criteria

(Added via `aiwf add ac` after M-0124 lands the positive test scaffolding, since the negative tests reuse the fixture helpers.)

## Approach

- Reuse the fixture builder from M-0124 with helpers that *deliberately* establish forbidden preconditions.
- For verb-time-rejection cells: subprocess invocation + exit-code check + `git status` rollback verification.
- For check-time cells (blocking or advisory): structured envelope parsing of `aiwf check --format=json`.
- Negative tests are often shorter than positive ones (just assert rejection) but more diverse (every illegal cell is its own scenario).

## What this milestone does *not* do

- Does not test branch-context illegality (E-0030's scope).
- Does not include fuzz / random walks.
- Does not extend the spec table — only consumes M-0123's output.

### AC-1 — Negative-precondition fixture helpers + self-verification

### AC-2 — Per-cell negative driver: verb-time rejection (exit-code + rollback)

### AC-3 — Per-cell negative driver: check-time rejection (finding-code present)

