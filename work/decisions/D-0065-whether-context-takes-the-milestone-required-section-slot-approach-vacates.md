---
id: D-0065
title: Whether Context takes the milestone required-section slot Approach vacates
status: proposed
relates_to:
    - M-0305
---
## Question

`Approach` leaves the milestone required set in M-0305, because it entered that set
by transcription rather than by decision. The shipped milestone template's second
section is `## Context`. Should the required set name `Context` in the vacated slot,
or should it shrink to `Goal` and `Acceptance criteria`?

## Decision

Proposed, not ratified: shrink to `Goal` and `Acceptance criteria`, and do not
promote `Context` into the required set.

## Reasoning

The case for `Context` is that it is the section the authoring surface actually
asks for. Requiring it would cost nothing on the day it lands — every milestone
drafted from the shipped template already carries one — and the required set would
for once describe the template rather than contradict it, which is the whole defect
M-0305 exists to close.

The case against is that "what exists before this milestone" is the part of a spec
most readily reconstructed from the tree. The parent epic states it, the
`depends_on` edges state it, and `aiwf history` states it. A required section whose
content is derivable is a second copy of a fact the kernel already holds, and the
copy is the one nothing re-derives.

What survives that test is the pair a reader genuinely cannot reconstruct from
anywhere else: why the milestone exists, and what would make it done. That argues
for the smaller set.

This is worth deciding rather than defaulting, because the required set is the one
surface that has already drifted from every authoring surface once, and the cost of
being wrong is asymmetric. A section wrongly required manufactures findings against
entities nobody was worried about. A section wrongly omitted costs only that the
template offers it and some authors decline.
