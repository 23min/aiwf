---
id: G-0435
title: Declared-sequence gate's shape list reads as exhaustive, blocking batching
status: open
priority: medium
discovered_in: M-0126
---
## Problem

Today, `aiwf authorize <entity> --to ai/<id>` only changes two things: the commit's actor trailer and the branch binding recorded in the `aiwf-branch:` trailer. It says nothing about check-in cadence. The standing collaboration guidance ("Every mutating action is its own approval gate... ask before each") is written unconditionally — it does not carve out a different gate cadence for work happening inside an open delegation scope. So opening a scope changes provenance bookkeeping but does not change how often a human is asked to approve an individual mutation.

This was tested directly in E-0034/M-0126: the epic was authorized to `ai/claude`, but every commit in that milestone's implementation and wrap still ran as `human/peter`, gated individually — dozens of approve/deny round-trips, no different in cadence from an in-loop (non-delegated) milestone. The delegation bought accountability tracking, not any reduction in interactive overhead, defeating the natural expectation that "delegating work" should mean less babysitting.

There is already a precedent for a different cadence living next to this: `tdd: required` milestones stream their AC phase-promotes live and ungated ("phase promotes (red/green/done/met) streamline by default — live, ungated, test-evidenced — wrap review and push stay the control points"). That is the same shape of trade-off — fewer live gates, review concentrated at a checkpoint — already accepted for one narrow case. This gap proposes generalizing that shape to delegated-scope work.

## Direction

While an `aiwf authorize` scope is `active` for an entity, local/reversible mutations performed within that scope (commits, `aiwf add ac`/`add gap`, non-sovereign promotes) stream without individual per-mutation gates. Review and approval concentrate at named checkpoints — an AC boundary, a milestone wrap, or an equivalent unit the operating guidance defines. Two classes of action are excluded from this relaxation and stay gated exactly as today, regardless of scope:

- **Sovereign acts** — `aiwf promote <epic> active`, `aiwf authorize` itself, any `--force` invocation. Human-only by the kernel's own sovereignty rule; delegation cannot touch them.
- **Outward/irreversible actions** — push, `gh pr create`, tag-push, remote-branch delete. These leave the machine or affect shared state; they stay individually gated regardless of scope.

This is a change to the standing collaboration guidance (`CLAUDE.md`'s "Gate discipline survives compaction" section and the shipped `internal/skills/embedded-guidance/aiwf-guidance.md` fragment), not a kernel/verb change — `aiwf authorize` itself doesn't need new code; what needs to change is the session-behavior rule that currently ties every mutation to a gate unconditionally. The rule needs a scope-aware carve-out: if an authorize scope is active for the entity being worked on, and review is concentrated at the next checkpoint, individual local/reversible mutations may proceed without a per-action gate.

Open design questions for whoever picks this up:

- What is the right checkpoint granularity — per-AC (smaller diffs to audit, more checkpoints) or per-milestone (maximizes friction reduction, larger diff to review at once)?
- Should the human get a live progress narration (a running log) even without being asked to approve each step, so they are not flying blind between checkpoints?
- Should this be opt-in per authorize invocation (a flag) or the unconditional default once a scope is active?