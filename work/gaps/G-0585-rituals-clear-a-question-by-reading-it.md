---
id: G-0585
title: Rituals clear a question by reading it
status: open
---
## What's missing

Shipped skills grant a verdict on the basis of reading, at every site the
appendix of `docs/initiatives/milestone-preflight-as-independent-review.md`
lists with its quoted line. That list is a reading, not a census: the same
clearance phrase recurs across surfaces, so the enumeration is a floor. Work on
this gap re-derives the sites rather than trusting the count.

The sharpest are `wf-review-code`'s output template, which ships the line
"something you checked and found sound; not a defect, since you verified it",
and `wf-codebase-health`'s Strong / Weak / Missing scorecard, whose adversarial
second pass is itself a read.

The always-on guidance is not among them, and neither is `wf-codebase-health`'s
convergence rule. Both terminate on *disposition* — "no defect that is not
already pinned, recorded, or tracked", explicitly "not 'no findings ever'" —
which is a stop rule rather than a clearance, and is the form the listed sites
should be repaired toward.

Several skills mix legitimate and illegitimate clearances under one word.
`wf-vacuity`'s mutation probe runs a command and earns its verdict; its
tautology probe is pure reading, and both land in the same `## Clean` section.
`wf-doc-lint`'s "No findings" summary covers two checks that run commands and
two the skill itself describes as heuristics. `wf-tdd-cycle`'s branch-coverage
audit is explicitly "agent-performed — not a tool invocation" and is declared
downstream as "branch-coverage audit clean".

The distinction that separates them is whether a command earned the verdict.
Where one did, the clearance is sound. Where reading did, it is not.

## Why it matters

A clearance earned by reading is worse than no review of that claim. Someone who
has not checked knows they have not checked; someone holding a report that says
verified has stopped looking. This was measured rather than reasoned: a blind
audit of a cancelled milestone read the decision record that named the relevant
templates and reported the spec's central factual claim sound. The claim was
false, and nothing written anywhere could have settled it — only running the
verb could. The reviewer did not merely miss the defect, it marked the claim as
checked.

The fix is not a warning. A warning that says "do not read a clean sweep as an
all-clear" concedes the state it warns about, and the reader who needed the
warning is the one who will not heed it. Removing the state is what removes the
failure: a verdict vocabulary with no clear-by-reading value in it, and a
separate word for what a command settled.

The distinction that separates a sound clearance from an unsound one has a
second consequence, and it splits the repair in two. Where no command was run,
the vocabulary change is the whole of it. Where one was run and it cannot fail,
the reviewer complies with the rule and still reports a clearance, so the word
is not what is wrong. G-0660 is that shape: the wrap ritual's compression
question requires the cut applied and the gates run, and the gates it names have
no failing state for a deleted test or a removed guard. Renaming the verdict
there leaves the reviewer running the same command. The repair is to route the
site to an instrument that can fail — and this gap already names one, in
`wf-vacuity`'s mutation probe, as a verdict a command earns.

Not a chore. Deciding which sites are legitimate takes judgment at each one, so
the change wants its own branch and its own review rather than riding along with
other work.
