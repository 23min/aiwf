---
id: M-0313
title: Retire the prose-assertion corpus over shipped surfaces
status: draft
parent: E-0087
depends_on:
    - M-0312
tdd: advisory
acs:
    - id: AC-1
      title: A phrase assertion over shipped prose fails the policy suite
      status: open
    - id: AC-2
      title: The cross-document citation walk still fails on a dangling reference
      status: open
    - id: AC-3
      title: The dispatch trigger-phrase checks still fail when a phrase is removed
      status: open
---
## Goal

Delete the prose- and heading-presence assertions over shipped surfaces across the
policy suite, preserving the two exception classes D-0070 names and demonstrating that
each surviving exception still fails when its property breaks.

## Context

Measured while scoping G-0596: the policy suite carries a body of test functions that
assert shipped prose still says particular things, spanning several thousand lines, with
no catch recorded across roughly fourteen months and several filed gaps recording drift
it failed to prevent. D-0050 fixed the rule for new tests but declined to retrofit;
D-0070 mandates the retrofit and settles that the disposition is deletion rather than
conversion.

M-0312 removes the mandate that regrows this corpus and must land first. Once it has,
deletion is unconstrained — whole files can go, including the package-level path
constants that only exist to satisfy the old predicate.

## Acceptance criteria

### AC-1 — A phrase assertion over shipped prose fails the policy suite

A policy fails the suite when a test that reads a shipped-surface fixture asserts its
prose contains a particular phrase, naming the file and line. The check runs over **test
source**, not over prose, which is what makes it immune to rewording and distinguishes it
from the class D-0070 retires.

**The allowlist is closed to the two exception classes named in D-0070, and carries no
grandfather entries.** This clause is what makes the deletion real rather than incidental.
Without it the ban is satisfiable by exempting everything that already exists — a move
this repo has precedent for in the firing-fixture gate's own ledger — and the corpus
survives with every acceptance criterion green. With it, a green suite means the corpus
is gone, because the two cannot both be true.

Fixture: plant a long-literal containment assertion in a test reading an embedded-skill
fixture; the policy fires and names it. Removing the plant returns the suite to green.

### AC-2 — The cross-document citation walk still fails on a dangling reference

The check that walks every ritual and fails a section reference naming a heading no
ritual defines survives this milestone intact, and still bites.

It is the one assertion in the corpus with a recorded catch — two dangling citations on
its first run — and D-0050 names its shape as the one to prefer: a relationship between
documents rather than a reading of one. No rewording makes it pass falsely.

Probe: introduce a section reference naming a heading that does not exist, confirm the
walk goes red, revert. An unproven survivor is a finding, not a completion.

### AC-3 — The dispatch trigger-phrase checks still fail when a phrase is removed

The assertions over trigger phrases in a skill's `## When to use` section and its
`description:` frontmatter survive, and still bite.

These pin dispatch behaviour rather than prose style: G-0353's session mining measured
the deployer agent at approximately zero dispatches before those phrasings existed. The
limit is worth restating, because it bounds how much the exception is worth — nothing
mechanical consumes a trigger phrase, so the property rests on an assistant's judgment.
D-0070 keeps the class on the strength of the evidence, not the soundness of the
mechanism.

Probe: remove one trigger phrase from the deployer card's `description:` and one from
`aiwfx-release`'s `## When to use`, confirm each goes red, revert.

## Constraints

- The two exception classes survive intact: cross-document relationship checks, and the
  trigger phrases in a skill's `## When to use` section and `description:` frontmatter.
  A pass that removes either has overshot.
- Disposition is per assertion against D-0070, not per file. Several files hold a
  genuine structural assertion within a few lines of a prose one.
- No test function is deleted merely to make a file pass.
- Every surviving exception is probed: break the property in the source document, confirm
  the test goes red, revert. An unproven survivor is a finding, not a completion.
- Coverage must not regress; the diff-scoped gate names any regression by file and line.

## Design notes

- D-0070 carries the disposition rules, the measurement, and the rejected alternatives
  (convert to shape assertions; limit the retrofit to headings; keep everything and hold
  content at review).
- Heading-presence assertions are in scope for deletion. A heading check exists to scope
  a body assertion; once the body assertion is gone it degrades to asserting the heading
  exists.
- The trigger-phrase exception rests on behavioural evidence rather than a mechanical
  consumer — D-0070 records that limit explicitly.

## Surfaces touched

- `internal/policies/` — the test files asserting over embedded skill, ritual, template,
  agent-card, and guidance prose
- `internal/policies/d5_structure_test.go` — the citation walk, preserved

## Out of scope

- Retiring the exposition-tier design documents that some of these tests lock. Removing
  the lock is in scope; what becomes of the documents is separate work with its own
  decision.
- Any change to the `skill-body-id` check or the shipped-surface id rule.
- Re-pointing the backstop, which is M-0312's deliverable.

## Dependencies

- D-0070, accepted.
- M-0312, done — the backstop must be re-pointed before deletion, per E-0087's
  constraints.

## Coverage notes

Deletion removes test code rather than production code, so the diff-scoped gate should
report no newly-uncovered statements. Where deleting a test does drop the last cover for
a production line, that line is a candidate for deletion in its own right rather than a
reason to keep a vacuous assertion.
