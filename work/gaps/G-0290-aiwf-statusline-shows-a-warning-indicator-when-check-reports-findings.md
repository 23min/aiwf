---
id: G-0290
title: aiwf statusline shows a warning indicator when check reports findings
status: open
prior_ids:
    - G-0288
---
## Problem

When `aiwf check` (or `aiwf doctor`) would report findings, nothing surfaces that
state in the Claude Code statusline. An operator working in the session has no
ambient signal that the planning tree has drifted into a findings state until they
run a verb by hand.

## Direction

When the aiwf-aware statusline is opted in (`aiwf init/update --statusline`), prefix
it with a warning indicator (e.g. a yellow triangle) whenever the tree is in a
findings state. The hard constraint: the statusline renders on every prompt, so it
must NOT run a live `aiwf check` (seconds-scale). It reads a cheap cached signal
instead — the result of the last pre-commit / pre-push check, or a fast shape-only
probe, persisted to a small state file the statusline reads. Settings edits remain
gated by the explicit per-invocation consent the statusline opt-in already requires
(ADR-0015).

Surfaced while planning the area-matrix validation work. Orthogonal to that work — a
general statusline UX signal for any findings state.
