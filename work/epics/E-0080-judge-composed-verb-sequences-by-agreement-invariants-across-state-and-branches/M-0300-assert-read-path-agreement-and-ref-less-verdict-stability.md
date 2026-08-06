---
id: M-0300
title: Assert read-path agreement and ref-less verdict stability
status: draft
parent: E-0080
tdd: required
---
## Goal

Give the stress harness an invariant oracle — properties that must hold in every
reachable state — starting with the two that judge a single tree: every read
path agrees on a verdict, and a verdict survives the absence of refs the tree
does not need.

## Context

The `verb-sequence` walker in `cmd/stresstest` composes real multi-step verb
chains and re-checks after every step, which is the right shape. Its oracle
asserts only that `aiwf check` never regresses, and monotonicity cannot catch a
finding carrying the wrong severity for its state.

D-0063 settles the direction: judge with properties that hold in every reachable
state rather than with an expected end state the harness computes. This
milestone builds that oracle seam and the two properties that need no branch
machinery. It deliberately does not widen what a sequence may mutate — D-0063 is
explicit that a wider state space under a monotonic oracle buys reachability
without judgment, so the oracle comes first.

## Approach

Add an invariant-oracle seam to the walker: after every step of a sequence,
evaluate a set of registered properties against the repository that step
produced. Two properties land here. Read-path agreement runs each surface that
renders a verdict over the same bytes and compares the finding sets. Ref-less
stability recomputes the verdict on a copy stripped of refs the tree does not
need, and compares.

Neither property knows the correct verdict. Both compare two observations of the
same bytes, which is what makes them cost nothing per axis once the mutation
space widens.

## Acceptance criteria

Each criterion below is asserted by the harness itself, not by review.

## Constraints

- **The oracle stays invariant-shaped.** No property computes an expected
  verdict, and none embeds the reference-resolution tier rules of ADR-0030 or
  ADR-0041. A second implementation of the kernel inside the harness would drift
  against the first and be wrong in a way no test catches, because it *is* the
  test.
- **No property measures the runner** — not how many actors get through, not how
  fast (G-0468).
- **Both properties fail against the current binary** and turn green when
  G-0558's fix lands. Do not weaken a property to make it pass early, and do not
  merge this milestone to mainline ahead of that fix: the red-to-green flip
  across it is the evidence the oracle works.

## Design notes

- D-0063 is the accepted direction. Its Decision names these properties; its
  Reasoning explains why an invariant oracle costs nothing per new axis while an
  exact-verdict oracle needs a model of the right answer.
- The surfaces read-path agreement compares are the ones G-0558 measured as
  disagreeing: `aiwf check`, `aiwf check --fast`, `aiwf check --shape-only`, and
  `aiwf status`.

## Surfaces touched

- `cmd/stresstest` — the walker and the new oracle seam
- `internal/stresstest` — scenario drivers

## Out of scope

- Widening what a sequence may mutate.
- The branch-independence property. It is unreachable until a sequence can cross
  a branch boundary, so it ships with the scenario that makes it reachable
  rather than as a test no generated condition ever exercises.
- Fixing the disagreement these properties detect — that is G-0558.

## Dependencies

- D-0063, accepted.
- G-0558's fix must land before either property can reach green. That edge is
  prose-only: `depends_on` is milestone-to-milestone and G-0558 is patch-shaped.
  It is the G-0073 ceiling E-0080 names, met on the epic's own first milestone.

## References

- D-0063 — widen the stress walker; keep its oracle invariant-shaped.
- G-0558, G-0556 — the measured instances these properties judge.
- G-0468 — the precedent for holding an oracle to a shape rather than a subject.
- G-0073 — why the G-0558 edge cannot be declared structurally.
- G-0121, E-0062 — the parent gap and the epic that built the walker.

## Work log

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
