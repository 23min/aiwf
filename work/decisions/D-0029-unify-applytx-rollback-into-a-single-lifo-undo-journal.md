---
id: D-0029
title: Unify applyTx rollback into a single LIFO undo journal
status: proposed
relates_to:
    - M-0186
---
## Question

M-0186/AC-3 retrofits `verb.Apply` onto the temp-index commit primitive and needs correct rollback of partial failures, including an `OpMove` that relocates an entire directory (not just a flat file) — directories can't be captured as a single byte blob the way a file's content can. The implementation as first written tracked directory moves (`applyTx.dirMoves`) separately from flat-file content capture (`applyTx.preApply` / `touchedPaths`), with `rollback()` always reversing directory moves first, before restoring any captured file content. Is that two-mechanism, fixed-order design correct, or does it need to change before AC-3 lands?

## Decision

Rewrite `applyTx`'s rollback bookkeeping into a single chronological undo journal (one `undoStep` per completed mutation — a directory/file rename or a captured file-content write — appended in execution order) and reverse it strictly LIFO, replacing the separate `dirMoves` list and the fixed "directories first" processing order in `rollback()`.

## Reasoning

A `wf-rethink` pass (fresh subagent, independent reconstruction from a pinned obligation list, no sight of the implementation) converged on the same journal-based design as the from-scratch answer. Comparing it against the shipped design surfaced a real bug, not just a stylistic difference: for a plan that moves a directory *and* rewrites a file nested inside it, and a later step fails —

- The shipped design's fixed order reverses the directory move first, *then* tries to restore the nested file's pre-rewrite bytes at its old (now-vacated) new-location path — producing a stray duplicate file there while the file at the real (moved-back) location keeps the bad rewritten content. Worse than the "known limitation" comment on that code claimed (it said rollback just "keeps whatever ended up inside it" — a no-op, not an active new bug).
- The journal design's LIFO order restores the nested file's pre-rewrite bytes *before* reversing the directory move, so the directory carries the now-correctly-restored file back with it in one rename. No stray file, no lost restoration.

No existing test exercised this composite scenario (directory move + nested rewrite + downstream failure) — it was found by tracing the rethink's comparison, not by a failing test. Alternatives considered: leaving the two-mechanism design and special-casing the ordering (rejected — the fix is exactly "stop special-casing order," so keeping two mechanisms and manually sequencing them correctly is strictly more complex than one journal that's LIFO by construction); documenting the composite case as a permanent, unfixed limitation (rejected — the fix is not materially harder than documenting around it, and the bug is more severe than the documentation claimed).

## Consequences

- `internal/verb/apply.go`: `applyTx.dirMoves` and the flat-file `preApply`/`touchedPaths` split are replaced by a single ordered journal; `rollback()` becomes one LIFO loop instead of two passes in a fixed order.
- A new test pins the composite directory-move + nested-rewrite + downstream-failure scenario before the rewrite lands (red), confirming the journal design resolves it (green).
- No change to `gatherCommitOps` or the `(removes, writes)` commit-set computation — that responsibility stays a separate, later, disk-state-driven pass, unaffected by how rollback bookkeeping is structured.
