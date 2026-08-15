---
id: G-0586
title: The handoff block carries settled findings and drops open ones
status: open
---
## What's missing

`aiwfx-handoff` produces the block an operator pastes into a context compaction,
so it decides what survives into the next session. It carries settled
conclusions and has no slot for an unsettled one.

Its rule for the payload line names "a finding not to re-open" — an instruction
to the next session to treat a question as closed. There is no counterpart
naming a question still open. The block's six lines are: what landed, the next
action, pointers into the tree, branch state, the gotcha, and the decisions
taken. Every one of those is something concluded.

The ten-line cap makes the omission structural rather than incidental. The cap
is right in itself — it is what stops the block duplicating committed state —
but with the current line set it spends its budget entirely on conclusions.

The pointer the block substitutes for detail does not close the hole either. It
directs the reader to `aiwf show`, and no entity template carries a section in
which an unsettled claim would appear.

## Why it matters

A compaction is where a session's working memory is replaced by a summary, so
whatever the block omits is gone rather than merely harder to find. An open
question that survives as "closed" is worse than one that is simply lost: the
next session will not re-ask it.

The failure compounds with the one recorded in the clearance gap. A pass that
can report a claim sound, followed by a handoff that carries findings not to
re-open, produces a later session confident about a claim nobody ever settled,
with no record that it went unchecked.

Fixing it is not a matter of adding a line. What belongs in ten lines when the
thing that must survive is a list of open questions is a real design question,
and it may be that the block should point at a durable ledger rather than carry
the content — which depends on a ledger existing.
