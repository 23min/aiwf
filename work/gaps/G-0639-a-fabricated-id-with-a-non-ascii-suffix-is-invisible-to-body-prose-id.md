---
id: G-0639
title: A fabricated id with a non-ASCII suffix is invisible to body-prose-id
status: open
priority: medium
---
## What's missing

`body-prose-id`'s `idTokenPattern` (`internal/check/body_prose_id.go`) matches
`[A-Za-z0-9_]` after the kind prefix. A fabricated id whose suffix is non-ASCII
therefore never becomes a candidate token: it is never classified, never judged
against the strict form, and produces no finding at any severity.

Measured 2026-08-26 on `main`:

```
grep -rnP '\b(?:E|M|G|D|C|ADR)-[^\x00-\x7F]' work/gaps/*.md work/decisions/*.md \
  work/epics/*/epic.md
```

returns two lines in the active tree, while `aiwf check` reports the tree clean:

- `G-0235`'s body carries `G-α`, `G-ζ`, `G-δ` and `G-η`, four ids naming nothing.
- `G-0181`'s body carries `M-…`, standing in for an unallocated milestone.

Both are the letter-and-placeholder suffix form the repo's prose-id convention bans
outright, and that convention names `body-prose-id` as the check enforcing it.
Archived entities are skipped by the rule, so the occurrences under
`work/epics/archive/` sit outside its reach by design rather than through this hole.

`idTokenPattern` carries a second hole, recorded in G-0369: an id written without its
hyphen is likewise never a candidate. That gap rejects widening the pattern for its
own case, on the ground that dropping the hyphen anchor reintroduces false positives
on tokens like `G7` and `M16`. The reason does not carry here, since widening the
class after the hyphen leaves the anchor in place — so the two holes want different
remedies rather than one shared fix.

## Why it matters

This check is the only mechanical backstop for the prose-id convention, and a clean
`aiwf check` is read as the convention holding. A fabricated id the scanner cannot
see is worse than one it reports: the tree passes while carrying it, so nothing marks
it for repair, and the next reader to meet an id-shaped token has no signal that it
resolves to nothing.
