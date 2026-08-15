---
id: G-0585
title: Rituals clear a question by reading it
status: open
---
## What's missing

Nine sites across eight shipped skills grant a verdict on the basis of reading.
An independent audit against
`docs/initiatives/milestone-preflight-as-independent-review.md` found them; the
appendix there lists each with its quoted line.

The sharpest are `wf-review-code`'s output template, which ships the line
"something you checked and found sound; not a defect, since you verified it";
`wf-codebase-health`'s Strong / Weak / Missing scorecard, whose adversarial
second pass is itself a read; and its convergence rule — "a review loop is
converged when a fresh reviewer, over the whole surface, finds no defect" —
which is mirrored into the always-on guidance and so binds every turn.

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

The fix is not a warning. `wf-measure-spec` already carries one — "do not read a
clean sweep as an all-clear" — which concedes the state it warns about. Removing
the state is what removes the failure: a verdict vocabulary with no clear-by-
reading value in it, and a separate word for what a command settled.

Not a chore. Deciding which of the nine are legitimate takes judgment per site,
and the always-on guidance is one of the surfaces, so the change wants its own
branch and its own review rather than riding along with other work.
