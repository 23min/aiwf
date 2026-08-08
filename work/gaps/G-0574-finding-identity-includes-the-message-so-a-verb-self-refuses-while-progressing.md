---
id: G-0574
title: Finding identity includes the message, so a verb self-refuses while progressing
status: open
priority: medium
discovered_in: M-0300
---
## What's missing

The verb-time projection guard reports only findings a mutation *introduces*,
comparing before and after by a key that concatenates the finding's message
text. Several check rules interpolate live state into that message — including
the very field a verb is about to mutate. When they do, a finding that was
already present reads as newly introduced, and the guard refuses a mutation that
was making progress toward clearing it.

Two instances measured, both reachable by a clean merge of two individually
legal branches:

- `acs-tdd-audit` interpolates the AC's `tdd_phase`. From a tree carrying the
  finding at `(absent)`, `aiwf promote <m>/AC-1 --phase red` is refused: the
  post-state message reads `red` where the pre-state read `(absent)`, so the
  same standing finding fails to match itself. Patching only the message to omit
  the phase value lets the whole ladder walk unforced to a clean tree.
- `milestone-done-incomplete-acs` interpolates the list of still-open ACs. From
  a `done` milestone with two open criteria, cancelling either one is refused,
  because the post-state message names a different remaining set.

A verb whose mutation cannot change any interpolated value is unaffected, which
is why an unrelated `aiwf promote <epic> active` succeeds on the same trees.

## Why it matters

The guard's purpose is to stop a verb landing a state the gate would refuse.
Here it does the opposite: it refuses a verb whose effect is to move *out* of
such a state, and it does so for a reason no operator can see, since the two
messages are equal in every respect they would think to compare.

The naive repair is unsafe. Dropping the message from the identity key would
collapse findings that differ only by message: three `entity-body-empty`
findings on one epic share code, subcode, path and entity id, and are
distinguished solely by which section they name. Collapse them and a verb could
introduce a newly-empty section undetected.

What is missing is a decided answer to what makes two findings the same finding
for the purpose of the guard — a stable identity that survives the mutation
under consideration without erasing genuinely distinct instances. That decision
is the prerequisite for both repairs above, and it is currently unwritten.
