---
id: G-0521
title: Whiteboard ordering regenerates instead of reconciling against the last one
status: open
priority: medium
---
## What's missing

The whiteboard ritual produces a tiered landscape and a recommended sequence, then writes them to a gitignored cache it overwrites in full on the next run. Nothing carries an ordering forward: a second run re-derives grouping and sequence from the tree alone, so an ordering a human considered and agreed to is replaced by one that merely resembles it.

That leaves a consumer with no stable answer to "what do I do next, in what order". Each planning surface answers a different question and none answers that one — the status snapshot says what is true now, the roadmap says what shape the work has and what is done, the initiatives tree holds ideas that are not yet entities, and the whiteboard thinks about the landscape freshly every time. A consumer who wants a sequence that holds still keeps their own file for it, unaided.

## Why it matters

Re-deriving an ordering from scratch is confidently wrong exactly where prior judgment lives.

Measured on this tree: a from-scratch triage of the gaps open since v0.30.0 collided with E-0074, E-0076 and E-0077, whose specs had already sequenced their own members. E-0077's spec carried a constraint the fresh pass dropped — G-0462's lint-cache fix lands before the duplication inventory is measured, because a concurrent linter makes that instrument report a working rule as dormant. The correction was to defer to the recorded structure and change only what had moved.

An ordering is also where churn is cheapest to bound. Most of what changes between runs is mechanical: entries whose entities went terminal, clusters that emptied. Only a little is judgment. A ritual that regenerates cannot tell the two apart, so it re-litigates the judgment every time it recomputes the mechanics.

## Resolution shape

Treat an existing ordering as the baseline and reconcile against it:

- drop entries whose entities are terminal, and clusters that have emptied, without asking
- propose a placement for each entity new since the ordering was written
- propose a move only where something changed that justifies it, naming the change
- present that as one reviewable delta rather than a series of questions — it is a proposed edit, not a decision gate, and the pending-decision gate's one-at-a-time rule is untouched
- build from scratch only when no ordering exists

A cluster's thesis is the stable unit; membership churns beneath it without moving it. Splitting, merging or inventing a cluster is a structural change and is raised on its own.

The consumer's file is theirs. No verb writes it, no check validates it, and it is not an entity kind — the deliverable is ritual instruction and nothing else.

Also to settle while editing that ritual: its cache anti-pattern permits a gitignored artifact on the grounds that such files regenerate on every invocation. An authored ordering is the opposite case and needs distinguishing there, or the permission reads as covering it for the wrong reason.
