---
id: G-0262
title: audit the whole test corpus for vacuous assertions, not just policies
status: open
---
## What's missing

G-0259 audited one corner — the firing paths of `internal/policies/` — and found 75% dark. That dynamic (the same LLM authored the implementation *and* its tests, graded on the tests passing) is not specific to policies; it applies to the **whole module**. No equivalent audit has been run across the rest of the test suite, so the strength of the assertions guarding `internal/verb`, `internal/check`, `internal/entity`, `internal/gitops`, the renderers, and the rest is unknown.

The repo already carries evidence the problem recurs outside policies: the CLAUDE.md "substring assertions are not structural assertions" lesson came from milestone-page tests that asserted `id="ac-1"` existed *somewhere* and would have passed with the AC rendered in the wrong tab. That is a vacuous assertion in production test code, found by audit, not by the suite.

This gap is the **retroactive corpus-wide sweep**: apply the `wf-vacuity` lens (mutation probe + assertion-shape probe) to the existing test suite as a whole, not one unit at a time.

## Scope and method

Two passes, mirroring the two probes:

1. **Mechanical (probe 1) — `mutate-hunt` corpus-wide.** Run the mutation harness per package (`--workers 1`, `--timeout-coefficient 15` per the repo's tuning), read survivors, fix or document each. This is the objective floor and the bulk of the coverage.
2. **Judgment (probe 2) — directed `wf-vacuity` on the load-bearing units.** Assertion-shape reasoning that mutation cannot do (tautologies, over-narrowed antecedents/fixtures, substring-not-structural). Cannot be mechanized, so it is a *directed* manual/LLM sweep over the highest-value units — the FSM, the id allocator, parsers/serializers, the verb plans, the check rules, the renderers — not an undirected whole-tree pass.

## Why it matters

G-0259 + the meta-chokepoint stop *new* vacuity in the policies corner; G-0260/G-0261 wire the rituals so future units get checked. Neither cleans up the **existing debt across the rest of the module**. A green suite that has never been adversarially probed is a suite of unknown strength — and the substring-vs-structural case proves the debt is real, not hypothetical.

## Shape

Milestone/epic-scale, not a patch — the unit of work is the whole test suite. Prioritize by blast radius: the load-bearing kernel packages first (`entity`/FSM, `verb`, `check`, `gitops`), renderers and CLI surfaces second. Each surviving mutant or vacuous assertion becomes either a strengthened assertion or a documented, justified exclusion.

## Source

G-0259 (the policies-corner finding that motivated this); the CLAUDE.md "substring assertions are not structural assertions" lesson (independent evidence of vacuity in production test code); the `mutate-hunt` workflow; G-0258 (`wf-vacuity`). Companion process gaps: G-0260 (`wf-vacuity` wiring), G-0261 (`wf-rethink` wiring).
