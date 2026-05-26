---
id: M-0146
title: Extend cellcoverage with authorized-scope fixtures
status: in_progress
parent: E-0037
depends_on:
    - M-0144
    - M-0145
tdd: required
acs:
    - id: AC-1
      title: Fixture framework builds an authorized-scope context a driver consumes
      status: open
      tdd_phase: green
    - id: AC-2
      title: Positive and negative scope-reach scenarios exercisable via the driver path
      status: open
      tdd_phase: red
---
## Goal

Extend the `internal/cellcoverage` fixture framework to stand up an **authorized-scope context** (an `authorize` commit + an agent actor) so the `m0124`/`m0125` positive/negative drivers can exercise a `scope-reach`-gated rule like any other cell.

## Context

`cellcoverage` today builds entity-state fixtures and evaluates entity-side predicates; it has no notion of an authorized-scope context (who is authorized on what). A `scope-reach` rule cannot be exercised by an entity fixture alone — it needs an open scope + an agent actor. This is E-0037's **unsized-risk** piece; M-0144's ADR sizes it and records the fallback (a dedicated test + AC-4 exemption) should it prove its own epic.

## Acceptance criteria

- **AC1** — The fixture framework can build an authorized-scope context (open scope on an entity, agent actor) that a driver consumes. *Evidence:* a test that builds the scope fixture and asserts the scope is active / loadable.
- **AC2** — A positive and a negative `scope-reach` scenario are exercisable through the driver path (in-scope agent verb succeeds; out-of-scope refused). *Evidence:* a driver-level test exercising both arms against the real binary (the global rule itself lands in M-0147; this milestone proves the *machinery* exercises a scope-gated cell).

## Constraints

- Per M-0144's sizing decision; if the extension exceeds this milestone, the ADR's recorded fallback applies.
- `tdd: required`.

## Out of scope

The global `scope-reach` rule + the reclassification + AC-5 (M-0147). This milestone is the test-infrastructure, not the rule.

## Dependencies

M-0144 (ADR sizing), M-0145 (the evaluable predicate the driver exercises).

### AC-1 — Fixture framework builds an authorized-scope context a driver consumes

The fixture framework can build an authorized-scope context (an open `aiwf authorize` scope on an entity + an agent actor) that a driver consumes.

*Evidence:* a test that builds the scope fixture and asserts the scope is active / loadable.

### AC-2 — Positive and negative scope-reach scenarios exercisable via the driver path

A positive and a negative `scope-reach` scenario are exercisable through the driver path (in-scope agent verb succeeds; out-of-scope refused).

*Evidence:* a driver-level test exercising both arms against the real binary. The global rule itself lands in M-0147; this milestone proves the *machinery* exercises a scope-gated cell (via the existing runtime `provenance-authorization-out-of-scope` gate).

