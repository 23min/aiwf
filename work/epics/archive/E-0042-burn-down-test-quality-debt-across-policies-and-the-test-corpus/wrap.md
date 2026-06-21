# Epic wrap — E-0042

**Date:** 2026-06-20
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0042-burn-down-test-quality-debt-across-policies-and-the-test-corpus
**Merge commit:** 41b6f265

## Milestones delivered

- M-0166 — Firing fixtures for the easy-majority dark policies (c0eab5a6 / faaba198)
- M-0167 — Verified linter cleanups and fsm-invariants ledger annotation (5bf63a22)
- M-0168 — Corpus-wide mutate-hunt sweep over the kernel packages (57c4f8b5)
- M-0169 — Directed wf-vacuity pass over the load-bearing units (e9eb7f4b)
- M-0170 — Firing tests for linter-config rules and the dormant forbidigo fix (337c0869)

## Summary

E-0042 burned down the test-quality debt the G-0259 vacuity audit exposed and
then probed the corpus those chokepoints guard. The firing-fixture-presence
meta-gate's `grandfatherDark` ledger went from 43 entries to its irreducible 1
(`fsm-invariants`, the lone structure-auditor) — every policy now has mechanical
evidence it can fire (M-0166), and the linter-config rules grew the same firing
discipline, reviving the dormant forbidigo enforcement (M-0170) and removing a
redundant policy now covered by gocritic (M-0167). With the chokepoints proven
live, the two complementary probes ran over the kernel: gremlins mutation
testing (M-0168, probe 1) and a directed wf-vacuity assertion-shape audit
(M-0169, probe 2). M-0168's scope shifted mid-flight — the sweep surfaced ~210
survivors against a plan that assumed far fewer, so AC-1 was re-cut to a
value-tiered pass (efficacy baseline + high-value kills + documented noise)
rather than 100% line-by-line disposition; 11 real survivors were killed and 3
vacuous assertions strengthened, each confirmed by injecting the exact bug and
watching the test go red.

## ADRs ratified

- none

## Decisions captured

- D-0025 — Policy subsystem stays bespoke; one linter cleanup excepted (M-0167)

## Follow-ups carried forward

- Deferred kernel mutate-hunt survivors — `internal/verb` (89 lived, efficacy
  86.2%), `internal/check` (45, 88.5%), and 5 in `internal/gitops` — recorded by
  class with counts in `docs/pocv3/m0168-mutate-hunt-survivor-disposition.md`.
  Killing them needs full apply/projection/check fixtures, not the cheap
  pure-function tests this epic targeted; left as a documented value-tiered
  deferral (no gap filed). The efficacy baseline is the objective floor a future
  pass would improve against.
- G-0264 — addressed but not yet archived (`terminal-entity-not-archived`
  advisory); routine `aiwf archive --apply` sweep.

## Doc findings

Scoped doc-lint over the epic change-set (the two new `docs/pocv3/` records and
this artefact): clean — every `M-/G-/E-/D-/C-/ADR-` reference is canonical-width
and resolves to a real entity; referenced file paths resolve.

## Handoff

The chokepoint corpus is now non-vacuous: every policy and the load-bearing
linter rules have firing evidence, and the kernel's test assertions have an
objective mutation-efficacy floor (entity 85.5% / gitops 91.9% / verb 86.2% /
check 88.5%) plus a vacuity-audit pass. The next pass that wants stronger kernel
tests starts from the M-0168 disposition record's deferred-survivor list.
Deliberately left open: the second-tier (renderer/CLI) mutation sweep and the
deferred kernel survivors above.
