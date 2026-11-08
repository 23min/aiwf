---
id: G-0366
title: ROADMAP.md renderer is epic-only; patch-closed gaps are invisible
status: open
priority: medium
---
## Problem

`aiwf render roadmap`'s renderer (`internal/roadmap/roadmap.go`, `render()`)
builds the entire document from `t.ByKind(entity.KindEpic)` — it never
reads gaps, decisions, or ADRs directly. An entity only appears in the
roadmap if some epic's title or body prose happens to mention it
(`"closes G-NNNN"`).

`wf-patch` exists specifically for changes "too small to warrant a
milestone" and never creates an epic. So a gap closed via a patch is
**structurally invisible** to the roadmap, regardless of when the
renderer runs or how fresh its regeneration is. Confirmed directly: none
of G-0363, G-0361, G-0352, G-0332, G-0350, G-0359, G-0271, G-0224, G-0199
(all closed via `patch/*` merges, all with real, substantive fixes) appear
anywhere in `ROADMAP.md`.

## Why it matters

The roadmap, rendered on demand by `aiwf render roadmap`
([D-0106](../decisions/D-0106-stop-committing-roadmap-md-render-it-on-demand.md)),
is the at-a-glance "what's the state of this project" view. It silently
under-represents real, shipped work — a reader has no way to learn from it
that a patch closed a gap, even though the same information is in git
history and in `CHANGELOG.md`, where `wf-patch` adds an entry for every
patch. The roadmap and the changelog end up splitting responsibility in a
way nobody decided on purpose: roadmap for epic progress, changelog for
patch-level deltas, with no section that actually correlates the two or
gives an operator one place to see everything closed recently regardless
of mechanism.

## Direction (not prescribed)

- Add a generated section — something like `## Recent patches` or
  `## Recently closed gaps` — listing gaps whose only path to `addressed`/
  `wontfix` was a patch (or, more simply, any recently-closed gap not
  already covered by an epic's own `"(closes G-NNNN)"` mention), each with
  id, title, and the closing commit/patch branch.
- Decide the window: every terminal gap not archived yet, or a
  time-bounded "since last release" list (mirroring `CHANGELOG.md`'s
  `[Unreleased]` framing) — the two give different answers once archiving
  catches up.

## Provenance

Found 2026-07-05 while answering "what aiwf command shows recently closed
gaps" — `ROADMAP.md` turned out to have no representation of patch-closed
work at all, confirmed by checking recent `patch/*` merges' gap ids
against the rendered roadmap directly.
