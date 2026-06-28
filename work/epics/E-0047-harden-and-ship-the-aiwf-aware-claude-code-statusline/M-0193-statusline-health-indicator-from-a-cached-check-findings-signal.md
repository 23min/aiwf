---
id: M-0193
title: Statusline health indicator from a cached check-findings signal
status: draft
parent: E-0047
depends_on:
    - M-0192
tdd: required
---
## Deliverable

The statusline prefixes a **health indicator** when the planning tree is in a findings state (G-0290).

- A warning glyph (e.g. a yellow triangle) prefixes the statusline when the cached check-state shows findings; absent when clean.
- The statusline **never** runs a live `aiwf check` — it renders on every prompt, so a seconds-scale check is forbidden. It reads a cheap *persisted* signal: the result of the last pre-commit / pre-push check (or a fast shape-only probe) written to a small state file.
- Settings edits remain gated by the explicit per-invocation consent the statusline opt-in requires (ADR-0015).

This milestone establishes the **shared tree-health signal** the epic's through-line names — the same signal `aiwf doctor` (G-0289) and the default `aiwf status` divergence flag (G-0277) can later consume. ACs at milestone start, tested against the M1 harness.
