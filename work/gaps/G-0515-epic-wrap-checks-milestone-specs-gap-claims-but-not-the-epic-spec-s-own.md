---
id: G-0515
title: epic wrap checks milestone specs' gap claims but not the epic spec's own
status: open
discovered_in: E-0075
---
## What's missing

The epic-wrap ritual checks that no *milestone* left a gap open that the
milestone's own spec claims to fix. An epic spec makes the same kind of claim —
`Addresses G-NNNN` in its Goal or Context — and nothing checks that one.

The two claims fail differently. A milestone's claim is checked twice: by the
milestone-wrap ritual at its own wrap, and again by the epic-wrap backstop. The
epic's claim is checked by neither, so it survives to the wrap artefact as a
`## Follow-ups carried forward` entry, or as nothing at all.

## Why it matters

Measured at E-0075's wrap: every milestone's claim was already discharged —
M-0283 named G-0463, M-0284 named G-0492 and G-0487, and all three were
`addressed` before the epic wrap ran. The only unchecked claim was the epic
spec's own "Addresses G-0466 and G-0463", and G-0466 was still open. It was
closed during that wrap because a human asked what had happened to it, not
because a step surfaced it.

That is the failure shape: a tracker overstating what remains open, discovered
by whoever happens to remember. The epic spec is also the *more* load-bearing
claim of the two, because it is what a reader consults to learn what the epic
was for.

## Sketch

Widen the existing precondition rather than adding a step. It already asks
whether a wrapped unit's own spec claims a gap it left open; the epic's spec is
one more such unit, and the ritual is already reading that spec at wrap.

The disposition sentence needs one correction to survive the widening. Today it
says to close the gap, citing the implementing commit. An epic can legitimately
*advance* a gap without finishing it — E-0075's own spec says G-0480 stays open
on its own terms — so demanding a terminal status would push a wrapper toward
closing something that is still real. The rule is a disposition, not a status:
either close it, or correct the spec's claim to say what actually landed and
what remains. Silence is the only forbidden outcome.

Precondition 6 already has an assertion, which pins its closure command and its
follow-up warning but says nothing about whose claim it covers. The evidence is
that assertion extended rather than a new one: the numbered item must name the
epic's own spec, and the disposition must offer correcting the claim alongside
closing the gap.

The milestone half of the widened precondition is deliberately unpinned.
`aiwfx-wrap-milestone` closes a milestone's claimed gaps at its own wrap, so the
epic-level check is already a backstop there; an assertion guarding it would
guard a backstop of a backstop against an edit nobody has made.

## Cost and retirement

This is +1 rule, so it answers to H3 (*additions carry*). It costs once rather
than per subject — a widened precondition and three checks inside an assertion
that already runs, not a mandate every future epic satisfies with a new
artifact. The test count does not move.

**Owner:** the `aiwfx-wrap-epic` ritual — the precondition and its assertion
live and die together.

**Retires when** gap-closure claims become structured. The claim is prose today
(`Addresses G-NNNN` in a spec body), which is why the check has to be a
human-read precondition backed by an assertion that the prose still says the
right thing. A frontmatter relation naming the gaps a spec claims to fix would
let a check rule read the relation directly and report the open ones, at which
point both the precondition sentence and its assertion are dead weight. Deferred
until a second instance of this failure makes the relation worth the schema
change.
