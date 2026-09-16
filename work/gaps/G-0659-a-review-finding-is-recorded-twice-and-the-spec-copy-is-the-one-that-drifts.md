---
id: G-0659
title: A review finding is recorded twice, and the spec copy is the one that drifts
status: open
discovered_in: M-0327
---
## What's missing

A review finding is resolved by an artefact — a check that fails without the fix,
a decision record, or a gap — and the commit that carries the fix explains why in
its message. The reasoning is written a second time into the planning tree, where
it becomes a third copy of a fact the check and the commit already hold.

The acceptance-criterion body is where that copy lands. `aiwf-add` §"What to
write per kind" states that a criterion's body leaves out what a review round
objected to, since the check or the decision record that settled it carries it —
and says in the same breath that the rule is advisory, because `aiwf check`
asserts a body's presence, not its structure. Nothing reads a criterion's body
for what it holds.

Measured 2026-09-01, when this gap was filed: of 225 criteria already substantial
when promoted `met`, 41 grew afterwards; across those, 3,749 words were added
against 1,377 removed, and 8 removed essentially nothing at all. Under a third of
the growth lands after `met` — the rest arrives between `add` and `met`, so a
rule keyed on the `met` promote would watch the smaller half. None of those
figures records the command that produced it, which is the class G-0668 tracks.

Where the added words belong is the sharper defect. The largest case adds 265 to
M-0313/AC-1, describing what that criterion's check does not recognize: a
`cmp.Diff`, a regexp match, a comparison against a literal with no call in it.
`internal/policies/shipped_prose_assertion.go` documents none of them:

```
$ grep -c -i 'cmp.Diff' internal/policies/shipped_prose_assertion.go   # 2026-09-16
0
$ grep -c -i 'regexp' internal/policies/shipped_prose_assertion.go
0
```

The knowledge is real, and it is filed where the archive will take it while the
file a reader opens stays silent.

The growth is self-reinforcing. Each added paragraph is more surface for the next
round to find a false claim in, and every round of M-0327's review found claim
defects in prose written by the round before.

The same shape appears in code comments, unbounded. Measured across `internal/`
on 2026-09-01: 4,513 production comment blocks, of which 1,027 exceed eight lines
and account for 17,426 lines; the longest is 115. 85% of those sit against a
declaration, where Go's own convention puts a symbol's contract; 158 float inside
function bodies, where nothing distinguishes an explanation from an argument.
M-0327 added 177 comment lines to production Go, 119 of them in blocks of ten or
more.

Nor is drift the dominant failure. Of sixteen claims of M-0327's that review
overturned, three were true when written and went stale; thirteen were never
true. The first class needs a re-derivation trigger when a rule changes. The
second needs something at the moment of writing, and nothing distinguishes them
today — both are repaired the same way, so the authoring failure is recorded as
maintenance and never addressed as itself.

The milestone spec is no longer a route into this. `## Reviewer notes` holds the
review's result under D-0094, and a defect pinned by the check landing with it
needs no further record. M-0327's own section holds 281 words and names the
check that pins each finding and the commit body that says why as the record:

```
$ awk '/^## Reviewer notes/,0' work/epics/E-0091-*/M-0327-*.md | wc -w   # 2026-09-16
281
```

## Why it matters

The third copy is the one nothing re-derives, so it is the one that drifts — and
drifted claims are what a later reader trusts, because the criterion body is what
the archive keeps beside the code. A test that stops being true goes red; a
paragraph that stops being true is read.

It also costs review time at the point it is scarcest. Across M-0327's eight
rounds, 26 defects were found in prose against 7 in code; the prose was
generating most of its own review burden while the behaviour it described was
already sound.

## Related

- D-0094 settles where the record of attacks that did not break the change lives:
  the review report, not the milestone spec.
- G-0668 records the class this body's own unsourced figures fall into — a number
  written without the command that produced it.
