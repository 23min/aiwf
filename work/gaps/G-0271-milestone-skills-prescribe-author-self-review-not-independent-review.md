---
id: G-0271
title: Milestone skills prescribe author self-review, not independent review
status: open
discovered_in: M-0171
---
## Problem

The milestone lifecycle skills are internally inconsistent about review — but
NOT because independent review is missing. `aiwfx-wrap-milestone` step 2
("Independent two-lens review — before the wrap") already prescribes dispatching
a fresh-context reviewer running `wf-review-code` *and* `wf-rethink` over the
milestone change-set before closure. The independent pass is in the lifecycle.
The defects are two seams that let an operator (human or agent) believe review =
author self-review and never reach that step, or reach it with nothing to read.

1. **Framing + no forward reference.** `aiwfx-start-milestone` step 7 frames the
   pre-completion review as author "self-review" ("run through the
   `wf-review-code` checklist *mentally*"), and step 8 declares completion and
   hands off to wrap without forward-referencing that wrap step 2 performs a
   *prescribed independent* two-lens review. An operator reading only
   start-milestone reasonably concludes review is the author's self-assessment
   and is done. (This gap was itself filed on that misreading: the wrap skill had
   not been consulted before claiming the lifecycle lacked independent review —
   corrected here.)

2. **Commit-timing contradiction.** start-milestone step 8 instructs "do not
   commit the implementation yet" (wrap bundles implementation + spec updates +
   closure into one approved sequence), but wrap step 2 reviews
   `git diff <base>..HEAD` — assuming the implementation is *already committed*.
   Under the hold-until-wrap model the independent reviewer has no committed diff
   to read; it must review the working-tree diff (tracked changes + untracked new
   files) instead. The two skills disagree on when implementation is committed,
   which undercuts the very step that is supposed to review it.

## Direction (converge at the milestone)

Make the lifecycle self-consistent:

- start-milestone step 7/8 forward-references wrap step 2 explicitly — "self-
  review now; an independent two-lens review runs at wrap before closure" — so an
  operator knows the author pass is not the last word.
- Reconcile commit timing: either wrap step 2 reviews the working-tree / staged
  diff under the hold-until-wrap model, or start-milestone commits the
  implementation incrementally so `git diff <base>..HEAD` is meaningful at wrap.
  One model, stated in both skills.

The property: a reader of either skill comes away with the same, accurate picture
of when implementation is committed and when independent review happens.

## Provenance

Surfaced during M-0171's wrap (E-0043). The implementing agent claimed — to the
human, and in this gap's original text — that the lifecycle lacked a prescribed
independent review; loading `aiwfx-wrap-milestone` revealed step 2 already
prescribes it, and exposed the two real seams above instead. The independent
two-lens review the human requested was, in effect, wrap step 2 performed early.
