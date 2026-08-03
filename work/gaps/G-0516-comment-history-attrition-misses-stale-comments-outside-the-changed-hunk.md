---
id: G-0516
title: comment-history-attrition misses stale comments outside the changed hunk
status: open
discovered_in: M-0288
---
## What's missing

Nothing catches a code comment that goes stale because of a change *elsewhere in
the same file*. Two rules cover the comment-history class and each misses this
shape from a different side:

- The pre-push `comment-history-attrition` scan is **diff-scoped to changed
  lines**. A comment describing behaviour that a distant hunk just inverted is
  never inspected, because the comment's own line did not change.
- The whole-tree sibling `PolicyCommentHistoryAttritionTree` inspects every line
  but matches a **fixed phrase set** (`used to`, `previously`, and similar).
  Narration that states a superseded fact in the present tense carries none of
  those markers and passes.

Measured instance: `ScanSkillBodyID`'s contract comment read *"Findings are
warnings while the shipped tree still carries the debris this rule now detects…
The sweep milestone clears the tree and raises the severity as its last act"*
while the emitting line 45 lines below returned `SeverityError`. Both the
diff-scoped scan and the whole-tree policy were green. Only a reviewer reading
the file caught it.

## Why it matters

The comment sat on an **exported function's contract**, which is the surface a
caller consults to decide whether the rule can block CI — so the reader most
likely to trust it is the one most likely to be misled. A severity claim that is
merely absent prompts a check of the source; one that is confidently wrong does
not.

The failure mode generalizes past severity: any comment stating a fact that a
sibling hunk falsified is invisible to both rules, and the further the comment
sits from the change, the less likely review is to reach it. That inverts the
intended safety gradient — the gates are strongest where a human would have
looked anyway, and weakest where only a gate would help.

Resolution shapes worth weighing rather than one pre-picked fix: widen the
diff-scoped scan from changed lines to changed *declarations* (a comment
attached to a symbol whose body changed); extend the whole-tree phrase set to
present-tense claims about severity, blocking, and enforcement, accepting the
false-positive rate that implies; or accept the class as review-held and record
that, since a phrase-grep cannot decide whether a factual claim is still true.
