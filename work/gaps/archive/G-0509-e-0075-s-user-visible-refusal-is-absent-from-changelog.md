---
id: G-0509
title: E-0075's user-visible refusal is absent from CHANGELOG
status: addressed
discovered_in: M-0285
addressed_by_commit:
    - 6249148e21f6ae2caa5967d3f24c991df33094f9
---
## What's missing

E-0075 changes what operators see. A structured-state verb run over an entity whose
file diverges from HEAD now refuses, with two error shapes that did not exist
before: an uncommitted-conflict refusal at the commit seam and a claim-divergence
refusal in the verb's prelude, each naming the diverging paths and a remedy.

`CHANGELOG.md` records none of it. The epic branch carries over 150 commits and not
one touches the file.

## Why it matters

An operator with a hand-edited entity file meets a refusal where the verb previously
succeeded. That is precisely the class the changelog exists for — the same class as
the `Unreleased` entries already there, which cover development-side changes at
greater length.

Left unwritten, the entry is owed at tag time by whoever cuts the release, from a
diff spanning several milestones, months after the reasoning was current.

## Scope

One `Unreleased` entry covering the epic's user-visible delta: what refuses now,
what the two refusals say, and how an operator proceeds (commit the body edit with
`aiwf edit-body`, or set it aside).

Write it at the epic wrap rather than per milestone — the surface is still moving
while milestones under E-0075 remain open, and one entry describing the shipped
behaviour beats several describing its construction.

## References

- ADR-0038 — the two seams and their refusal messages
