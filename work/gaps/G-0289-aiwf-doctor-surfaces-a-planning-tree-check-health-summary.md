---
id: G-0289
title: aiwf doctor surfaces a planning-tree check-health summary
status: open
prior_ids:
    - G-0287
---
## Problem

`aiwf doctor` reports operator-setup health — version, materialized-skill drift,
hook drift, recommended-plugin presence, env/plugin-mount, and a narrow id-collision
check. It does NOT surface the planning-tree validation that `aiwf check` owns, so an
operator running `aiwf doctor` to answer "is everything OK?" gets a clean bill of
health even when `aiwf check` would report findings (area-unknown, dead-glob, FSM,
ids, and the rest). The two verbs answer two halves of "is everything OK" and only
one half is reachable from the doctor entry point.

## Direction

Add a single delegated summary line to the doctor report — e.g.
`planning tree: clean` / `planning tree: N findings -> run aiwf check` — by invoking
the existing check pass and reporting its finding count, NOT by recomputing any
finding. Recomputation would be a second source of truth for findings `check`
already owns — the parallel-source-of-truth the kernel forbids, and exactly the kind
of drift the doctor `--self-check` exists to catch. The summary is general (overall
check health), not scoped to any one finding family. It is advisory: it points at
`aiwf check`, which remains the authoritative pre-push gate.

Surfaced while planning the area-matrix validation work. General, not area-specific —
areas would simply be some of the findings the summary counts.
