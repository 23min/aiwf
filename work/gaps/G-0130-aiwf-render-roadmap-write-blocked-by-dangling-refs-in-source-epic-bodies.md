---
id: G-0130
title: aiwf render roadmap --write blocked by dangling refs in source epic bodies
status: addressed
addressed_by:
    - ADR-0010
---
`aiwf render roadmap --write` composes `ROADMAP.md` by copying entity references out of source epic bodies. When those bodies contain pre-uniform-width (narrow-id) paths or references to files that have since moved or never existed, the regenerated `ROADMAP.md` inherits the dangling refs verbatim. The verb then attempts a single commit, the pre-commit policy hook (`PolicyNoDanglingEntityRefsInNarrativeDocs` under `internal/policies/`) fires on `ROADMAP.md`, and the commit aborts.

Net effect: `aiwf render roadmap --write` cannot complete on the kernel repo as of 2026-05-19. The render itself runs and produces a regenerated tree, but the verb's atomic commit is blocked.

## How to reproduce (historical, 2026-05-19)

```
./aiwf render roadmap --write
```

Observed (E-0034 planning session, 2026-05-19):

```
ROADMAP.md:152: [G-0055](../../gaps/G-055-...md) -> not found
ROADMAP.md:167: [G-0058](../../gaps/G-058-...md) -> not found
ROADMAP.md:183: [M-0070](M-070-...md) -> not found
ROADMAP.md:183: [M-0071](M-071-...md) -> not found
ROADMAP.md:188: [G-0062](../../gaps/G-062-...md) -> not found
ROADMAP.md:188: [G-0064](../../gaps/G-064-...md) -> not found
ROADMAP.md:328: [ADR-0010](../../../docs/adr/ADR-0010-branch-model-...md) -> not found
```

Two failure classes were mixed in the output:

1. **Narrow-id drift.** Entries 1–6 reference entities by their pre-E-0023 narrow-width ids. E-0023 (Uniform 4-digit kernel ID width) renamed the files to canonical 4-digit width but the source epic bodies (E-0016, E-0017, E-0018) still cited the old paths.
2. **Stale file reference.** Entry 7 referenced an `ADR-0010` that did not exist at filing time.

## Why it mattered

- **The render verb's contract was broken.** `aiwf render roadmap --write` is documented as the canonical way to regenerate ROADMAP.
- **Drift compounded.** As long as the verb was blocked, ROADMAP could not regenerate.
- **The chokepoint was right but the renderer was the wrong layer to fail.** The renderer is mechanically deterministic from source bodies, so failures here are always upstream-caused.

## Disposition (2026-05-20, organic close)

Both failure classes resolved between 2026-05-19 (filing) and 2026-05-20 (this close) without a dedicated fix:

1. **Narrow-id drift cleared organically.** The specific cited refs (`G-055`, `G-058`, `G-062`, `G-064`, `M-070`, `M-071`) are no longer present in current `ROADMAP.md`. Either source epic bodies were swept in passing during other work, or the cited entities transitioned into archive and the renderer no longer surfaces broken paths to them. Verified via `grep` against current ROADMAP.md — none of the patterns remain. One residual narrow-id mention persists in `docs/pocv3/plans/acs-and-tdd-plan.md:169` as prose discussion (not a markdown link), which the dangling-ref policy correctly ignores.

2. **ADR-0010 landed** at commit `281c81eb` (`aiwf add adr ADR-0010 "Branch model: ritualized work on branches, author iteration on main"`) and is now `accepted`. The current ROADMAP.md line 328 cites it at exactly the path the file occupies — link resolves.

`aiwf render roadmap --write` now runs cleanly (reports "already up to date" against an in-sync ROADMAP.md).

## What remains undone

The original gap proposed three fix candidates; only candidate (1) — "renderer canonicalizes at emit time" — is a durable chokepoint. Candidates (2) and (3) were transient cleanups that happened organically and don't prevent the next sloppy ref from re-introducing the symptom.

A successor gap for the renderer-canonicalization chokepoint is **not opened** at close time per the closure decision: the chokepoint is preventative discipline, and filing it speculatively risks the kind of "open gap with no forcing function" drift the planning tree avoids. **If the dangling-ref block reappears** on a future `aiwf render roadmap --write` invocation, file a fresh gap then — with the renderer-canonicalization fix shape as the proposed approach, and a citation back to this closing record.

## References

- **E-0023** — Uniform 4-digit kernel ID width.
- **ADR-0004** — Uniform archive convention.
- **ADR-0010** — Branch model (the formerly-dangling reference; now resolves).
- **E-0034** — Retire docs/pocv3/ and declare doc-authority hierarchy. Surfaced this gap during its planning session.
- `internal/policies/no_dangling_entity_refs.go` — the chokepoint policy.
