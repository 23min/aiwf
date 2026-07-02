---
id: G-0341
title: Shipped aiwf-guidance carves an approval-batching exception it doesn't own
status: open
---
## Problem

The shipped embedded guidance (`internal/skills/embedded-guidance/aiwf-guidance.md`,
materialized into every consumer's `.claude/aiwf-guidance.md`) frames the
declared-sequence gate as *"the one bounded exception"* to gate-per-mutation. That
positions aiwf — a tool — as amending the operator's **general approval philosophy**.

Per this repo's "how to OPERATE aiwf" vs "how the operator collaborates" split,
whether a human is willing to batch a sequence of local, reversible actions into one
approval is a collaboration-philosophy choice the operator owns — not an
aiwf-operating mechanic the framework should ship a position on. The batching
allowance belongs in the operator's own base guidance layer (e.g. a personal
standing-rules file), where it can loosen the default. A consumer who imports
aiwf-guidance but holds a strict never-batch baseline gets framework text that
overrides their own rule.

## Direction (for the milestone that addresses this)

- Ship a strict, neutral default in `aiwf-guidance.md`: each mutation is its own
  approval gate; outward/irreversible actions (push, `gh pr create`, tag-push,
  `--force`) are never batched. Drop the declared-sequence-gate "exception" prose so
  aiwf stays neutral on whether local reversible sequences may be batched.
- Preserve the `gate-per-mutation` anchor: `PolicyM0211GuidanceOperatingAnchors`
  pins the fragments `each mutating action` / `approval gate`, which must survive the
  edit.
- Coherence check: the wrap rituals (`aiwfx-wrap-milestone`, `aiwfx-wrap-epic`)
  present a terminal local sequence (promote-done, local merge, cleanup) as one gate.
  That mechanic stands on its own and is permitted whenever the operator's base layer
  allows local batching — it does not need aiwf-guidance to assert an exception.
  Verify the ritual prose still reads coherently once the shipped exception is gone.
- Re-materialize via `aiwf update` so `.claude/aiwf-guidance.md` regenerates.

## Why a gap, not a direct edit

The change touches shipped consumer-facing guidance, a policy chokepoint, and a
coherence sweep across the wrap rituals — enough surface to plan and test
deliberately rather than hand-edit in passing.
