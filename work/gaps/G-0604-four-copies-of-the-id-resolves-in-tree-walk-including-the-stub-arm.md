---
id: G-0604
title: Four copies of the id-resolves-in-tree walk, including the stub arm
status: open
discovered_in: M-0312
---
## What's missing

"Does this id name something in the tree, counting entities the loader could not parse"
is now written out in four places: twice in `internal/check/check.go`, once in
`internal/check/body_prose_id.go`, and once in the skill-edit provenance backstop's
resolver. Each spells the same walk — resolve by current id, fall back to prior ids, then
scan the stub list.

`internal/tree` already owns half of it as `ResolveByCurrentOrPriorID`. The stub arm is
the part that keeps being re-inlined.

## Why it matters

The rule the copies encode is not obvious, and the consequence of getting it wrong is
silent in both directions. Omit the stub arm and a malformed entity reads as a missing
one, which turns a landed commit red over an edit to a different file. Include it without
knowing the id is path-derived and a caller may treat the result as stronger evidence than
it is.

Four copies drift one plausible line at a time, and nothing compares them.

## Options

Extract to `internal/tree` beside `ResolveByCurrentOrPriorID` — a single function that
answers "is this id claimed by anything in the tree", with the stub caveat documented once
at the definition rather than at four call sites.

The work spans `internal/tree` and `internal/check`, so it wants its own branch and review
rather than riding a milestone whose subject is elsewhere. Worth checking during the
extraction whether the four callers genuinely want identical semantics — one of them may
have wanted the narrower "parsed entities only" and got the wider answer by copying.
