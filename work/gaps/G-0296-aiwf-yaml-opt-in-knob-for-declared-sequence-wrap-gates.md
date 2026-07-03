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

Add an `aiwf.yaml` knob, `gates.declared_sequence_wraps`, defaulting **`true`**:

- `true` (default) -> milestone/epic wraps use the declared-sequence gate from
  the Tier-1 rule (one approval over an enumerated, subset-approvable sequence
  that aborts-and-re-gates on any deviation);
- `false` -> wraps fall back to a separate per-action gate for each mutation
  (the conservative form).

Default-on because it matches how the rituals already operate and the direction
of travel: G-0314 shipped the streamlined TDD phase-promote cadence as the
guidance default. The declared-sequence gate is safe by construction — it names
every action, permits partial approval, and stops the instant anything deviates
— and the shipped mechanical backstops narrow the residual risk of default-on:
`aiwf init` wires a pre-push net, and the G-0355 promote-time `--by-commit`
reachability check refuses a tracker closure recorded onto a commit trunk lacks,
in any consumer repo, regardless of the knob.

The bright line still binds regardless of the knob: outward/irreversible actions
(push, tag-push, remote-branch delete, `--force`) and `tdd: required` phase
promotes are never batched.

Mirrors the existing config style (`guidance.wire_claudemd`,
`archive.sweep_threshold`). Because the default is on, the meaningful config an
operator writes is `false` — i.e. the knob is an opt-*out* for repos that want
per-action ceremony, not an opt-in; the entity title should be corrected from
"opt-in knob" to a neutral framing when this is built or sooner.

## Scope

`aiwf.yaml` schema + loader, the wrap rituals' reading of the knob. Deferred:
implement only after the Tier-1 declared-sequence rule has been in use long
enough to confirm the convention is worth crystallizing into config.
