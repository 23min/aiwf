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
---
## Goal

For every **illegal** cell in M-0123's spec table, write a test under `internal/policies/` that drives the real `aiwf` binary against a fixture tree, attempts the cell's verb, and asserts:

- Non-zero exit code (matching the rule's expected severity tier)
- The error or finding code matches the rule's `ExpectedErrorCode`
- The pre-state is unchanged (or the planning tree is otherwise in the rule's `ExpectedRollbackState`)

This is the closing chokepoint: if the impl ever silently *allows* an illegal workflow, the corresponding negative test fails and CI blocks.

## Severity tiers

The spec table's illegal cells will fall into severity tiers (decided in M-0123):

- **Hard reject** — verb returns non-zero exit, no commit, no side effect. Tested via subprocess invocation + exit-code assertion.
- **Aiwf-check error** — verb succeeds but `aiwf check --strict` returns non-zero with a specific finding code. Tested via verb → check → finding inspection.
- **Aiwf-check warning** — verb succeeds, `aiwf check` returns 0 but emits a warning-severity finding. Tested via finding inspection.

Each illegal cell carries one of these three tiers in its spec entry.

## Test shape

Per illegal cell:

1. Build a fixture tree that satisfies the cell's preconditions.
2. Attempt the verb.
3. Assert the expected severity-tier behavior.
4. Confirm rollback / no-side-effect for hard-rejects.
5. Confirm finding code via `aiwf check --format=json` for check-tier cells.

## Coverage commitment

Every illegal cell in `Rules()` has at least one negative test. The meta-test from M-0124 extends to require negative coverage as well — `Expected = illegal` rules without a matching test fail CI.

## Acceptance criteria

(Added via `aiwf add ac` after M-0124 lands the positive test scaffolding, since the negative tests reuse the fixture helpers.)

## Approach

- Reuse the fixture builder from M-0124 with helpers that *deliberately* establish forbidden preconditions.
- For hard-reject cells: subprocess invocation + exit-code check + `git status` rollback verification.
- For check-tier cells: structured envelope parsing of `aiwf check --format=json`.
- Negative tests are often shorter than positive ones (just assert rejection) but more diverse (every illegal cell is its own scenario).

## What this milestone does *not* do

- Does not test branch-context illegality (E-0030's scope).
- Does not include fuzz / random walks.
- Does not extend the spec table — only consumes M-0123's output.
