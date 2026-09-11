---
id: G-0659
title: A review finding is recorded twice, and the spec copy is the one that drifts
status: open
discovered_in: M-0327
---
## What's missing

A review finding is resolved by an artefact — a check that fails without the fix,
a decision record, or a gap — and the commit that carries the fix explains why in
its message. Nothing says that is the whole record, so the reasoning is also
written into the milestone spec, where it becomes a third copy of a fact the
check and the commit already hold.

The wrap ritual states the right rule once and then licenses the opposite in the
same sentence: a defect pinned by the check landing with it takes no gap, "note
it under `## Reviewer notes` if the reader needs the reasoning, and nowhere if
the check speaks for itself". The second clause is the correct default and reads
as the exception.

Measured on M-0327, whose review ran ten rounds: `## Reviewer notes` held 602
words, of which six of seven bullets restated a rule already pinned by a test
that dies under mutation. The corrective commit bodies carry 886 words saying
what was wrong and why. The acceptance criteria show the same shape at
sub-entity granularity, and it is repo-wide rather than local to this milestone.
Of 225 criteria already substantial when promoted `met`, 41 grew afterwards;
across those, 3,749 words were added against 1,377 removed, and 8 removed
essentially nothing at all. Under a third of the growth lands after `met` — the
rest arrives between `add` and `met`, so a rule keyed on the `met` promote would
watch the smaller half.

Where the added words belong is the sharper defect. The largest case adds 265 to
M-0313/AC-1, describing what that criterion's check does not recognize: a
`cmp.Diff`, a regexp match, a comparison against a literal with no call in it.
`internal/policies/shipped_prose_assertion.go` runs to 903 lines and documents
none of them. The knowledge is real, and it is filed where the archive will take
it while the file a reader opens stays silent.

The growth is self-reinforcing. Each added paragraph is more surface for the next
round to find a false claim in, and every round of this milestone's review found
claim defects in prose written by the round before.

The same shape appears in code comments, unbounded. Measured across `internal/`
at HEAD: 4,513 production comment blocks, of which 1,027 exceed eight lines and
account for 17,426 lines; the longest is 115. 85% of those sit against a
declaration, where Go's own convention puts a symbol's contract; 158 float inside
function bodies, where nothing distinguishes an explanation from an argument.
M-0327 added 177 comment lines to production Go, 119 of them in blocks of ten or
more.

Nor is drift the dominant failure. Of sixteen claims of this milestone's that
review overturned, three were true when written and went stale; thirteen were
never true. The first class needs a re-derivation trigger when a rule changes.
The second needs something at the moment of writing, and nothing distinguishes
them today — both are repaired the same way, so the authoring failure is
recorded as maintenance and never addressed as itself.

## Why it matters

The third copy is the one nothing re-derives, so it is the one that drifts — and
drifted claims in a milestone spec are what a later reader trusts, because the
spec is what the archive keeps. A test that stops being true goes red; a
paragraph that stops being true is read.

It also costs review time at the point it is scarcest. Eight rounds found 26
defects in prose against 7 in code; the prose was generating most of its own
review burden while the behaviour it described was already sound.

What has no other home, and is worth writing, is the opposite record: the attacks
a reviewer made that did not break anything. That is what lets the next round
skip ground already covered, and it does not accumulate, because each round's
answer supersedes the last.
