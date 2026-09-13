---
id: D-0090
title: Absent and empty required sections get separate codes
status: proposed
relates_to:
    - E-0083
    - E-0084
    - ADR-0042
    - ADR-0043
    - D-0051
---
> **Date:** 2026-09-13 · **Decided by:** human/peter

## Question

Two unbuilt rules each report a body section that is not what the kind
requires. E-0084's push seam reports one that is **absent**; E-0083's
readiness rule reports one that is **present and empty**. ADR-0043 and
ADR-0042 both defer the same question to whichever lands first: does one
finding code serve both properties, or does each get its own?

It is not obvious because the two read as one concern — "the body is
incomplete" — and D-0051 settled a question of this exact shape in the
opposite direction, extending one rule rather than adding a sibling.

## Decision

Each property keeps its own finding code. E-0083's readiness rule extends the
existing `entity-body-empty`, whose meaning it does not change — only when and
how hard it fires. E-0084's push seam takes a new code naming absence. Net
growth in the taxonomy is one code, not two.

## Reasoning

D-0051's rubric decides it, applied rather than overridden: what settles the
shape is whether the two share a remediation. There they did — both fixes were
*write the canonical placeholder* — and one hint served both. Here they do not.
Absence is fixed by adding the heading; emptiness by writing prose under a
heading already present.

The two also never fire together. A draft carrying every required heading and
no content satisfies membership and is exactly what E-0083 permits until
readiness is claimed, so the same body is legal under one rule and illegal
under the other depending on status. A shared code would name a state that is
simultaneously fine and not fine.

Sharing is worse than merely imprecise here, because the two remediations are
mutually escaping: deleting a heading satisfies the emptiness refusal, and
adding an empty one satisfies the absence refusal. `requireCompleteBody` in
`internal/verb/add.go` already carries a comment requiring its two messages to
differ for that reason. One code forces one hint to hedge between opposite
fixes, which is the confusion E-0084 exists to close rather than to restate.

D-0051's cost argument does not reach this case. It weighted the shipped
`aiwf-check` findings table heavily because `skill-body-id` is structurally
inert in a consumer repo — the trees it scans do not exist there, so the row
was documentation a consumer could never act on. Both rules here fire on a
consumer's own entities, so each row is actionable and the cost is paid for.

Subcodes under one code were the remaining alternative and lose on churn rather
than on principle. They are idiomatic here — `body-prose-id` carries seven — but
reaching them means renaming `entity-body-empty`, which is consumer-visible, and
the rename lands in the same epic that is already retiring `tdd.strict` from the
shipped tables. It couples two epics' delivery to buy a taxonomy nicety.
