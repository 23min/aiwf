---
id: G-0160
title: per-edge FSM coverage drift unpoliced (spec table vs entity.transitions)
status: open
prior_ids:
    - G-0155
discovered_in: M-0124
---
## Problem

M-0123's spec drift policies (`internal/policies/m0123_ac5_drift_test.go`) include `TestM0123_AC5_ImplToSpec_EntityFSMCovered` which asserts every `(Kind, FromState)` pair appearing in `entity.transitions` has a corresponding cell in `spec.Rules()`. This catches the "new FSM **state**" drift mode.

It does **not** catch the "new FSM **edge** to an existing state" drift mode. Concretely: if a contributor adds a new edge `(epic, active) → in_progress` to `entity.transitions` (resurrecting an active epic), the per-state coverage check still passes because `(epic, active)` already has a Rule cell. The new edge's target (`in_progress`) gets zero positive subtest coverage — the M-0124 per-cell driver's target-derivation reads from `entity.AllowedTransitions`, but the spec table's cell carries no enumeration of *which* targets it covers.

Surfaced by the M-0124 reviewer agent's audit. Per the reviewer: *"new edges to existing states are rare,"* so the gap is narrow but real.

## Why it matters

The kernel's "framework correctness must not depend on LLM behavior" principle says spec→impl drift should be policed mechanically. Today the per-state check is mechanical; the per-edge check is not. A future FSM expansion that adds a new edge without a new state will not surface in CI.

## Fix outline

Two options:

1. **Per-edge mechanical check.** Add a drift policy that iterates `entity.transitions` at the (from, to) edge level and asserts each edge has at least one positive subtest in M-0124's `enumerateLegalCases` output. Requires the cell key to encode the target (it already does in the case-name signature).

2. **Encode targets explicitly in spec cells.** Today the spec doesn't list the targets a `promote` cell reaches; the driver derives them from `entity.AllowedTransitions`. Adding an explicit `Targets []string` field to Rule (or splitting cells per target) would let M-0123's policies compare the spec's claimed targets to the FSM's allowed targets.

(1) is the smaller change; (2) is more principled but requires a spec-schema migration. Either closes the gap.

## Discovered in

M-0124 (reviewer-agent audit pre-merge).

## Status

`open`.
