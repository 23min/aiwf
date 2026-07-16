---
id: D-0029
title: Unify applyTx rollback into a single LIFO undo journal
status: accepted
relates_to:
    - M-0186
---
# D-0029 — Unify applyTx rollback into a single LIFO undo journal

> **Date:** 2026-07-05 · **Decided by:** human/peter

## Question

M-0186/AC-3 retrofits `verb.Apply` onto the temp-index commit primitive and needs correct rollback of partial failures, including an `OpMove` that relocates an entire directory (not just a flat file) — directories can't be captured as a single byte blob the way a file's content can. The implementation as first written tracked directory moves (`applyTx.dirMoves`) separately from flat-file content capture (`applyTx.preApply` / `touchedPaths`), with `rollback()` always reversing directory moves first, before restoring any captured file content. Is that two-mechanism, fixed-order design correct, or does it need to change before AC-3 lands?

## Decision

Rewrite `applyTx`'s rollback bookkeeping into a single chronological undo journal (one `undoStep` per completed mutation — a directory/file rename or a captured file-content write — appended in execution order) and reverse it strictly LIFO, replacing the separate `dirMoves` list and the fixed "directories first" processing order in `rollback()`.

## Reasoning

A `wf-rethink` pass (fresh subagent, independent reconstruction from a pinned obligation list, no sight of the implementation) converged on the same journal-based design as the from-scratch answer. Comparing it against the shipped design surfaced a real bug, not just a stylistic difference: for a plan that moves a directory *and* rewrites a file nested inside it, and a later step fails —

- The shipped design's fixed order reverses the directory move first, *then* tries to restore the nested file's pre-rewrite bytes at its old (now-vacated) new-location path — producing a stray duplicate file there while the file at the real (moved-back) location keeps the bad rewritten content. Worse than the "known limitation" comment on that code claimed (it said rollback just "keeps whatever ended up inside it" — a no-op, not an active new bug).
- The journal design's LIFO order restores the nested file's pre-rewrite bytes *before* reversing the directory move, so the directory carries the now-correctly-restored file back with it in one rename. No stray file, no lost restoration.

No existing test exercised this composite scenario (directory move + nested rewrite + downstream failure) — it was found by tracing the rethink's comparison, not by a failing test. A second-opinion review (independent subagent, adversarial stance) confirmed the bug by its own trace and found it reachable today through `reallocate` and `rewidth` on epic entities — any Phase-2 write failure on a multi-file epic triggers it — not merely a theoretical op shape.

Alternatives considered: swapping the two existing rollback loops (restore file content before reversing directory moves) is a *smaller* diff and does fix this exact scenario — the second-opinion review confirmed this, correcting an earlier, imprecise version of this reasoning that claimed the journal was simpler. Rejected anyway: a fixed loop order is only correct for *this* interleaving (directory moved, then a file inside it rewritten). The mirror interleaving — a nested path moved, then its parent directory moved — would need the opposite order, and swapped-fixed-order would silently corrupt it. No shipped verb produces that interleaving today, so a loop swap would be correct by accident, not by construction. Since `verb.Apply` is the foundational commit-construction primitive for every mutating verb in E-0045, a single LIFO journal (correct for *any* interleaving, by construction, since it replays actual execution order in reverse) is the right invariant to hold at this layer — a non-obvious ordering argument that would otherwise need re-verification against every future verb's op shape. Documenting the composite case as a permanent, unfixed limitation was also considered and rejected — the fix is not materially harder than documenting around it, and the bug is more severe than the documentation claimed (see below).

## Consequences

- `internal/verb/apply.go`: `applyTx.dirMoves` and the flat-file `preApply`/`touchedPaths` split are replaced by a single ordered journal; `rollback()` becomes one LIFO loop instead of two passes in a fixed order.
- A new test pins the composite directory-move + nested-rewrite + downstream-failure scenario before the rewrite lands (red), confirming the journal design resolves it (green).
- No change to `gatherCommitOps` or the `(removes, writes)` commit-set computation — that responsibility stays a separate, later, disk-state-driven pass, unaffected by how rollback bookkeeping is structured.
