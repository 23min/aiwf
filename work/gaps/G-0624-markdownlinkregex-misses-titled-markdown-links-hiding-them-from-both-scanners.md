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

Two consumers inherit the blind spot. `auditDanglingEntityRefs` scans the named
narrative docs for links at entity files and reports the ones that no longer
resolve; a titled link is skipped rather than resolved, so a rotted one passes.
`linksIntoWork`, in `internal/policies/m0317_docs_link_coverage_test.go`, reuses
the same pattern to decide which `docs/` files link into `work/`, so a file whose
only such link carries a title reads as having none.

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

- `internal/policies/no_dangling_entity_refs.go` — the pattern and its two
  consumers' shared home.
- `internal/policies/m0317_docs_link_coverage_test.go` — `linksIntoWork`, the
  second consumer, whose table is where a titled-link case belongs.
