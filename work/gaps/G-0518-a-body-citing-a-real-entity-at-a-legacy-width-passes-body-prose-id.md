---
id: G-0518
title: A body citing a real entity at a legacy width passes body-prose-id
status: open
---
## Problem

`body-prose-id` canonicalizes an id-shaped token before resolving it, so an
entity body citing a real entity at a legacy width passes silently. Measured
across this repo's active entity bodies, a number of citations do exactly that.
(No figure is recorded here: it moves with every body edit, and backticked
tokens are exempt, so the worklist is whatever the rule reports once it exists.)

This is the same defect the doc corpus was swept for, in a surface that
matters more: the entity tree is the planning record, and a body is the
canonical place to reference another entity.

## Why the doc rule does not cover it

`doc-id-width` is width-shaped and scans configured documentation. That is
correct for tutorial prose, where a narrow id is invented fiction and the fix
is a placeholder. An entity body is the opposite: the prose genuinely names a
real entity, so the fix is a widened number, and a placeholder would be a
`body-prose-id` violation in its own right.

The rule this wants is therefore reference-shaped rather than width-shaped —
fire when the token does not resolve as written but its canonical form does.
That distinction also handles the case a pure width rule would get wrong: a
repo that archived before migrating holds genuinely-narrow archived entities,
and a body citing one is correct as written, because read tolerance is
permanent.

## Resolution

Add a `body-prose-id` subcode for a token that resolves only after
canonicalization, and sweep this repo's own bodies. Note the polarity: unlike
the doc rules, backticks stay a legitimate opt-out here, since `body-prose-id`
already treats a backticked token as syntax discussion rather than a citation.
