---
id: M-0304
title: Classify every catalog scenario against the named-scenario rule
status: draft
parent: E-0080
tdd: required
---
## Goal

Give every scenario in the catalog an explicit classification under D-0063's
rule — a named scenario is reserved for a verdict some document specifies — and
a chokepoint that keeps a new one from arriving without it.

## Context

The harness exists for workflow legality, and its catalog has drifted away from
that: D-0063 measured five scenarios judging workflow legality against eleven
judging concurrency or fault injection. The drift is not itself a defect — each
scenario was justified when it landed — but nothing records which group a
scenario belongs to, so the ratio moves without anyone deciding it should.

D-0063 gives the rule that makes the question answerable: write a named scenario
when the subject is a verdict some document specifies, not when it is a state
some sequence reaches. States belong to the walker; specified verdicts belong to
named scenarios.

## Approach

Census the catalog against that rule, record each entry's classification beside
its registration, and add a policy test that fails when a registered scenario
carries none. Where a scenario turns out to judge a state rather than a
specified verdict, the classification records that finding — retiring or folding
it into the walker is a separate decision with its own evidence.

## Acceptance criteria

Each criterion below is asserted by the harness itself, not by review.

## Constraints

- **Classification is recorded, not inferred.** A rule that derives the group
  from a scenario's name or file path would go stale the first time either
  changes, and would let a new scenario inherit a group nobody chose.
- **No scenario is retired here.** A retirement changes what the suite covers
  and earns its own decision entity; this milestone produces the census that
  such a decision would rest on.
- **Lane and classification are orthogonal.** Whether a scenario is tagged is a
  cost decision per `CLAUDE.md`; what it judges is this classification. Neither
  implies the other.

## Design notes

- D-0063's Decision supplies the rule, and its rejection of "more named
  scenarios as the primary route" supplies the reason it matters: new scenarios
  accrete on the concurrency side by default, which is how the five-to-eleven
  split arose.
- The classification also answers a question the widened walker raises. As the
  mutation space grows, a random walk becomes less likely to generate any
  particular composition, so a verdict a document pins deliberately wants a test
  that reaches it deliberately.

## Surfaces touched

- `cmd/stresstest/registry.go` — the catalog and its entries
- `internal/policies` — the chokepoint test

## Out of scope

- Retiring, folding, or rewriting any scenario.
- Changing any scenario's build-tag lane.
- Adding scenarios. This milestone classifies what exists.

## Dependencies

- D-0063, accepted. No milestone dependency: the census reads the catalog as it
  stands and does not touch the oracle.

## References

- D-0063 — the named-scenario rule and the measured five-to-eleven split.
- G-0468 — the prior catalog-wide correction, on oracle shape rather than
  subject.
- `CLAUDE.md` — the lane rules this classification is orthogonal to.

## Work log

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
