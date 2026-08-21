---
id: D-0070
title: Retire prose-content assertions over shipped surfaces
status: accepted
relates_to:
    - D-0050
    - G-0596
---
> **Date:** 2026-08-21 · **Decided by:** human/peter

## Question

D-0050 settled what a *new* structural test over ritual prose may assert: document
shape, never that a paragraph still says a particular thing. It deliberately declined
to retrofit the existing suite, on the grounds that retrofitting needed its own
justification and that a decision silently invalidating passing tests is one nobody
can act on.

G-0596 supplies that justification. But the measurement behind it reaches further than
D-0050's rule does, and raises a question D-0050 did not face: is "assert the shape
instead" the right retrofit target, or are the shape assertions themselves ceremony?

## Decision

Prose-presence assertions over shipped surfaces are **deleted, not converted**. The
scope is the surface set the `skill-body-id` check already scans: skill and ritual
`SKILL.md` bodies and their frontmatter, entity templates, role-agent cards, and the
always-on guidance fragment.

Heading-presence assertions are included in the deletion. A heading check exists to
*scope* a body assertion; once the body assertion is gone it degrades to "this heading
exists", which is cheaper ceremony rather than a different kind of thing.

Two exceptions survive:

1. **Cross-document relationship checks** — a check that resolves a reference in one
   document against a target in another, and fails when the target does not exist. The
   exemplar is the cross-skill citation walk that fails a section reference naming a
   heading no ritual defines. It covers the whole tree rather than one change, and no
   rewording makes it pass falsely.
2. **Trigger phrases** — the phrasings in a skill's `## When to use` section and its
   `description:` frontmatter that decide whether an assistant reaches for the skill at
   all. These pin dispatch behaviour, not prose style.

Conversion is not a third disposition. Where an assertion would survive only by being
rewritten as a shape check, it is deleted.

## Reasoning

Measured over `internal/policies` on the date above: 123 of 904 policy test functions
assert over shipped prose, spanning roughly 4,800 lines. Searching the repository
history surfaced no instance of one of them catching prose drift. It surfaced the
inverse repeatedly — drift that reached a consumer and had to be filed, G-0504 being
the clearest case, where ritual and guidance drift read as healthy. The catches that
are recorded came from kernel checks, not from these assertions.

The two exceptions are the cases where evidence points the other way. The citation walk
caught two dangling references on its first run. The trigger phrases have behavioural
evidence: G-0353's session mining measured the deployer agent at approximately zero
dispatches before those phrasings existed. Worth stating plainly, because it bounds how
much the second exception is worth: nothing mechanical consumes a trigger phrase. The
consumer is an assistant's judgment, and per the repo's own rule a guarantee resting on
model behaviour is not a guarantee. The exception is kept because the evidence is real,
not because the mechanism is sound.

Why deletion rather than D-0050's own middle disposition:

- **Convert to shape assertions.** Keeps the test, and with it the two-file-edit-requiring-Go
  friction that G-0596 names as the actual cost. It also produces heading-presence checks
  over headings nobody has broken. D-0050 already found that the defect rate scales with
  the number of assertions; converting holds the count roughly constant.
- **Limit the retrofit to headings.** A defensible floor, and better than the status quo,
  but it lowers the tax rate rather than removing it, and does not reach the stated goal
  of fewer tests.
- **Keep everything, hold content at review.** This is what the corpus already does in
  practice, since it catches nothing — but it charges for the appearance of a gate. A
  green run that means less than it appears to is the failure D-0050 named.

## Consequences

- A careless edit can weaken shipped prose with nothing going red. This is the trade
  D-0050 already accepted; this decision takes it at full scope.
- Reviewers of ritual-content changes carry more weight. The wrap and patch rituals
  already dispatch an independent reviewer for exactly that.
- Deleting the corpus does not stop it regrowing. The mandate that generates it is the
  subject of the companion decision on the skill-edit backstop, and this decision is
  worth less without it.
- The surfaces describing the structural-test discipline — CLAUDE.md's ritual-authoring
  and enforcement sections chief among them — need reconciling with the narrower rule.
- G-0596's scope is a strict subset of this one and is addressed by the same work.

## Provenance

Decided while scoping G-0596's retrofit, after the first batch showed that the brief's
delete/convert/keep triage would spend its most expensive half on conversions a fuller
reading of the evidence deletes. The measurement that prompted the widening: the
assertion corpus has no recorded catch, while the mandate that generates it is traceable
to a growth curve that steepens the month it landed.
