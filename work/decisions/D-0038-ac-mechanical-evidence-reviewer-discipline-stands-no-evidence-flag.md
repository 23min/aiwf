---
id: D-0038
title: 'AC mechanical-evidence: reviewer-discipline stands, no --evidence flag'
status: accepted
relates_to:
    - E-0033
    - D-0005
    - G-0140
---
# D-0038 — AC mechanical-evidence: reviewer-discipline stands, no --evidence flag

> **Date:** 2026-07-19 · **Decided by:** human/peter

## Question

D-0005 committed to mechanizing "AC promotion requires mechanical evidence"
via a promote-time `--evidence <test-symbol>` flag, verb-time hard-reject on
absence, and check-time resolution against `go test -list`. Implementation
was scoped out to a follow-up (G-0140). Before building it: does this
mechanism actually deliver the guarantee it's named for, at a cost
proportionate to the milestone it would take?

## Decision

Do not implement D-0005's `--evidence` flag mechanism. AC mechanical-evidence
enforcement stays at review time, via `wf-vacuity` (adversarial check that a
test can actually fail) and `wf-review-code` (AC-coverage discipline) — not a
new kernel verb flag, frontmatter field, or check-time toolchain integration.
D-0005 is superseded; G-0140 is cancelled as a direct consequence.

## Reasoning

- **The mechanism's core guarantee is thinner than it looks.** Check-time
  resolution against `go test -list` proves a named symbol exists in the
  compiled test binary — not that the test exercises the AC's claim.
  `--evidence TestSomeUnrelatedThing` satisfies the gate. D-0005 rejected
  alternative E (grep for the AC id in test files) as "illusion of
  enforcement"; mechanism D has the same gap one level removed — existence,
  not relevance, and relevance is what actually matters.
- **Cost is out of proportion to what it buys.** The full scope per D-0005's
  own follow-up note — repeatable evidence list, frontmatter field, commit
  trailer, `go test -list ./...` check-time integration, and a migration
  verb to backfill ~200+ already-`met` ACs — is a genuine milestone, not the
  ~50-80 LOC G-0140 estimated (that estimate covered roughly one of five
  scoped pieces).
- **Go-toolchain coupling cuts against the kernel's stack-agnostic posture.**
  `go test -list` is Go-specific; aiwf ships stack guidance for Python,
  TypeScript, C#, Rust, and Elixir too. Baking one language's test-discovery
  command into a general-purpose kernel check means non-Go consumers get
  zero benefit from half the mechanism.
- **The relevance judgment the flag can't make is exactly what already-
  shipped skills make.** `wf-vacuity` breaks the implementation and confirms
  a real test goes red; `wf-review-code` checks AC-coverage discipline with
  judgment a string-match can't replicate. The gap this closes isn't "the
  verb lacks a flag" — it's "these skills aren't wired as a mandatory gate
  before AC-met," which is a ritual/process fix, not a schema change.
- **A record-only variant (no check-time validation) was also considered and
  rejected.** Dropping just the `go test -list` staleness check keeps the
  schema/verb/CLI cost but produces a field nobody queries today — the
  promote commit's `--reason` text already gives the same paper trail for
  free.

## Consequences

- G-0140 is cancelled — the fix, as scoped, will not be implemented.
- The legal-workflow spec table (`internal/workflows/spec/rules.go`) drops
  the `self.evidence` precondition from both `AC.open.promote → met` Legal
  cells (keeping the `parent.tdd` split, which independently encodes
  G-0153's discipline) and removes the Illegal companion cell
  (`ac-evidence-missing`) entirely.
- Corresponding test-only plumbing is removed in the same follow-up:
  `internal/workflows/spec/evaluate.go`'s `self.evidence` predicate case,
  `deferredImplErrorCodes` in `m0123_ac5_drift_test.go`, the
  `ac-open-promote` entries in `m0125_negative_driver_test.go`, the
  `self.evidence` special-casing in `m0124_positive_driver_test.go`, and the
  predicate-materialization case in `internal/cellcoverage/fixture.go`.
- No production code changes are required beyond the spec/test layer — the
  `--evidence` flag was never implemented, so there's no runtime behavior to
  unwind.
