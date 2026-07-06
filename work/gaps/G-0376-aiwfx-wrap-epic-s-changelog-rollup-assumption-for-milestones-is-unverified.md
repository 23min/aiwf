---
id: G-0376
title: aiwfx-wrap-epic's CHANGELOG rollup assumption for milestones is unverified
status: open
prior_ids:
    - G-0368
---
## Problem

`aiwfx-wrap-epic`'s CHANGELOG step (step 7) assumes every milestone's user-visible
delta ends up captured in the epic's own single `CHANGELOG.md` entry, added once at
wrap time. But `aiwfx-wrap-milestone` has no CHANGELOG step of its own — grepping its
`SKILL.md` for "changelog" returns zero hits, mirroring what G-0365 found for
`wf-patch` before that fix — and nothing forces the epic-wrap author to enumerate
every milestone's user-visible change when writing that one entry. `aiwfx-release`'s
own skill only moves whatever already sits under `[Unreleased]`; it never synthesizes
entries from milestone or wrap history.

G-0365 raised this as an aside while fixing the unambiguous, adjacent gap: a
`wf-patch` has no parent epic to roll its change into, so its own missing CHANGELOG
step was a clear-cut bug. This gap is the epic/milestone side of the same question,
deliberately left unresolved there to avoid bundling two different fixes into one
patch.

## Why it matters

If an epic ships several milestones and the wrap author's one CHANGELOG entry only
mentions a subset — the most memorable milestone, or the last one worked — the rest
go unrecorded: the same "real, shipped, user-visible change with zero CHANGELOG
trace" failure mode G-0365 fixed for patches, one level up. This case is easier to
miss than the patch case, because it has a plausible-looking safety net: `wrap.md`
already enumerates every milestone delivered. It is tempting to assume that
enumeration is equivalent to a CHANGELOG record, but `wrap.md` and `CHANGELOG.md` are
different documents for different audiences with no mechanical link between them —
whether the assumption actually holds in practice is exactly what this gap should
settle.

## Direction (not prescribed)

Audit past epic CHANGELOG entries against their `wrap.md` milestone lists: does every
delivered milestone get at least a mention in the epic's entry, or have some been
silently dropped? If gaps turn up, candidate fixes to weigh when this is worked:

1. **A structural check** — every milestone listed in an epic's `wrap.md` is
   referenced (by id) somewhere in that epic's `CHANGELOG.md` entry.
2. **Push the source of truth down** — have `aiwfx-wrap-milestone` append a short
   changelog-relevant note to the milestone's own record (not `CHANGELOG.md`
   directly, since a milestone isn't independently releasable), which the epic wrap
   step then folds in verbatim instead of re-deriving each milestone's user-visible
   delta from memory.

## Provenance

Raised as an aside in G-0365 ("worth confirming that's actually the intended design
rather than the same gap") while adding `wf-patch`'s own missing CHANGELOG-entry
step; deliberately scoped out of that patch per `wf-patch`'s own anti-pattern rule
against bundling unrelated fixes into one change.
