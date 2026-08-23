---
id: G-0622
title: URL guard skips entity links whose query or fragment contains ://
status: open
discovered_in: M-0314
---
## What's missing

`rewriteLinkDestination` (`internal/verb/linkrewrite.go`) refuses to rewrite any
destination containing `://`, to keep an id inside a URL's path from being
treated as an entity reference. The test runs against the whole destination,
before `splitDestinationSuffix` separates the bare path from a `?query` /
`#fragment` suffix. A destination whose *suffix* carries `://` therefore reads
as a URL even though its bare path names a real entity.

Measured against a move of a milestone between two epics, with the link-region
primitive called directly:

| destination | with the guard | guard removed |
|---|---|---|
| `https://example.invalid/work/epics/<old>/M-NNNN-<slug>.md` | unchanged | unchanged |
| `work/epics/<old>/M-NNNN-<slug>.md?u=https://example.com` | unchanged | rewritten |
| `work/epics/<old>/M-NNNN-<slug>.md#see-https://example.com` | unchanged | rewritten |
| `../epics/<old>/M-NNNN-<slug>.md?ref=git://x` | unchanged | rewritten |

Rows two through four point at the moved entity. No verb rewrites them, so the
link breaks on the move and nothing reports it. The suffix-preserving path the
primitive already documents — a suffix is split off before resolution and
reattached verbatim on a rewrite — is unreachable whenever the suffix itself
contains a scheme separator.

Nothing pins the guard's current breadth: the full suite passes with the `://`
test deleted outright, which is what surfaced this while auditing a neighbouring
change.

## Why it matters

This is the first bullet of ADR-0033 unmet for a narrow input class, in the
shared primitive rather than in one verb — so every mover routed through it
(`archive`, `move`, `reallocate`, `rename`, `retitle`) carries the same hole.

Fixing it is a behaviour change to a shipped surface and wants its own decision
rather than an inline correction: the guard has to distinguish a scheme
separator in *scheme position* from one inside a suffix, which means either
testing the bare path returned by `splitDestinationSuffix` instead of the whole
destination, or matching a scheme prefix rather than a substring. Those differ
for a destination that is relative and carries a colon, so the choice is not
mechanical.

Two properties the resolution should end up pinning, since neither is pinned
today: a URL-shaped destination embedding an entity path stays untouched, and an
entity-shaped destination carrying a scheme separator in its query or fragment
is rewritten with the suffix reattached verbatim.
