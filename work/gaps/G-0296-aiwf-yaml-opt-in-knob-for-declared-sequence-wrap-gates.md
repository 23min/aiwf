---
id: G-0296
title: aiwf.yaml opt-in knob for declared-sequence wrap gates
status: open
---
## Problem

Follow-up to the declared-sequence gate generalization (Tier 1). That rule lets
an operator opt into a low-ceremony wrap gate *per sequence* by invoking the
convention conversationally. An operator who wants that preference recorded
**durably** — visible, checked-in, and compaction-proof — rather than re-stated
each session, has no mechanism today.

## Decision (Tier 2, deferred)

Add an `aiwf.yaml` opt-in knob, e.g. `gates.declared_sequence_wraps: true`:

- default off -> milestone/epic wraps use per-action gates (conservative);
- on -> the wraps use the declared-sequence gate from the Tier-1 rule.

Mirrors the existing opt-out/opt-in config style (`guidance.wire_claudemd`,
`archive.sweep_threshold`). It is advisory-on-advisory — the rituals read the
knob and honor it — but it turns the per-action-vs-declared-sequence choice into
a recorded per-repo preference instead of a fragile conversational instruction.

The bright line still binds: outward/irreversible actions and `tdd: required`
phase promotes are never batched regardless of the knob.

## Scope

`aiwf.yaml` schema + loader, the wrap rituals' reading of the knob. Deferred:
implement only after the Tier-1 declared-sequence rule has been in use long
enough to confirm the convention is worth crystallizing into config.
