---
id: G-0261
title: auto-invoke wf-rethink when a unit introduces a non-trivial design
status: addressed
addressed_by_commit:
    - 9a4bd65d
---
## What's missing

`wf-rethink` (design-quality re-derivation of one unit) fires only on explicit invocation. It should be **auto-invoked when a unit of work introduces a non-trivial design**, before that design is committed — while it is still cheap to change. Today an LLM agent (a greedy optimizer) commits the first adequate design, and the local-optimum residue — single-caller abstractions, defensive layers, edit-history shape — ships unreviewed.

## Where it wires in

- **`wf-patch`** — before the commit gate, *when the patch introduced a non-trivial design* (a new module boundary, core abstraction, data model, or exported type).
- **Milestone implementation** — after a design lands and before wrap.

## The hard design question (rethink-specific)

**The trigger condition.** Most patches are trivial; auto-invoking `wf-rethink` on every one would be churn, and the skill itself warns against overuse and "never run over the whole codebase at once." The crux of this gap is defining *what counts as a non-trivial design* that warrants a rethink — a new exported type? a new package boundary? a data model? more than N lines of new branching logic? That condition is the deliverable, and it is distinct from anything the sibling vacuity-wiring gap (G-0260) needs — which is why the two are separate gaps rather than one.

## The feedback loop

`wf-rethink` is biased-to-keep, but a rewrite verdict is gated and, if taken, re-enters the cycle (re-test, re-review). The wiring must route a rewrite back through the patch's own gates rather than bypassing them.

## Why it matters

Design quality, unlike test strength, has no mechanical gate at all — there is no `mutate-hunt` equivalent for "is this design accreted?". So invocation wiring is the *only* lever. But because it can trigger a rewrite, it must be conditioned carefully (the trigger question above) to avoid churn — the opposite failure mode from never invoking it.

## Source

The `wf-rethink` ritual; surfaced in the session that filed G-0259/G-0260. Sibling: G-0260 wires `wf-vacuity` by the same invocation mechanism at a different trigger.
