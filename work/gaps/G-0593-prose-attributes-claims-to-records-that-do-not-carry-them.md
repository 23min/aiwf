---
id: G-0593
title: Prose attributes claims to records that do not carry them
status: open
---
## What's missing

Nothing catches a claim of the form *record R settles C* where R does not carry
C. `body-prose-id` checks that a cited id resolves, never that the attribution
is true. `skill-body-id` bans ids from shipped surfaces outright. The always-on
guidance requires that an edit adding or restating a claim re-opens every source
that claim cites — which was followed and did not prevent the defect, because
re-opening a source confirms a claim's direction and leaves its scope unchecked.

The defect is not argument in a spec, and the genre instruction added to the
spec templates for G-0592 does not reach it: a stripped, genre-compliant draft
produced a fresh instance.

## Why it matters

A false attribution reads as a settled question. The next reader treats the
cited record as having decided something it did not, and stops looking — the
same failure the no-clearance rule exists to prevent, arriving through a
citation instead of a verdict.

Measured in one session, each refuted by a command against the cited file:

- `D-0052` cited as settling shape-descriptions for repository paths. Its
  Decision is the `skill-body-id` keep-list and id shapes.
- `G-0591` cited as naming a deferral with no destination worse than one to
  declined work. It distinguishes delivered work from declined work and makes no
  such comparison.
- `ADR-0007` cited as carrying a placement test phrased "no meaning outside
  aiwf". Its discriminator is whether a skill primarily surfaces a kernel
  capability.
- `G-0580` cited as recording that the structural-test backstop misses entity
  templates. It names role-agent cards and the verb skills under
  `internal/skills/embedded/`.
- `D-0056` cited as grounding a decline of deferred work. It excludes a defect
  from its declines explicitly.
- `docs/design/oracles.md` cited as requiring an inventory row to declare a
  failure asymmetry. That inventory's columns are oracle, class, judges, fires
  at, on failure.
- `G-0584` cited as leaving a non-vacuous acceptance-criterion form to a future
  milestone. It names `M-0308`'s derivation as the first instance of one.
- `G-0356` cited in a test comment as the ground for the guidance line-budget
  position. It is about bless-mode gate symmetry and carries nothing on budgets
  or ceilings; the commit that addressed it never touched that file.

Two of these were written while repairing the others, in edits where the cited
source had been re-opened in the same action. The last was found by applying the
rule below to the file that implements it.

## Resolution shape

An attribution is written by extraction rather than assertion: pasting the
sentence from the cited record that carries the claim. Where no sentence can be
pasted the claim is wrong rather than merely unsupported. That converts a
judgment about what a record means into a lookup, which is the operation that
succeeds.

Three surfaces carry it. The always-on guidance holds the writing-moment rule,
inside the existing revise-and-re-derive bullet rather than as a new one, since
that bullet is the rule this defect defeats. The spec templates hold the
authoring-moment ban: name the record and what it settles for this work, do not
restate what it argues. `wf-review-code` holds the review-moment check, which
moves the detector below out of a per-dispatch brief and into a standing step.

None of the three is mechanically checkable. A check cannot separate an accurate
attribution from an over-reaching one, so the class stays held at review, and
what ships is a rule that makes the over-reach visible to the reviewer rather
than one that blocks it. The reviewer that caught every instance above was
asked, in its brief, to check each citation against the cited file — that
instruction is the closest thing to a detector this defect has.
