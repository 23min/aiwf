---
id: D-0084
title: Record a review finding where it resists, not in the milestone spec
status: proposed
---
> **Date:** 2026-09-01 · **Decided by:** human/peter

## Question

A review returns findings. Each one is acted on, and the reasoning behind it has
to go somewhere. Today it goes into the milestone spec, into a code comment, or
into a decision entity, with nothing choosing between them — and the first two
compound, because each round's prose is surface for the next round to find a
false claim in.

Underneath is a question nobody had answered: what makes an artefact a safe place
to put a claim?

## Decision

Route a finding to the artefact that resists a false claim, and record reasoning
where it cannot go stale.

- A **bug**: the fix, plus a check that fails without it. The check is the record;
  the commit body says why. Nothing else.
- A **design choice**: a decision. If it binds work beyond this milestone, its own
  entity. If it explains this milestone's implementation, the spec's
  `## Decisions made during implementation` section, so it forgets when the
  milestone archives.
- **Deferred work**: a gap.
- A **finding declined** that a fresh reader would raise again: a comment at the
  site. If a reader would ask, the code is not self-explanatory there, and the
  answer belongs where the question arises.
- An **attack that held**: recorded with the review outcome. It is a statement
  about review coverage rather than about code, so it cannot drift, and each
  round's supersedes the last.

Prose in a living document describing what the code does is not on that list. That
category is empty.

A comment says what holds and why. Arguing against an alternative at length is a
decision that wandered into a comment. Comments floating inside a function body
are capped at eight lines, diff-scoped; doc comments are not capped, because line
count cannot tell a contract from an argument and a rule that fires on correct
API documentation is one that gets turned off.

## Reasoning

Artefacts differ in how hard they resist. A check is executable, so it is verified
by running rather than reading. A commit message can be wrong on the day but
cannot become wrong. A decision describes a choice, so code moving beneath it does
not falsify it. Prose in a living document fails on both axes at once — it accepts
a false claim when written and drifts as the code moves — and it is where every
claim defect of the milestone that produced this decision landed.

Homing by lifetime rather than by importance follows from measurement: decision
entities do not forget (83 recorded, 5 archived, and 70 sit at `accepted`, which
is not terminal), while milestones do (327 live against 320 archived). Routing
every implementation argument to its own entity would build a corpus that only
grows. Routing it into the milestone grows a file that is already on the forgetting
pipeline, and grows it in the one category that does not drift.

The eight-line cap is not a brevity rule, which is why it is set above the natural
size of the thing rather than below it. An explanation carrying knowledge from
outside the code, or one rejected clause, runs one to four lines. Hitting eight
says a different kind of writing has happened, and forces it somewhere. A ceiling
set below natural size is raised, as this repo's own advisory line budget shows;
one set above it is not, because raising it only hides what hitting it revealed.

Rejected: capping doc comments by the same count. Measured, 85% of the blocks over
eight lines in `internal/` sit against a declaration, which is where Go puts a
symbol's contract and what `go doc` prints. The count is a proxy for "is this
arguing", and substituting a proxy for the thing meant is the error this decision
exists to reduce.

Also rejected: a check that reports a milestone spec growing after its acceptance
criteria are met. Two forms of it were measured against this repo's history at 16%
and 66% false positives; the signal separating rework from ordinary sequential
work is a reviewer's judgment, not anything derivable from timestamps or paths.
