---
id: G-0362
title: Decide whether aiwf ships a mechanical role-agent routing mechanism
status: open
---
## Problem

G-0361 fixed the concrete, ritual-content-scoped half of 'deployer never gets dispatched' — broadened trigger phrases and an explicit delegation instruction in `aiwfx-release`'s frontmatter description and body. That fix is still advisory: it only improves the odds an assistant recognizes release-cutting intent and chooses to dispatch `deployer`, the same class of guarantee CLAUDE.md's own principle already flags as insufficient (*'A guarantee that depends on the LLM remembering to invoke a skill is not a guarantee.'*).

The mechanical alternative — a `UserPromptSubmit` Claude Code hook that pattern-matches release-cutting phrasing and forces a directive into context — was deliberately NOT built as part of G-0361, because it would be a new **shippable aiwf feature** (every consumer running `aiwf init`/`update` should get the same reliability, not just this repo), and shipping it means reopening the boundary ADR-0015 drew on purpose: 'aiwf does not edit your .claude/settings.json without explicit per-invocation consent.' That's a real architectural decision, not a bug fix — it needs its own ADR before any implementation.

Separately, code review during G-0361 noted the `deployer` agent card's own frontmatter description carries no release-trigger phrases at all (unlike aiwfx-release's, which now does) — worth weighing in the same pass, whether agent cards should carry redundant trigger phrasing as another (still-advisory) surface, independent of whatever the ADR decides about a mechanical hook.

## Why it matters

Without this decision, 'is deployer reliably discoverable' stays bounded by whatever the advisory ritual-content fix achieves — real, but not the guarantee the underlying complaint (deployer at ~0 dispatches across 73 sessions per the archived G-0353 gap) asked for.

## Shape (sketch)

An ADR (via `aiwfx-record-decision`) weighing at least: (a) ritual-content-only, stay advisory; (b) strengthen the always-on embedded guidance fragment (broader reach than one skill's description, still advisory, no new consent surface); (c) a new opt-in hook-materialization feature mirroring the statusline consent pattern (aiwf init/update flag, gated like ADR-0015's statusline opt-in), with the same hook-chaining composability aiwf's git hooks already have (G-0045) so it never clobbers a consumer's own hooks.

Resolved via ADR-0028: option (b), broaden the always-on embedded guidance fragment; option (c) rejected for now.