---
id: M-0124
title: 'Positive cell coverage: legal workflows succeed with expected post-state'
status: in_progress
parent: E-0033
depends_on:
    - M-0123
    - M-0130
    - M-0131
tdd: required
acs:
    - id: AC-1
      title: 'spec.EvaluatePredicate primitive: closed Subject + Op vocabulary over Predicate'
      status: met
      tdd_phase: done
    - id: AC-2
      title: Fixture helpers + satisfyPredicate self-verification
      status: met
      tdd_phase: done
    - id: AC-3
      title: 'Per-cell positive driver: table-driven over spec.Rules() Legal cells'
      status: met
      tdd_phase: done
    - id: AC-4
      title: 'Coverage meta-test: every Legal rule has a corresponding positive subtest'
      status: open
      tdd_phase: red
---
## Goal

For every **legal** cell in M-0123's spec table, write a test under `internal/policies/` that drives the real `aiwf` binary against a fixture tree, executes the cell's verb, and asserts:

- Exit code 0
- The post-state matches the rule's `Expected` field
- The commit (if any) carries the expected `aiwf-verb` / `aiwf-entity` trailers

Tests are table-driven from the spec — adding a new legal cell to the spec automatically grows the test surface.

## Test shape

Per cell:

1. Build a fixture tree that satisfies the rule's preconditions (predicates + entity states).
2. Invoke the binary: `exec.Command(aiwfBinary, verb, args...).Run()`.
3. Assert success conditions.
4. Re-load the tree via `tree.Load` and assert the post-state.
5. Compare against the rule's `Expected` outcome via `go-cmp`.

The fixture tree is built using kernel verbs themselves (`aiwf add epic`, `aiwf add milestone`, etc.) — not by hand-writing markdown — so the test inputs share the same legality model as the production path.

## Coverage commitment

Every legal cell in `Rules()` has at least one corresponding positive test. A meta-test asserts this: walk `Rules()`, for each rule with `Expected = legal`, confirm there's a test that names that rule id. Missing coverage fails CI.

## Acceptance criteria

(Added via `aiwf add ac` after M-0123 lands the spec schema.)

## Approach

- Build a shared test fixture helper (similar pattern to the existing `internal/policies/shared_tree_test.go` for read-only loading, but the helper here builds *isolated* fixtures per test since they mutate).
- Run tests with the existing parallel-by-default discipline; mark serial only where subprocess saturation demands it (per CLAUDE.md's serial-skip-list discipline).
- Aim for sub-second per-test where possible by sharing the binary build via `sync.Once`.

## What this milestone does *not* do

- Does not cover illegal cells (M-0125's scope).
- Does not test branch-context preconditions (E-0030's scope).
- Does not exercise random sequences (deliberately deferred).

### AC-1 — spec.EvaluatePredicate primitive: closed Subject + Op vocabulary over Predicate

### AC-2 — Fixture helpers + satisfyPredicate self-verification

### AC-3 — Per-cell positive driver: table-driven over spec.Rules() Legal cells

### AC-4 — Coverage meta-test: every Legal rule has a corresponding positive subtest

