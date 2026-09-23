---
name: code-health
description: Stack-agnostic field guide of code-health principles — module boundaries, contracts, data discipline, tests that pin behavior, errors/logs/audit, reasoning aids, operational properties. Use when designing a module, planning a refactor, reviewing a non-trivial diff, or scoring a codebase Strong/Weak/Missing with file:line evidence.
---

# code-health

Advisory **forces, not rules** — consult them; the project's own conventions win. Two uses:
**prime** (hold the high-leverage ones while writing) and **score** (rate a unit
Strong / Weak / Missing per principle, with file:line evidence). The short priming subset
is in my standing guidance; this is the full rubric for deeper passes.

## A. Module boundaries
- **A1 High cohesion.** A module does one thing; everything in it changes for the same reason.
- **A2 Low coupling.** Modules depend on narrow interfaces, not each other's internals — a
  change ripples to few places.
- **A3 Layered, no upward dependencies.** Dependencies point one way, toward the core; lower
  layers never import higher ones.

## B. Contracts
- **B1 Typed interfaces.** Boundaries pass named types, not loose maps, positional tuples, or
  magic strings.
- **B2 Schemas at boundaries.** Every shape crossing a process/IO boundary has one declared
  schema, validated on read at the edge — not deep in the consumer. No blind casts.
- **B3 Pre/post-conditions & invariants.** State what must hold on entry and exit; assert
  invariants where they could actually break, not everywhere.

## C. Data discipline
- **C1 Single source of truth.** Each fact lives in one place; derive the rest with pure
  functions. Every cache names its invalidation rule.
- **C2 Idempotence.** Re-running an operation yields the same state — safe to retry.
  Distinguish create-or-update from create.
- **C3 Atomic writes.** Persisted state is fully-old or fully-new: write a sibling temp +
  rename; multi-file → write all temps, then rename in order.
- **C4 Versioned schemas with migration paths.** Persisted formats carry a version; readers
  handle old versions or a migration upgrades them. No silent format breaks.

## D. Tests that pin behavior, not implementation
- **D1 Behavior pinned, not structure.** Assert what the code does for given inputs; mock only
  at process/network/filesystem seams, never internal functions.
- **D2 Equivalence tests at seams.** Where one side writes a value another reads, test the
  writer↔reader pair against shared scenarios.
- **D3 Branch coverage on touched code.** Every reachable branch on changed lines has a test,
  or an explicit ignore with a reason.
- **D4 Tests at the right altitude.** Unit-test pure logic; integration-test seams; don't e2e
  what a unit test pins, or unit-test what only appears integrated.

## E. Errors, logs, audit trail
- **E1 Structured logs.** Events have a name + fields, not interpolated prose. One logger per
  process; bare print/console.log only in throwaway scripts.
- **E2 Designed failure modes.** Decide per failure: retry, fail-fast, degrade, or escalate.
  No bare catch-and-continue.
- **E3 Audit trail.** State-changing actions are reconstructable after the fact
  (who / what / when) where it matters.
- **E4 Self-explaining errors.** An operator-facing failure says what failed, the current
  state, and what to do next.

## F. Reasoning aids
- **F1 Names that don't lie.** A name says what the thing is or does; rename when behavior
  drifts. No `data2`, no `helper` doing five things.
- **F2 Comments only for non-obvious "why".** Code says what; comments capture hidden
  constraints, trade-offs, and workarounds — not a paraphrase of the line below.
- **F3 Decision records that survive turnover.** Non-obvious architectural choices are written
  down so the "why" outlives the author.

## G. Operational properties
- **G1 Reproducible.** Same inputs → same outputs; push time, randomness, and environment to
  the edges so the core is deterministic.
- **G2 Reversible.** A change can be undone — feature flag, migration down-path, or "open a new
  entity for the inverse." Know the undo before you ship.
- **G3 Observable in production.** You can tell what the system is doing from the outside —
  metrics, logs, and health at the seams that matter.

## Scoring a unit
For each principle: **Strong** (holds, with evidence), **Weak** (partial / inconsistent), or
**Missing** (absent / violated). Cite file:line. Don't grade on a curve — Weak is Weak.

## When NOT to apply this
Throwaway scripts, spikes, and one-off migrations don't owe you C4 / E3 / G2 — match the rigor
to the code's lifespan. Every principle taken too far becomes its own anti-pattern (B2 →
schema-everything ceremony; A2 → indirection soup). Prefer the boring middle.
