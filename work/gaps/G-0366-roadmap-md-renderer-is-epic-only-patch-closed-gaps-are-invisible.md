---
id: G-0366
title: ROADMAP.md renderer is epic-only; patch-closed gaps are invisible
status: open
---
## Problem

`aiwf render roadmap`'s renderer (`internal/roadmap/roadmap.go`, `render()`)
builds the entire document from `t.ByKind(entity.KindEpic)` — it never
reads gaps, decisions, or ADRs directly. An entity only appears in
`ROADMAP.md` if some epic's title or body prose happens to mention it
(`"closes G-NNNN"`).

`wf-patch` exists specifically for changes "too small to warrant a
milestone" and never creates an epic. So a gap closed via a patch is
**structurally invisible** to `ROADMAP.md`, regardless of when the
renderer runs or how fresh its regeneration is. Confirmed directly: none
of G-0363, G-0361, G-0352, G-0332, G-0350, G-0359, G-0271, G-0224, G-0199
(all closed via `patch/*` merges, all with real, substantive fixes) appear
anywhere in `ROADMAP.md`.

## Why it matters

`ROADMAP.md` is the committed, at-a-glance "what's the state of this
project" document. Right now it silently under-represents real, shipped
work — a reader has no way to learn from it that a patch closed a gap,
even though the same information is (in principle) available in git
history and (inconsistently — see the sibling gap on `wf-patch`'s missing
CHANGELOG step) in `CHANGELOG.md`. The roadmap and the changelog end up
splitting responsibility in a way nobody decided on purpose: roadmap for
epic progress, changelog for patch-level deltas, with no section that
actually correlates the two or gives an operator one place to see
everything closed recently regardless of mechanism.

## Timing note — G-0350 makes the fix cheaper now

G-0350 (addressed) decoupled `aiwf render roadmap --write` from committing:
it now only writes the file (no stash dance, no commit, runs on a dirty
tree). That makes it cheap and safe to wire a roadmap regen into
`wf-patch`'s own wrap sequence — riding along with the wrap's already-gated
commit, no new gate needed. But wiring the call in is **necessary and not
sufficient**: today, calling `--write` after a patch-only wrap produces
**zero diff**, because the renderer has nothing to say about a bare closed
gap. The renderer needs a new section before the call is worth adding.

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
- Once this section exists, wire `aiwf render roadmap --write` into
  `wf-patch`'s wrap sequence (see the timing note above) so the roadmap
  actually reflects patch activity without adding a new gate.

## Provenance

Found 2026-07-05 while answering "what aiwf command shows recently closed
gaps" — `ROADMAP.md` turned out to have no representation of patch-closed
work at all, confirmed by checking recent `patch/*` merges' gap ids
against the rendered roadmap directly.
