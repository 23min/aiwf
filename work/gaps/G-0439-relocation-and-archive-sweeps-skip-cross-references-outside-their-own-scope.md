---
id: G-0439
title: Relocation and archive sweeps skip cross-references outside their own scope
status: open
priority: medium
discovered_in: E-0034
---
## What's missing

Two movers rewrite cross-references only inside the scope each treats as its own, and links outside that scope keep pointing at a vacated path.

**A doc-relocation sweep skips `CHANGELOG.md`.** E-0034 relocated the pre-migration docs tree under `docs/archive/`, sweeping cross-references across the `docs/` tree and source code but not `CHANGELOG.md` — a pre-migration `## [X.Y.Z]` entry still linked to the vacated path for `gaps-pre-migration.md`, which stopped resolving the moment the relocation landed. The file now lives under `docs/archive/pocv3/`. Nothing caught this until a later release cut's pre-release link-check ran.

**`aiwf archive` skips `docs/`.** The sweep rewrites every cross-reference in an entity body, and a doc holds none of those, so a doc linking to an entity keeps the active-tree path once that entity moves under `archive/`. Measured at the v0.31.0 cut: four links in `docs/initiatives/quality-signal-and-cadence.md` still pointed at E-0073, M-0281 and G-0468 under `work/epics/` and `work/gaps/` after earlier sweeps had moved all three. link-check was red for nine consecutive runs before the paths were corrected by hand.

## Why it matters

`CHANGELOG.md` is append-only/forget-by-default (per its own documented convention) specifically so past entries don't need to track a moving codebase — but that convention assumes referenced files don't move without the mover also fixing CHANGELOG's copy of the link. A doc-relocation epic like E-0034 has every incentive to sweep docs/ and source but easily forgets CHANGELOG.md, since it isn't under docs/ and doesn't read as "documentation" in the moment.

`aiwf archive` carries the sharper form of the same blind spot, because it is a verb rather than a one-off human sweep: it runs routinely, it rewrites entity bodies correctly, and nothing tells the operator that a file outside the entity tree just lost a target. The archived entity also leaves the sweep's scan for good, so no later run revisits the question.

Both instances surfaced at a release cut — the wrong gate, where the fix is a hand-edit under time pressure rather than a rewrite at the sweep that caused it.

## Sketch

Three candidate fixes, not mutually exclusive:

- **Widen the mover's scan.** `aiwf archive`'s link rewrite walks entity bodies; extending it to a configured set of doc roots fixes the archive case at the sweep. A relocation epic includes `CHANGELOG.md` in its scan on the same principle.
- **Exempt what is deliberately frozen.** `CHANGELOG.md`'s historical links are excluded from link-check the way its content is already exempted from other doc-lint rules, so a moved target doesn't retroactively break an already-published release note.
- **Detect earlier.** link-check runs in CI on markdown pushes; a local scan over the paths a sweep is about to vacate catches the archive case at the sweep instead.

The second applies only to the CHANGELOG instance, and trades detection for silence — right for frozen content, wrong for live docs.

The third is the live one, and the measurement narrows why. `CHANGELOG.md` is not outside the detector's reach: measured at `origin/main` da34c1009, moving `docs/archive/pocv3/gaps-pre-migration.md` produced a `link-check` error attributed to `CHANGELOG.md` itself. `.lychee.toml` excludes `docs/archive` and `work` from the files lychee *reads*, not from the destinations it resolves, so a frozen entry linking into either is checked like any other line. Both instances in this gap were therefore reported before a human found them; both were found at a release cut anyway. So what is skipped is the *repair* at the sweep, not the *report* — and moving the report to the sweep is what the first and third sketches do. The second would move it the other way, and is only defensible if `CHANGELOG.md`'s frozen links are judged not worth resolving at all.
