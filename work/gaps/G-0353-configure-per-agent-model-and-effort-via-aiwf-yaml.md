---
id: G-0353
title: Configure per-agent model and effort via aiwf.yaml
status: open
---
## What's missing

aiwf materializes the four role agents (planner, builder, reviewer, deployer)
into `.claude/agents/` with the `model:` and `effort:` frontmatter fields
absent, so every dispatched subagent runs on whatever model and effort the
session inherits. Claude Code has supported both fields on agent cards since
`v2.1.198` (`model` ∈ opus | sonnet | haiku | fable | inherit; `effort` ∈
low | medium | high | xhigh | max; thinking is not settable per-agent), but
aiwf exposes no `aiwf.yaml` surface to configure them and no materialization
path to inject them. An operator who wants a role on a cheaper tier can only
hand-edit the gitignored card — which the next `aiwf update` overwrites.

## Why it matters

Model and effort are the primary compute-cost lever, and today they are
unreachable through configuration — the one thing an operator most wants to
tune cheaply. Session mining across 73 local sessions shows the reviewer
subagent is dispatched ~178× (builder ~12×; planner and deployer ~0× as
dispatched agents), so the reviewer alone is a large, recurring spend running
at the session's default tier when a cheaper one would do. Because only a
*dispatched* subagent can mechanically carry a model/effort — a skill runs in
its caller's context and can't — the card is the one surface where a tier
actually binds, and it is exactly the surface with no config knob. The gain is
concrete: putting the reviewer on a near-Opus-quality but ~40–60% cheaper model
at a lower effort (a step down from the `xhigh` the session otherwise passes
through) is a large recurring saving realized by editing one config value — but
nothing implements that path.

## Shape (sketch)

A minimal, high-leverage design — no new roles, no skill rewrites:

- An `agents:` block in `aiwf.yaml`, keyed by **shipped-agent name**, each entry
  an optional `model` and optional `effort`. Keys validate against the set of
  agents aiwf actually materializes (a closed set derived from the embedded
  snapshot, not a hardcoded list), so a typo is a clean finding and a future
  shipped agent is automatically configurable.
- On `aiwf init` / `aiwf update`, inject `model:` / `effort:` frontmatter into
  each matching card; omit a field → the card inherits the session. Applying a
  change is `aiwf update`; reverting is deleting the block and re-running it.
- Values are closed sets; aiwf validates membership only — it does not police
  model×effort compatibility. This is an advisory cost feature, never
  load-bearing on correctness (the guarantee cannot depend on which tier ran).
- Discoverability per the `config_fields_discoverable` chokepoint (config-field
  docs + a doctor/skill surface). An ADR recording the "advisory,
  agent-anchored, not-load-bearing" model-policy stance likely rides along.

This overlaps the compute concern gestured at in E-0051 (proposed); this gap
captures the model/effort-config lever on its own, independent of that epic's
fate. The session-topology pillar of E-0051 is deliberately out of scope here.
