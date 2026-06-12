---
id: G-0241
title: BodyProseIDIndex skips TrunkIDs; trunk-only ids appear unresolved
status: addressed
addressed_by_commit:
    - 56eba164
---
## What's missing

`check.BodyProseIDIndex(t)` (added by G-0184) builds the id-resolution
index from `t.Entities` and `t.Stubs` only. The tree loader also
surfaces `t.TrunkIDs` — ids known to exist on the configured trunk ref
but not in the current working tree (used by the `ids-unique` rule's
trunk-collision detection per G37). The body-prose-id index does not
consult this slice.

Consequence: a body that references a trunk-only id fires
`body-prose-id/unresolved` on a feature branch even though the id IS
valid on trunk. Typical case: a fresh clone where the operator
short-cuts past the initial trunk sync, or a feature branch that's
fallen far behind trunk.

## Why it matters

The base G-0184 patch has this gap. The G-0184 verb-time follow-up
AMPLIFIES it: verb-time refusal is now the chokepoint, so a body
referencing a trunk-only id refuses to write at all. Before the
verb-time patch, the operator got a confusing finding at pre-push; now
they get a confusing refusal at verb time, with no clear remediation
("rebase against trunk" is correct but not in the hint).

Reviewer-pass evidence: surfaced as track-for-later T3 across both
G-0184 reviewer passes. Pre-existing, not introduced by the verb-time
patch; deferred each round to keep patches scoped.

## Direction

Two viable shapes, both targeted at `BodyProseIDIndex`:

- **Add TrunkIDs to the index.** Each `TrunkID` carries an id + path
  string but no full Entity struct. The index value type would have
  to widen to accept a "trunk-known" marker rather than `*Entity`, or
  the index could store a synthetic `Entity` stub per trunk id (id
  set, path set, everything else zero). Either keeps the lookup
  cheap; both bend the index's "real entity" semantics a bit.

- **Add a second-tier resolver.** `classifyBodyToken` checks the
  primary index first (active + stubs); on miss, checks a "trunk-id
  set" (just `map[string]bool`). Cleaner separation. Hint refinement:
  if the trunk-id set contains the token, change the finding's hint
  to "this id exists on trunk; rebase against trunk to pick up the
  entity" instead of the generic "check the spelling."

Recommend the second-tier resolver — keeps `BodyProseIDIndex`'s
"real entity" semantics intact, surfaces a useful per-case hint.

## Test surface

- Positive control: a body referencing a trunk-only id is silent
  when the trunk-id set contains it.
- Negative: a body referencing a truly-unknown id still fires
  `unresolved` (the new tier doesn't mask the existing case).
- Hint refinement: when the trunk-id branch catches the token, the
  hint mentions trunk-rebase explicitly.

## Source

G-0184 reviewer passes: track-for-later T3 (initial pass, post-merge
review) and T3 (verb-time pass).
