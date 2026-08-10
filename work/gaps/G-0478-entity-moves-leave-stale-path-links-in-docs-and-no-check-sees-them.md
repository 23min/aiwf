---
id: G-0478
title: Entity moves leave stale path links in docs/ and no check sees them
status: open
priority: medium
---
## What's missing

Moving an entity file breaks every path-based markdown link that names its old location, and nothing reports it.

Every mover rewrites links, and every mover's walk stops at the entity tree. `aiwf archive` relocates entities into their per-kind `archive/` subdirectory and repairs the entity bodies that linked to them, through `planArchiveRewrites`. `aiwf rename` and `aiwf retitle` do the same through `planLinkRewriteWrites`. Both walks iterate the loaded tree's entities; neither reads `docs/`. So a link written from a doc to an entity file has no maintainer.

`aiwf check` cannot catch the result either. Cross-references between entities resolve by id, and the loader resolves ids across both active and archive trees, so the entity-side guarantee holds. A markdown link is a *path*, and no rule resolves one.

## Why it matters

Measured on this tree: 59 relative links point from `docs/` into `work/`. Four of them have broken, from two separate move events:

- The archive sweep that closed G-0469 vacated its active path; the two links to it in `docs/initiatives/quality-signal-and-cadence.md` were repaired by hand in the same session.
- Two links in that same document still name `work/epics/E-0073-mutating-verb-ux-uniformity/epic.md` and its `M-0281` sibling. Both now live under `work/epics/archive/`. Nothing reported them; they were found only by walking the links directly.

The second pair is the point. The first pair was caught because someone happened to be watching the sweep that caused it. Rot that arrives with no signal is found by accident or not at all, and every sweep adds more.

It has since recurred, which is the argument for acting rather than watching. On 2026-08-10 the `link-check` workflow had been red for six consecutive runs — spanning a scheduled run and two unrelated pushes — over three links naming `work/gaps/G-0559-…` and `work/gaps/G-0438-…`, both swept to `archive/`. G-0439 logs the same shape at the v0.31.0 cut, red for nine runs. Each time the repair was a hand-edit, and each time it was found by someone looking at an unrelated failure. A gate that is red by default reports nothing about the change that turned it red, so the rot's cost is not the broken link but the signal it takes down with it.

E-0063 scoped `docs/` out of link rewriting deliberately, so this is a known boundary rather than an oversight — but the boundary was drawn on the assumption that the id-resolution guarantee covers what matters. It does not extend to prose that names a path. The Archival tier's forget-by-default convention is likewise not a defence: it exempts links *inside* `docs/archive/`, not links *into* `work/` from a Normative or Forward-looking document.

## Resolution shape

Two independent halves. Either one closes most of the exposure, and they are worth sequencing rather than bundling.

**Detection** is the smaller change and the better first move: a check rule that resolves every relative markdown link whose target is a `.md` file under `work/`, and reports the ones naming no existing file. It needs no id semantics — whether a path exists is a filesystem question — and it catches the cases prevention cannot, including a link that was wrong the moment it was typed. Scope it to the live documentation tiers; `docs/archive/` is exempt by the same forget-by-default convention that already exempts it elsewhere.

**Prevention** is widening the rewrite: extend the walk past the entity tree so a move repairs `docs/` too. The machinery to do it already runs on every sweep — what is missing is reach, not a rewriter. This is the larger change, and on its own it still leaves hand-authored mistakes unreported.

Worth settling as part of this: whether path links into `work/` should be discouraged in favour of bare ids, which the loader already resolves and which survive every move. That would shrink the surface rather than police it.

## Where to fix

- `internal/verb/linkrewrite_ops.go` — the rewrite walk's scope.
- `internal/verb/archive.go` — `planArchiveRewrites` already repairs entity bodies; the scope stops there.
- `internal/check/` — the new path-link rule, if detection is chosen first.
- `CLAUDE.md` §"Documentation hierarchy" — the tier boundary that decides which docs a rule holds to.
