---
id: G-0387
title: Add duration to the verb.completed/verb.failed diagnostic event
status: open
priority: low
discovered_in: M-0238
---
## What's missing

The verb.completed/verb.failed diagnostic event (M-0238/AC-5, AC-6)
carries verb/entity/actor/run_id and, for cancel/move, a sha — but no
duration. There is no verb.started event either, so there is nothing
to measure a duration from.

## Why it matters

Duration is useful for a different diagnostic question than the sha
(which answers "what did this verb do to the repo") — it answers "why
is this verb slow," relevant when a CI run or a hook invocation hangs
or takes longer than expected. Adding it needs a verb.started-shaped
timestamp capture at the same mint-once point AC-5 introduced, then a
duration computed at the same EmitVerbOutcome call site.

Recommended: add this to whichever milestone next touches
cliutil.EmitVerbOutcome's call sites (M-0239 is a natural candidate,
since it already touches the same seam for correlation_id wiring).