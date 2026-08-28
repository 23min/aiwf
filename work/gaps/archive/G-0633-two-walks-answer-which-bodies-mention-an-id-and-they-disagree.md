---
id: G-0633
title: Two walks answer which bodies mention an id, and they disagree
status: addressed
priority: medium
addressed_by_commit:
    - f4912b7a0
---
## What's missing

Two walks answer "which entity bodies mention this id", independently, and give
different answers.

`findProseMentions` (`internal/verb/reallocate.go:605`) reads every entity file,
splits the frontmatter, and matches the raw body against `proseRewritePattern`
(`:297`) — `\b` plus `entity.IDGrepAlternation(id)` plus `\b`, an alternation
over every zero-padded form of the number.

`CitersOf` (`internal/check/citers.go`) performs the same read and split, then
matches `idTokenPattern` against `proseAndCodeMask(body)` and compares through
`entity.Canonicalize`, additionally treating a composite token as naming its
parent milestone.

Neither file names the other. The walk is duplicated; the answers differ because
the matching is not.

Measured 2026-08-24 across the active tree by comparing, per body, the
canonicalized ids reachable in raw bytes against those reachable with markdown
link destinations removed: **four (body, id) pairs where one walk sees a mention
and the other does not** — D-0001 → E-0017, G-0129 → E-0033, M-0066 → E-0016,
and E-0031's link to its own epic directory. In each the only occurrence sits
inside a link destination, which the mask blanks and the raw match does not.

## Why it matters

Not that the two disagree — some divergence is correct, since `reallocate` must
rewrite a link destination it would otherwise break, while a citation notice is
reporting what a body claims. What is wrong is that the divergence is neither
recorded nor tested: no assertion compares the two, and neither call site points
at the other, so whoever next changes the id grammar or the masking convention
has to find both from memory.

The width-tolerance rule is the sharpest case. `IDGrepAlternation` and
`Canonicalize` are two encodings of one rule, and `CanonicalPad` reaches the
second directly while reaching the first only through a regex builder. A change
that lands in one and not the other leaves two functions in one binary
disagreeing about whether a body mentions an entity, with nothing failing.
