---
id: G-0624
title: markdownLinkRegex misses titled markdown links, hiding them from both scanners
status: open
discovered_in: M-0317
---
## What's missing

`markdownLinkRegex` in `internal/policies/no_dangling_entity_refs.go` does not
match CommonMark's titled link form, so every destination written that way is
invisible to the two scanners built on it.

The pattern is `\[[^\]]*\]\(([^)\s]+)\)`. The capture stops at whitespace and a
closing paren must follow it, so `[text](path "title")` fails to match at all —
not a partial capture, no match. Measured directly: the bare form
`[x](work/gaps/G-0001-a.md)` captures its destination, and the same link with a
title captures nothing.

`auditDanglingEntityRefs` inherits the blind spot: it scans the named narrative
docs for links at entity files and reports the ones that no longer resolve, so a
titled link is skipped rather than resolved and a rotted one passes.
`markdownLinkPattern` in `internal/policies/design_doc_anchors.go` is a second
pattern over the same shape, and it disagrees: `\[([^\]]+)\]\(([^)]+)\)` matches
the titled form but captures `work/gaps/G-0001-a.md "t"` — destination and title
together, which is not a path either. So one pattern skips the shape and the
other mis-captures it; whichever way this is decided, the two want reconciling
rather than diverging further.

A second CommonMark shape fails differently and is worth deciding alongside it.
The pointy-bracket destination `[x](<work/gaps/G-0001-a.md>)` does match, but the
capture keeps the delimiters, yielding `<work/gaps/G-0001-a.md>` — a path that
can never resolve. So the titled form is skipped silently and the bracketed form
would be reported as broken when it is not.

Nothing is missed today: no tracked document uses either form, measured across
`docs/`.

## Why it matters

The exposure is not the links that exist but the ones a future author writes. The
titled form is ordinary CommonMark that no linter here discourages, and both
consumers fail silently on it: neither reports "unparsed link", they simply
return fewer results. A dangling titled reference in `CLAUDE.md` would pass
`auditDanglingEntityRefs`, and a docs subtree whose work-links all carried titles
would read as unaffected by an `exclude_path` change that in fact shadowed it.

Widening the pattern is not obviously the right fix, which is why this is a
decision rather than a one-line change. It changes what `auditDanglingEntityRefs`
scans, and that check's scope is load-bearing for what it currently guarantees:
more matches means more paths asserted to resolve, and any that do not turn a
currently-green check red. The alternatives are to widen it and absorb whatever
that surfaces, or to leave the shape unread and record that as the boundary —
either is defensible, and picking one is the work.

Should it be widened, the two consumers want one shared pattern rather than two.
They already collided once on a duplicated helper.

## Where to fix

- `internal/policies/no_dangling_entity_refs.go` — the pattern and its consumer.
- `internal/policies/design_doc_anchors.go` — `markdownLinkPattern`, the second
  pattern over the same shape, which resolves the titled form differently.
- `internal/policies/m0317_delegation_test.go` —
  `TestM0317_MarkdownLinkRegexShapes` pins the current behaviour of the first
  pattern, so a decision here rewrites that table.
