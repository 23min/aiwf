---
id: G-0669
title: The two embedded-tree ban policies duplicate one walk and one rule shape
status: open
discovered_in: E-0091
---
## What's missing

`internal/policies/embedded_no_work_log_section.go` and
`internal/policies/embedded_rituals_tracking_doc.go` are the same policy with
different literals. Both walk a tree of shipped markdown, scan it per line, and
apply two rules in the same order: a strong literal that always reports, then a
weaker phrase that reports unless the line carries an escape word. Both build
their violations the same way. Neither reads anything the other does not.

The two exist because each was written for a retirement in progress — the v1
tracking-doc convention, then the milestone spec's `## Work log` section — and
the second was written by reading the first. That is the point at which a shape
gets extracted rather than copied a second time.

They also disagree about scope, and only one of them is right.
`PolicyEmbeddedRitualsNoRetiredTrackingDoc` walks
`internal/skills/embedded-rituals/` alone; `PolicyEmbeddedNoWorkLogSection` walks
every `internal/skills/embedded*` tree. The narrower scope is a hole rather than
a decision: the stale references that motivated the newer ban included two in the
`aiwf show` verb skill under `internal/skills/embedded/`, which the older ban
cannot see. A tracking-doc reference reintroduced into a verb skill today reports
nothing.

## Why it matters

Two copies of one walk drift a plausible line at a time, and the drift here is
already measurable: one of them scans a tree the other does not, for no stated
reason. Extracting the shared walk closes that by construction — the older ban
gains whole-tree scope as a consequence of sharing the traversal, not as a
separate change someone has to remember to make.

The cost is measured rather than estimated. A reviewer wrote the extraction
during the G-0530 review and ran the gates on it: the pair goes from 372 to 249
lines with the whole policy suite green, and the rewritten tests kill a mutant
the submitted ones did not.

This is deliberately not folded into G-0530's patch. Widening a shipped
chokepoint's scope can surface findings in trees it has never read, which is its
own review rather than a footnote to a retirement.
