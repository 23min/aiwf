---
id: D-0080
title: Carry gap-closure claims in a start-time body section, not a commit trailer
status: proposed
---
> **Date:** 2026-08-28 · **Decided by:** Peter Bruinsma

## Question

A milestone or patch that fixes a gap routinely reaches `done` with the gap still `open`. G-0431 shipped a ritual step that closes them at wrap, but the step's *input* is free prose: the wrap reader must determine, by reading a spec, which gaps the work claimed to fix. What mechanism should carry that claim?

The answer was non-obvious because the obvious one — a commit trailer, `Fixes #123` in aiwf's vocabulary — is what every tracker uses, and it turns out to fit poorly here.

## Decision

Carry the claim in a fixed body section, `## Closes`, written at the *start* of the work. The wrap rituals read that section instead of walking prose. Ship it as ritual content only: no verb, no trailer key, no schema change, no check rule.

Status stays `proposed`. The choice is being tried, not committed to.

## Reasoning

The failure was the moment, not the mechanism. Declaring at wrap means reconstructing intent at the point of least context; declaring at start costs nothing, because the gap is usually why the work was started.

A named section is already machine-readable — `aiwf show <milestone> --format json` returns body sections keyed by heading slug, so `body.closes` is queryable today with no code written. Free prose yields nothing reusable. The ritual step is therefore also the on-ramp: if a check is ever wanted, it reads a section that already exists rather than requiring new substrate.

**A commit trailer (`aiwf-closes: G-NNNN`) was designed in full and rejected.** It binds to the work rather than the plan, which is a genuine advantage — it would catch a gap fixed opportunistically mid-milestone without the spec being touched. Three things outweighed it. It is silently lost under squash-merge: measured in a fixture, `git log --format='%(trailers:only=true,unfold=true)'` returns empty on a squash commit, because the concatenated messages indent the trailers out of the final paragraph, and the declaring commits are unreachable from trunk afterwards. aiwf inherits this exactly, since `ParseTrailers` consumes git's extraction rather than the raw message. The resulting failure is a silent false negative — no trailer, so no debt, so a check that reads as "nothing outstanding" when the declaration was destroyed. It also needs several new moving parts to be useful, including a sovereign `acknowledge` subverb, because a wrong declaration on an immutable merged commit is otherwise permanent.

**A `closes:` frontmatter field was rejected** as strictly worse than the section: a schema change to reach data the section already exposes, and a mandate that costs on every milestone forever.

**Deleting the rituals' existing prose steps was rejected.** A check can only enforce what it can prove; the ritual covers the residue — undeclared closures, and gaps no check could pin. Removing them would trade broad-unreliable coverage for narrow-reliable coverage and lose on the difference.

Two measurements shaped the scope. Of 183 open gaps, zero carry `addressed_by` and 87 carry `discovered_in` — which means *found here*, not *fixed here* — so no existing edge in the tree records an intent to fix, and the reverse-reference index cannot supply one. And 16% of open gaps enumerate four or more distinct concerns, so they cannot close in any shape; that is a gap-writing problem this surfaces rather than solves.

## Consequences

The claim stays dependent on a human or assistant writing the section and reading it. The ask shrinks from reconstruction to lookup; it does not disappear.

The section binds to the plan, so a gap fixed opportunistically mid-milestone, without the spec being updated, is missed. Whether that happens often is the open question this decision turns on, and it is what would reopen the trailer design.

`## Closes` is omitted when the work closes nothing, so it is not a required heading.
