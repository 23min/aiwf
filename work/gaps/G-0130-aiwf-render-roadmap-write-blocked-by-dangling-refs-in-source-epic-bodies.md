---
id: G-0130
title: aiwf render roadmap --write blocked by dangling refs in source epic bodies
status: open
---

`aiwf render roadmap --write` composes `ROADMAP.md` by copying entity references out of source epic bodies. When those bodies contain pre-uniform-width (narrow-id) paths or references to files that have since moved or never existed, the regenerated `ROADMAP.md` inherits the dangling refs verbatim. The verb then attempts a single commit, the pre-commit policy hook (`PolicyNoDanglingEntityRefsInNarrativeDocs` under `internal/policies/`) fires on `ROADMAP.md`, and the commit aborts.

Net effect: `aiwf render roadmap --write` cannot complete on the kernel repo as of 2026-05-19. The render itself runs and produces a regenerated tree, but the verb's atomic commit is blocked.

## How to reproduce

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

Two failure classes are mixed in the output:

1. **Narrow-id drift.** Entries 1–6 reference entities by their pre-E-0023 narrow-width ids (`G-055`, `G-058`, `M-070`, `M-071`, `G-062`, `G-064`). E-0023 (Uniform 4-digit kernel ID width) renamed the files to canonical 4-digit width but the source epic bodies (E-0016, E-0017, E-0018) still cite the old paths. The renderer copies those citations verbatim into ROADMAP. The kernel-shipped fix-path was `aiwf rewidth --apply` — which canonicalizes entity files on disk but does not rewrite the prose body of *other* entities that cite them by narrow path.

2. **Stale file reference.** Entry 7 references `docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md`. That file does not exist (no `ADR-0010` ships in the current tree). Likely an aspirational reference to an as-yet-unwritten ADR. Different drift class from the narrow-id one.

## Why it matters

- **The render verb's contract is broken.** `aiwf render roadmap --write` is documented as the canonical way to regenerate ROADMAP (`aiwfx-plan-epic` skill instructs operators to run it; `internal/skills/embedded/aiwf-render/SKILL.md` confirms). When it cannot complete, every planning session that touches an epic leaves ROADMAP stale.
- **Drift compounds.** As long as the verb is blocked, ROADMAP cannot regenerate, which means new epics (e.g. E-0034 added today) never appear in the rendered table. Readers either rely on `aiwf status` (different surface, different shape) or read ROADMAP and form an out-of-date mental model.
- **The chokepoint is right but the renderer is the wrong layer to fail.** The pre-commit policy fires correctly — dangling refs in ROADMAP *should* block. But the failure surface is the wrong actor: the renderer is mechanically deterministic from source bodies, so failures here are always upstream-caused. The verb should either canonicalize on the way out, or fail with a hint pointing at the offending source bodies, not just dump the dangling-ref findings as if a human authored ROADMAP.

## Fix shape

Three candidate moves, not mutually exclusive:

1. **Renderer canonicalizes at emit time.** When composing ROADMAP, the renderer resolves every entity reference through the loader (which already handles narrow-id and active/archive transparency per ADR-0004) and emits canonical-width bare-id form (`G-0055` not `[G-0055](../../gaps/G-055-...md)`) or canonical paths. Source bodies stay as they are; the rendered output is always clean. This is the proper chokepoint move — the renderer is the single emit point for ROADMAP, so canonicalization belongs there.

2. **Sweep source epic bodies to canonical refs.** Run a one-off audit of every epic body containing a narrow-id ref and update via `aiwf edit-body` to use bare-id form. Cheaper, but a transient fix — the next time a new epic is authored with a sloppy ref, the problem returns. The first-class fix is (1).

3. **Land ADR-0010 (or remove the reference).** The `ADR-0010` reference in some epic body (likely E-0030) points at an ADR that does not exist. Either land the ADR (it's referenced from a `proposed` epic E-0030 about branch model, so it might be the right time anyway), or rewrite the citation to a different form pending ADR allocation.

(1) is the load-bearing fix; (2) and (3) are residual cleanups.

## Out of scope

- The pre-commit policy itself is correct. This gap does not relax `PolicyNoDanglingEntityRefsInNarrativeDocs`; it asks the renderer not to *produce* the drift it would then flag.
- Regenerating ROADMAP retroactively for entities authored while this gap is open. Once the renderer canonicalizes, the next `--write` invocation will sweep the stale state.

## References

- **E-0023** — Uniform 4-digit kernel ID width. Renamed entity files but did not sweep cross-references inside other entities' body prose.
- **ADR-0004** — Uniform archive convention. The loader transparency it commits to is precisely what the renderer should lean on for canonical emission.
- **E-0034** — Retire docs/pocv3/ and declare doc-authority hierarchy. Surfaced this gap during its planning session when the `aiwf render roadmap --write` step blocked.
- `internal/policies/no_dangling_entity_refs.go` — the chokepoint policy.
