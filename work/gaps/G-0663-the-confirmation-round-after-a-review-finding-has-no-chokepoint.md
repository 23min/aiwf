---
id: G-0663
title: The confirmation round after a review finding has no chokepoint
status: open
discovered_in: M-0327
---
## What's missing

A review returns findings, the findings are fixed, and the fixes are confirmed. The
confirmation is stated as an obligation and gated nowhere.

In `wf-patch`, "then confirm the fixes" is the second-to-last clause of a single
long paragraph in step 6, and step 8's commit gate enumerates what must be shown to
the human without naming it. In `aiwfx-wrap-milestone`, step 7's commit gate states
neither the review outcome nor the confirmation outcome; both live in step 2's prose.

## Why it matters

A gate is what makes an obligation real. Where the gate's checklist does not name
the confirmation, an unconfirmed fix reaches the human indistinguishable from a
confirmed one, and the record the wrap writes cannot be told apart either.

Sequencing: this lands after an acceptance criterion's surplus prose has somewhere
to go. Gating the confirmation round multiplies review rounds, and doing that over
an input set with no remove path makes the growth worse rather than better.
