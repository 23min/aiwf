---
id: D-0081
title: Ratification evidence is the kernel rule, not a policy-test duplicate
status: proposed
---
## Question

The kernel's history audit reports an unratified sovereign act at error
severity, and `aiwf check` at pre-push is what blocks on it. Should a
policy test assert the same property a second time, so it is also gated
in CI?

## Decision

No. The evidence that this repo's history carries no unratified sovereign
act is the shipped `fsm-history-consistent/forced-untrailered` rule
together with the acknowledgment commit that clears it. No policy test
duplicates the rule.

## Reasoning

The duplicate was built and measured before being discarded. Running the
rule over this repo takes 100 seconds — `WalkHeadCommits` accounts for
389ms of it and the acknowledged-SHA walk 219ms, so essentially all of it
is the per-entity history walk. Narrowing the walk to the kinds the
sovereign closed set names, 90 epics out of 1198 entities, moved the total
from 112 to 107 seconds: the cost is not proportional to the entity count,
so scoping cannot recover it.

`internal/policies` is the slowest package in the suite at 40 seconds, and
the suite's wall time is the slowest package. Adding the assertion there
makes every `go test ./...` take roughly 140 seconds instead of 40 — a
cost paid by every contributor on every push, forever, for a property
already gated.

What the duplicate would have bought is narrow. `aiwf check` runs the same
rule at pre-push and blocks on the same finding, so the only uncovered
case is a push made with `--no-verify` or without hooks installed. That
gap is not specific to this rule: `body-prose-id`, `ids-unique` and
`skill-body-id` are all gated the same way and none carries a policy-test
twin. Leaning on the pre-push chokepoint here follows the kernel's own
design rather than making an exception to it.

A third option was rejected outright rather than on cost. The predicate
could be re-derived cheaply from the head walk's trailers, in about a
second, by finding commits with a non-human actor and no force trailer
that move an epic into a terminal status. That is a second copy of
`forcedUntraileredFindings`' predicate, and the two would diverge the
moment either changed — merge-commit handling and the acknowledgment
lookup are both live parts of it. A check that can disagree with the rule
it mirrors reports green while the real rule fires, which is worse than
having no check.

The cost of this decision is that a pinnable claim is left unpinned. It is
recorded here rather than left silent so the next reader meets a judgment
instead of a gap.
