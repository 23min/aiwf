---
id: E-0042
title: Burn down test-quality debt across policies and the test corpus
status: proposed
---
## Intent

A test-quality audit (2026-06-19, G-0259) found that 41 of 55 policies in
`internal/policies/` never fire in any test — their `Violation` construction
runs in no test, so there is no evidence they can detect the regression they
exist to catch. The firing-fixture meta-chokepoint landed and seeded a 44-entry
`grandfatherDark` ledger. The same author-grades-own-tests dynamic is not
specific to policies (G-0262): the strength of the assertions guarding the rest
of the module is unknown and has never been adversarially probed.

This epic burns down that debt in four milestones:

1. Firing fixtures for the easy-majority dark policies — each driven to fire,
   each removed from the ledger.
2. Structure-auditor policies (which fire only by mutating a hardcoded Go
   structure): `mutate-hunt` corroboration plus an annotated kept ledger entry.
3. Corpus-wide `mutate-hunt` sweep over the kernel packages — each survivor a
   strengthened assertion or a documented exclusion.
4. Directed `wf-vacuity` pass over the load-bearing units — assertion-shape
   reasoning that mutation testing cannot do.

Milestones 1–2 close the policies-corner portion of G-0259 and G-0262;
milestones 3–4 close the corpus-wide portion of G-0262.

## Closes

- G-0259 — 41 of 55 policies never fire in any test (vacuous chokepoints).
- G-0262 — audit the whole test corpus for vacuous assertions, not just
  policies.
