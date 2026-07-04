---
id: G-0361
title: aiwfx-release never routes release intent to the deployer agent
status: open
---
## Problem

Session mining (G-0353) showed the `deployer` subagent dispatched ~0 times across 73 sessions, versus `reviewer` at ~178x. The `deployer` agent card already lists `aiwfx-release` under 'Skills you use', but the cross-reference is one-directional: `aiwfx-release/SKILL.md`'s own 'When to use' section never mentions delegating to `deployer`, and its trigger phrases (*release v1.2*, *tag a release*, *publish*) don't cover common phrasings (*let's ship*, *let's release*, *make a new version*). When an assistant matches a user's release-cutting language against the skill directly, it runs `aiwfx-release` inline in the main conversation rather than reaching for the agent — the aiwf.yaml-configurable model/effort tier for `deployer` (per G-0353) never applies because the agent never runs.

## Why it matters

The four role agents (planner/builder/reviewer/deployer) exist specifically so each carries a tunable model/effort tier. A tier no session ever exercises is dead configuration surface. Beyond cost tuning, running the release ritual inline also consumes the calling session's context budget instead of an isolated one.

## Fix shape

1. Broaden `aiwfx-release`'s 'When to use' trigger phrases.
2. Add an explicit instruction there to delegate to the `deployer` subagent rather than running inline.
3. A `UserPromptSubmit` Claude Code hook (session/repo-local, not an aiwf-shipped feature — aiwf itself doesn't touch settings.json hooks per ADR-0015) that pattern-matches release-cutting phrasing and injects a directive nudge toward dispatching `deployer`. This is the reliability backstop; 1-2 alone are still advisory.