---
id: G-0461
title: Composite --for-entity acks never suppress the rule they target
status: open
priority: medium
discovered_in: M-0281
---
## What's missing

`aiwf acknowledge illegal <sha> --for-entity <id>` emits `aiwf-entity` at the
id's full composite width, and `check.WalkAcknowledgedSHAEntities` stores that
value canonicalized but otherwise verbatim — so an AC-scoped ack is keyed
`` `M-NNNN/AC-N` ``.

The rule the flag exists to suppress, `provenance-untrailered-entity-commit`,
looks acks up through `isShaEntityAcked` (`internal/check/provenance.go`), which
rolls the touched id up to its parent before indexing. The key it searches for
is therefore always the bare parent id, which the composite-width ack never
matches.

Net effect: `--for-entity` with a composite id records an audit commit that
suppresses nothing. The operator sees a successful acknowledgment and the
finding keeps firing.

## Why it matters

This is the one flag whose entire purpose is to silence that rule for a
specific (commit, entity) pair. When it silently fails, the operator's recourse
is to widen the ack to the parent id — suppressing more than they intended —
or to leave a permanent finding in `aiwf check`. Neither is the behavior the
per-pair shape was designed for, and nothing surfaces the mismatch: the ack
commit lands, exits 0, and looks correct in `aiwf history`.

The asymmetry is internal to the verb, not operator error. `verifySHATouchesEntity`
already rolls a composite argument up to its parent before checking that the
SHA's diff touches it, because a diff resolves to a milestone path rather than
an AC. So the verb validates at parent granularity and then emits at composite
granularity. The emit is the odd one out.

## Scope

Pre-existing, and wider than the convergence work that surfaced it. M-0281
fixed only the verb's own duplicate guard, which had inherited the same
rolled-up lookup and so never matched the composite value the verb itself
wrote — that repeat now converges. The suppression path this gap describes is
untouched by that fix.

## Resolution options

1. **Roll up at ingest**, in `WalkAcknowledgedSHAEntities`. One line, and it
   makes every consumer agree on the key without changing what the trailer
   records. The ack map loses AC-level granularity, but no consumer uses it
   today — both readers roll up before looking anything up.
2. **Roll up at emit**, in the verb. The ack commit would carry the parent id,
   which discards the record of which AC the operator meant — the one piece of
   information the composite form exists to preserve.
3. **Look up both keys in the rule.** Keeps granularity on both sides, at the
   cost of a second lookup in every consumer and a rule that has to know about
   composite ids.

Option 1 is the lean: the mismatch is an ingest-side inconsistency, the trailer
stays the durable record of operator intent, and it needs no change to any
consuming rule.
