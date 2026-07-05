---
id: ADR-0028
title: Role-agent routing stays advisory via broadened guidance
status: proposed
---

# ADR-0028 — Role-agent routing stays advisory via broadened guidance

> **Date:** 2026-07-05 · **Decided by:** human/peter

## Status vocabulary (aiwf)

aiwf's ADR statuses are: `proposed | accepted | superseded | rejected`.

## Context

G-0353 (archived) found the `deployer` role agent dispatched ~0 times across
73 sessions despite having a configurable `model`/`effort` tier — a tier that
is dead configuration when the agent never runs. G-0361 (addressed) diagnosed
the immediate cause for `deployer` specifically: `aiwfx-release` carried
narrow trigger phrasing and no instruction to delegate, so an assistant
matching release-cutting language against the skill directly ran it inline
instead of dispatching `deployer`. G-0361 shipped an advisory fix scoped to
that one skill: broadened trigger phrases plus an explicit "dispatch the
`deployer` subagent" instruction in `aiwfx-release`'s frontmatter description
and body.

G-0362 asks whether that scoped fix is enough, or whether aiwf should ship a
mechanical backstop. Two shapes were weighed:

(b) Broaden the always-on embedded guidance fragment
(`internal/skills/embedded-guidance/aiwf-guidance.md`, `@`-imported into
every consumer's CLAUDE.md per the wiring ADR-0018 established) to name
dispatch triggers for all four role agents, not just `deployer` via one
skill's own description. Confirmed the fragment today says nothing about
role-agent dispatch at all. Still advisory, but in context every turn rather
than only when the assistant happens to load `aiwfx-release`.

(c) A new `UserPromptSubmit` Claude Code hook, shipped by aiwf and
materialized opt-in (mirroring the `--statusline` consent flow ADR-0015
established), that pattern-matches release-cutting phrasing and injects a
directive nudge.

## Decision

Broaden the always-on embedded guidance fragment (option b). Reject a
mechanical `UserPromptSubmit` hook (option c) for now.

The problem being solved is cost-tier utilization and discoverability, not
correctness — the release ritual runs correctly whether or not `deployer` is
the one running it. CLAUDE.md's principle that "a guarantee that depends on
the LLM remembering to invoke a skill is not a guarantee" is written for
correctness-bearing surfaces (`aiwf check`, the pre-push hook), where a missed
invocation produces a wrong or unvalidated result. A role agent's dispatch
rate is not that kind of guarantee; a proportionate fix for a soft nudge is a
stronger nudge, not new mechanical infrastructure.

A `UserPromptSubmit` hook would also reopen the settings.json consent
boundary ADR-0015 deliberately drew, requiring its own opt-in consent flow
plus hook-chaining composability work so it doesn't clobber a consumer's own
hook. Even after that engineering, such a hook can only inject context into
the prompt — it cannot force a tool call — so it does not close the gap to an
actual mechanical guarantee; it only raises the odds further, the same
category of improvement broadened guidance buys more cheaply.

Concretely: extend `internal/skills/embedded-guidance/aiwf-guidance.md` with
dispatch guidance covering all four role agents (planner/builder/reviewer/
deployer). Exact wording is follow-up implementation work, not prescribed
here.

## Consequences

- `internal/skills/embedded-guidance/aiwf-guidance.md` gains a new section on
  role-agent dispatch triggers (separate follow-up work; not scoped here).
- No new settings.json surface, no new consent flow, no reopening of
  ADR-0015's boundary.
- The dispatch-rate problem is not fully solved — it remains advisory. If a
  second role agent shows the same near-zero-dispatch pattern after the
  guidance broadens, that recurrence is the signal to revisit option (c) with
  real multi-agent evidence rather than a single case.

## Validation (optional)

Revisit if a second role agent shows a comparable near-zero dispatch rate
after the guidance fragment broadens.

## References

- G-0362, G-0361, G-0353 (archived)
- ADR-0015
