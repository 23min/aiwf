---
id: G-0370
title: Broaden the always-on guidance fragment with role-agent dispatch triggers
status: open
priority: high
---
## Problem

`ADR-0028` decided that the fix for role-agent dispatch discoverability
(`G-0362`) should broaden `internal/skills/embedded-guidance/aiwf-guidance.md`
with dispatch-trigger guidance covering all four role agents
(planner/builder/reviewer/deployer) — not just `deployer` via one skill's own
description, which `G-0361` and the follow-up `deployer.md` card fix already
addressed. The ADR recorded the decision but did not implement it: the
always-on guidance fragment today still says nothing about role-agent
dispatch at all.

## Why it matters

The always-on fragment is `@`-imported into every consumer's `CLAUDE.md` and
loaded on every turn, regardless of which skill an assistant happens to load
first — it is the broadest-reach surface `ADR-0028` chose specifically because
it isn't gated on any one skill being read first. Without this follow-through,
the dispatch-discoverability problem the `G-0353` → `G-0361` → `G-0362` →
`ADR-0028` chain diagnosed remains only partially addressed: individual
skill/card descriptions now name triggers, but the one surface guaranteed to
be in context every turn still doesn't.

## Shape (sketch)

Extend `internal/skills/embedded-guidance/aiwf-guidance.md` with a short
section naming dispatch triggers for planner/builder/reviewer/deployer,
mirroring the trigger-phrase pattern already established on the role-agent
cards and in `aiwfx-release/SKILL.md`. Small, content-only change; a
`wf-patch`-sized fix.

Needs a hand-written pinning test under `internal/policies/` (the
`skill-edit-structural-test-backstop` mechanical gate is scoped to
`*/SKILL.md` paths only, so it will not fire automatically on this file —
follow the same manual-pinning convention already used for other
non-`SKILL.md` ritual content, e.g. `builder_tdd_opt_in_test.go` and
`deployer_card_release_triggers_test.go`).
