---
id: G-0309
title: wf-tdd-cycle promotes AC to met before branch-coverage audit and vacuity
status: open
discovered_in: M-0209
---

## What's missing

`wf-tdd-cycle`'s SKILL.md narrates its steps as RED → GREEN → REFACTOR → RECORD →
audit → vacuity, where RECORD promotes the AC to `met`. But the skill's own
Constraints say the branch-coverage audit is a hard rule that runs *before* the
commit-approval prompt, and the vacuity check is required before declaring done.
So the narrative order contradicts the mandated execution order: `met` is the
"this AC is done" judgment, yet it is narrated *before* the evidence that
substantiates it (audit + vacuity) has run.

Fix: reorder the skill so the sequence is RED → GREEN → REFACTOR → branch-coverage
audit → vacuity → RECORD (promote `met`). The "done" judgment — the HITL/agent
moment where someone can still act — must sit *after* the evidence, not before.

## Why it matters

This is the under-gating complement to G-0295's over-gating: a judgment gate
placed where the judge cannot yet see the evidence is a vacuous gate. Promoting an
AC to `met` before the coverage audit invites "green tests, untested branch"
closures — exactly what the audit exists to catch.

Addressed by E-0048 / M-0199 (wf-tdd-cycle / wf-review-code honesty), which already
opens the `wf-tdd-cycle` body; the reorder rides that surface rather than spawning
a separate work item.
