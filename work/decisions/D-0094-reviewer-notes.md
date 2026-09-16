---
id: D-0094
title: Reviewer notes record the review's result, not its rounds
status: proposed
relates_to:
    - D-0054
    - D-0085
    - G-0635
    - G-0659
---
> **Date:** 2026-09-15 · **Decided by:** human/peter

## Question

What does a milestone spec's `## Reviewer notes` hold about the review — what
holds once the review is over, or also an account of how the review went?

Shipped surfaces pull both ways. `aiwfx-wrap-milestone` defines the section as
holding "the review's own outcome", the wording G-0635 set, and asks for a fix
made at wrap to be recorded there after its corrective commit. Against that, the
guidance fragment asks of any sentence that reaches backwards whether a reader who
never saw the earlier version would need it; `wf-review-code` §"Verdict" holds
that a defect "already fixed and pinned needs no further record"; and the
`aiwf-add` skill already keeps "what a review round objected to" out of an
acceptance criterion's body. Specs follow the ritual: M-0330's notes open by
counting its review rounds and what each found, and M-0329's list the attacks
that failed to break the change under a heading addressed to the next round.

## Decision

`## Reviewer notes` records the review's result: each finding declined, with the
reason it was declined; each limit knowingly left in place; each trade-off or
rejected approach, with its reason; and the review's final verdict. It records no
account of the review — not how many rounds ran, what each found, who argued
what, which attacks failed to break the change, or which fixes were committed,
during the loop or at wrap. What a later round needs beyond the declined findings
stays in the review report. This covers `## Reviewer notes` only; what other
entity bodies may say about the work that produced them is not decided here.

## Reasoning

Two readers use the section: a later round of the same review, which the reviewer
agent card sends there for what an earlier round already weighed, and whoever
opens the milestone once the review is over. Both can act on what still holds — a
judgment not to act, a limit and where it stops, the reason an approach lost.
Neither can act on how the rounds went, which is not a property of the change.
This applies D-0054's account of what a record is for to this one section; if
that account is wrong, so is this decision.

The list of attacks that did not break the change is the part of the account with
a use: it lets the next round skip ground already covered. G-0659 counts it worth
writing for that reason, and holds that it has no other home. The review report is
that home — the `wf-review-code` format lists "something you checked and found
sound" under its non-issues, and a later round's brief can carry it forward. A copy
in the spec does that job no better. G-0659 also holds that the list does not
accumulate, since each round's answer supersedes the last; after the last round
nothing supersedes it, and it stays in the archive reading as assurance about code
that may since have moved. The cost falls on a loop whose context is lost between
rounds: a handoff is capped near ten lines, too short to carry the list, so the
next round re-attacks ground already covered.

**Allow the account during the loop and cut it at wrap** — rejected. It keeps the
next-round use inside the spec, but the cut lands after the deciding review, where
unless something sends the spec back through that review no independent reader
sees it, so whether the cut was made goes unchecked. Under this decision a
sentence written mid-loop is in front of every later round.

**Allow both** — rejected. D-0054's tier model prices archived prose as close to
free, so on loading cost the account is cheap. That model prices loading a record,
not what the record is for, and D-0054's own answer to the second question is the
ground this decision stands on. Allowing both would also leave the ritual asking
for the account, so no round could treat it as a finding.

**Extend the drafting-history rule in the guidance fragment to accounts of the
work in every entity body** — rejected. The accounts are observed in this section;
the fragment is paid for on every turn; and a wider rule would have to separate an
account of the work from the dated measurement records the same fragment requires.

## Consequences

- `aiwfx-wrap-milestone`, which the milestone-spec template names as this
  section's owner, changes to match: a fix made at wrap is no longer recorded in
  the section, its references to the review's outcome name the result rather than
  the review, and the declined-finding record stays. Its brief for a later round
  carries what the earlier round's report found sound, since the spec no longer
  does.
- The reviewer agent card still reads the section for declined findings.
- G-0659 holds that the record of attacks that did not break the change has no
  other home; this decision names the review report as that home.
- A later round can hold the section to this rule, since it reads what earlier
  rounds wrote there. What is added after the deciding round is read only if
  something sends the spec back through review.
