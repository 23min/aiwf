---
id: G-0648
title: wrap-epic's gap-closure precondition has no route for the epic's own spec
status: open
---
## What's missing

`aiwfx-wrap-epic`'s precondition 6 opens by scoping itself to two places — "Neither
the epic's own spec nor any milestone's left a gap open that it claims to close" —
and then supplies a mechanical reading route for only one of them: "Read each child
milestone's `## Closes` section and check the status of every id it names".

There is no such section on an epic. `templates/milestone-spec.md` carries `## Closes`;
`templates/epic-spec.md` does not, and its section list runs Goal, Context, Scope, Out
of scope, Constraints, Success criteria, Open questions, Risks, Milestones, ADRs
produced, References. The precondition's closing sentence, "Nothing checks the epic's
own spec earlier", names the epic as the uncovered case without giving the reader a way
to cover it.

So the reader is told to check a surface, told that nothing else checks it, and handed a
route that does not reach it. The fallback the precondition does offer — "A milestone
predating the section has no list to read, so fall back to its body prose" — is written
for milestones authored before `## Closes` existed, not for a kind that has no such
section at all.

## Why it matters

Precondition 6 is the backstop for a gap a wrap left open: `aiwfx-wrap-milestone` closes
what a milestone's own body claims to fix, and this precondition catches what that step
missed. An epic body can name a gap it closes — E-0090's `## References` names G-0646
exactly that way — and no ritual step reads it, so the case the precondition was written
to catch is the case it cannot see.

The failure is silent in the direction that costs most: the tracker keeps a gap open that
the epic's own body says is fixed, and the next reader of that gap has no signal that the
work already landed.
