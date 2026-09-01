---
id: G-0661
title: No check caps an in-function comment at the length D-0084 sets
status: open
discovered_in: M-0327
---
## What's missing

D-0084 caps a comment floating inside a function body at eight lines, diff-scoped,
and leaves doc comments uncapped. Nothing implements it. `internal/policies/`
carries `comment_history_attrition.go` and `directive_comment.go`; neither measures
length, and no policy anywhere keys on a line count.

The cap is the half of D-0084 that needs code. Its spec-section half — routing what
a review produces out of the artefact it lands in — is guidance and ritual prose.
The two share a parent decision and nothing else, so the code half can be built or
declined on its own terms.

## Why it matters

An accepted decision with no implementation reads as settled to anyone who finds it,
and the comment that hits eight lines is exactly the one nobody is watching for.
G-0659 measures the population this would govern; if that measurement is wrong, the
cap is aimed at the wrong thing and this gap should be re-scoped rather than built.

D-0084 records why the ceiling sits above the natural size of the thing rather than
below it, and why a count would be the wrong rule for doc comments. Implementing it
means honouring both.
