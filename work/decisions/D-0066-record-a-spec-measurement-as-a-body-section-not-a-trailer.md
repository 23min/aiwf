---
id: D-0066
title: Record a spec measurement as a body section, not a trailer
status: proposed
relates_to:
    - E-0085
    - M-0308
---
## Question

A pass that measures an entity's factual claims has to leave something behind,
or a pass that was skipped is indistinguishable from one that ran and found
nothing. E-0085 names the record its one blocking question and lists three
candidate shapes — a commit trailer, a line in the entity body, or an empty
`aiwf edit-body` — without choosing between them.

The choice has to survive a later reader that is a rule rather than a person:
whatever carries the record should be readable without parsing prose, because
E-0085 defers the rule that would read it until use shows what it contains, and
a shape chosen for human eyes now forecloses that.

## Decision

A completed pass writes a `## Spec measurement` section into the body of the
entity whose claims it measured, and lands it with `aiwf edit-body`. The
section's presence is the record; its contents are prose for a human.

A pass that changed nothing writes the section too, saying so. That is what
makes "ran, found nothing" a different state from "never ran" rather than the
same absence.

## Reasoning

Two of the three candidates are not available, which measurement settled rather
than preference.

**A commit trailer has no slot.** `aiwf edit-body` emits exactly the standard
three — `aiwf-verb`, `aiwf-entity`, `aiwf-actor` — via `standardTrailers`. Its
`--reason` flag does not become `aiwf-reason`; it is assigned to the plan's
commit-message body as free prose, and only `authorize`,
`acknowledge-illegal` and `acknowledge-mistag` emit that trailer. Adding one
for measurement is a kernel change, which E-0085 bans outright.

**An empty `edit-body` produces no commit.** Bless mode refuses a working copy
with no diff, and the explicit-content path converges to a NoOp when HEAD
already carries the body. Both routes leave nothing behind, so the candidate
does not record anything at all.

That leaves a body write, and within body writes a heading wins on three counts.
It is the only unit with a parser already in the tree — `scanH2Sections` in the
body check, `extractMarkdownSection` in the policy tests — so a later rule reads
a heading and never prose. It is uniform across the two kinds the seams will
wire: a gap has no template at all and only two required sections, so any shape
keyed to an existing section would fork by kind. And it guarantees a body diff,
which is what gives the no-change pass something for `aiwf edit-body` to commit.

Two alternatives lost. A line under an existing section — `## Validation` on a
milestone — needs no new heading, but gaps have no such section, and a line
sitting among other prose is findable only by reading the prose around it, which
is the property this decision exists to avoid. A fixed-prefix line anywhere in
the body is uniform and greppable, but a prefix match passes because someone
typed the token, which is the vacuity shape G-0584 names, and it mints a
micro-format with no parser here when every other structural surface in the tree
is a heading or a frontmatter field.

## Consequences

The section lands only on entities that were actually measured, so it is
evidence rather than scaffolding. That is why it does not cut against G-0530's
pruning of milestone-spec sections, which targets template sections every
milestone carries whether or not they hold anything.

Nothing validates a closed set of body sections — the entity-body rule checks
that required sections are present and non-empty, never that others are absent —
so the section is legal on every kind without a kernel change.

A rule that reads the section stays deferred, per E-0085. What this decision
fixes is that such a rule would have a heading to find rather than a sentence to
interpret.
