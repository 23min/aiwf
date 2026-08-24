---
id: G-0478
title: Entity moves leave stale path links in docs/, reported only after the push
status: open
priority: high
---
## What's missing

Moving an entity file breaks every path-based markdown link that names its old location, and the only thing that reports it does so after the push, into a run that is usually already red.

The `link-check` workflow runs lychee over the markdown its `./**/*.md` glob reaches and `.lychee.toml` does not exclude, on markdown-touching pull requests, markdown-touching pushes to `main`, and a weekly schedule; it does resolve `docs/`-to-`work/` destinations: measured at `origin/main` da34c1009, retitling a linked gap took it from three errors to nine, the six new ones naming the vacated path. So detection is not absent, and this gap is not a request to build it. What is missing is a report that arrives while the mover is still in hand, and one that is legible when it arrives. On this repo's trunk-based flow the wait is longer than "after the push" suggests: no PR is required, and a push to an epic or milestone branch triggers nothing, so the first run that can see the break is the one after the merge to `main`.

Every mover rewrites links, and every mover's walk stops at the entity tree. `aiwf archive` relocates entities into their per-kind `archive/` subdirectory and repairs the entity bodies that linked to them, through `planArchiveRewrites`. `aiwf rename` and `aiwf retitle` do the same through `planLinkRewriteWrites`. Both walks iterate the loaded tree's entities; neither reads `docs/`. So a link written from a doc to an entity file has no maintainer.

`aiwf check` cannot catch the result either. Cross-references between entities resolve by id, and the loader resolves ids across both active and archive trees, so the entity-side guarantee holds. A markdown link is a *path*, and no rule resolves one.

## Why it matters

Measured 2026-07-30: 59 relative links point from `docs/` into `work/` — 78 at `origin/main` da34c1009, of which lychee reads 73. Four had broken then, from two separate move events:

- The archive sweep that closed G-0469 vacated its active path; the two links to it in `docs/initiatives/quality-signal-and-cadence.md` were repaired by hand in the same session.
- Two links in that same document named `work/epics/E-0073-mutating-verb-ux-uniformity/epic.md` and its `M-0281` sibling after both moved under `work/epics/archive/`. They were found by walking the links directly — not because nothing reports that class, but because the report lands in a CI run nobody was reading. Repaired since, in `5e4750dc9`.

The second pair is the point. The first pair was caught because someone happened to be watching the sweep that caused it. Rot that arrives with no signal is found by accident or not at all, and every sweep adds more.

It has since recurred, which is the argument for acting rather than watching. On 2026-08-10 the `link-check` workflow had been red for six consecutive runs — spanning a scheduled run and two unrelated pushes — over three links naming `work/gaps/G-0559-…` and `work/gaps/G-0438-…`, both swept to `archive/`. G-0439 logs the same shape at the v0.31.0 cut, red for nine runs. Each time the repair was a hand-edit, and each time it was found by someone looking at an unrelated failure. A gate that is red by default reports nothing about the change that turned it red, so the rot's cost is not the broken link but the signal it takes down with it.

E-0063 scoped `docs/` out of link rewriting deliberately, so this is a known boundary rather than an oversight — but the boundary was drawn on the assumption that the id-resolution guarantee covers what matters. It does not extend to prose that names a path. The Archival tier's forget-by-default convention is likewise not a defence: it exempts links *inside* `docs/archive/`, not links *into* `work/` from a Normative or Forward-looking document.

## Resolution shape

Two independent halves. Either one closes most of the exposure, and they are worth sequencing rather than bundling.

**Detection** already exists in CI and is the smaller change to *relocate*: a check rule that resolves every relative markdown link whose target is a `.md` file under `work/`, and reports the ones naming no existing file. It needs no id semantics — whether a path exists is a filesystem question — and it catches the cases prevention cannot, including a link that was wrong the moment it was typed. Scope it to the live documentation tiers; `docs/archive/` is exempt by the same forget-by-default convention that already exempts it elsewhere.

A third option is cheaper than either and is already half-built. `auditDanglingEntityRefs` in `internal/policies/no_dangling_entity_refs.go` resolves path-form entity references and reports the ones that no longer exist — exactly this class — and `TestPolicy_NoDanglingEntityRefsInNarrativeDocs` runs it over a two-entry list, `CLAUDE.md` and `ROADMAP.md`, at `make check-fast` and CI tier. Its own comment invites extension: *"If a new narrative doc enters the same pattern, add its path here."* Widening that list to the live `docs/` tiers would cover the measured class without writing a rule, at the policy suite's gate rather than the push's. Note it is the *only* detector for `ROADMAP.md`, which `.lychee.toml` excludes outright. Whether the list should grow file-by-file or become a tier walk is the question that makes this a choice rather than a one-line edit.

Putting that rule in `aiwf check` rather than only in CI is what changes the outcome, and the reason is the gate rather than the logic. lychee already computes the same answer, so a rule here buys no new detection — it buys a pre-push refusal attributable to the commit that caused it, instead of a workflow result that has to be noticed. Measured at `origin/main` da34c1009, `link-check` is red over three links whose targets were swept to `archive/`, which is what a report nobody is obliged to read looks like after a while. The exclusion list needs no id semantics either: `.lychee.toml` names `work` among its `exclude_path` entries, and that filters the files lychee *reads*, never the destinations it resolves.

**Prevention** is widening the rewrite: extend the walk past the entity tree so a move repairs `docs/` too. The machinery to do it already runs on every sweep — what is missing is reach, not a rewriter. This is the larger change, and on its own it still leaves hand-authored mistakes unreported.

Worth settling as part of this: whether path links into `work/` should be discouraged in favour of bare ids, which the loader already resolves and which survive every move. That would shrink the surface rather than police it.

## Where to fix

- `internal/verb/linkrewrite_ops.go` — the rewrite walk's scope.
- `internal/verb/archive.go` — `planArchiveRewrites` already repairs entity bodies; the scope stops there.
- `internal/check/` — the new path-link rule, if detection is chosen first.
- `CLAUDE.md` §"Documentation hierarchy" — the tier boundary that decides which docs a rule holds to.
