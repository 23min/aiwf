---
id: D-0065
title: Whether Context takes the milestone required-section slot Approach vacates
status: rejected
relates_to:
    - M-0305
---
## Question

`Approach` leaves the milestone required set in M-0305, because it entered that set
by transcription rather than by decision. The shipped milestone template's second
section is `## Context`. Should the required set name `Context` in the vacated slot,
or should it shrink to `Goal` and `Acceptance criteria`?

## Decision

The question does not hold, and is refused rather than answered. `Approach` left the
milestone required set with nothing taking its place, so there is no vacated slot to
fill: the set is `Goal` and `Acceptance criteria`. That is recorded where it is
enforced — the owned table and the surfaces checked against it — and needs no
decision entity to be true.

Promoting `Context` stays available on its own evidence. It would be a new
requirement, argued from what a milestone body needs, and not from a gap left behind
by a section that was retired.

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

What does not decide it is cost. Requiring `Context` would raise nothing against the
tree as it stands: the rule reports a section that is present and empty, never one
that is absent, and every live milestone carries a filled `## Context`. The only new
findings would be forward, where the scaffold writes the heading and an author leaves
it blank — which is the rule working rather than noise. An argument resting on that
asymmetry decides nothing here, and reaching for it is part of why the question
wanted re-posing rather than answering.
