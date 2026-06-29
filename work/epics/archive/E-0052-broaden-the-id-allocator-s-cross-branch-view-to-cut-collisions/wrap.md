# Epic wrap — E-0052

**Date:** 2026-06-29
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0052-broaden-the-id-allocator-s-cross-branch-view-to-cut-collisions
**Merge commit:** 180cd446

## Milestones delivered

- M-0212 — Union all local refs into the allocator's cross-branch id view (merged 80a421f0)
- M-0213 — Opt-in best-effort fetch before id allocation (merged 0e060be3)
- M-0214 — Broaden allocator and --fetch to all remote-tracking refs (merged 5691528d)

## Summary

Widened the trunk-aware id allocator's cross-branch view from `{working tree + one
trunk ref}` to the full published surface — `{working tree + all local refs/heads +
all remote-tracking refs/remotes + trunk}` — so the dominant collision classes are
caught at allocation time instead of surfacing at push via `aiwf reallocate`. The
widened set feeds allocation (prevention) ONLY; the `ids-unique` check keeps its
working-tree-vs-trunk basis, so the same entity present on two branches is never
false-flagged. `aiwf add --fetch` opt-in-refreshes the published view before
allocating (`git fetch --all`, best-effort, never blocks).

Scope expanded mid-flight: the epic spec planned two milestones (local-refs scan +
trunk fetch); the operator then chose to also build the remote-side view (M-0214,
closing G-0316), which broadened `--fetch` from M-0213's single-branch trunk refresh
to `git fetch --all` — a deliberate supersession. The stable-id-from-creation model is
preserved entirely (no inbox, mint, or slug phase); this is the cheap, model-preserving
point on the axis whose structural endpoint remains ADR-0001 (mint at integration).

## ADRs ratified

- ADR-0025 — Allocator's cross-branch view spans all refs, fed to allocation only
  (accepted). Records the widened-view decision and relates it to ADR-0001 (proposed,
  the structural endpoint deferred by this epic).

## Decisions captured

- The allocator's published view = all refs (local + remote + trunk), fed to allocation
  only — documented in the M-0212 and M-0214 spec Goal/Decisions sections and in
  `CLAUDE.md` §"Id-collision resolution at merge time".
- `--fetch` evolved from trunk-only (M-0213) to `git fetch --all` (M-0214) — captured in
  the M-0214 Decisions section.

## Follow-ups carried forward

- G-0274 — No batch resolution for id collisions; reallocate is one-at-a-time (open; the
  cure side, deliberately out of E-0052's prevention scope).
- G-0308 — promote-on-wrong-branch mis-attributes commits across a reallocation (open;
  cure side).
- ADR-0001 — Mint entity ids at trunk integration (proposed; the structural endpoint for
  team / sustained-parallel-agent scale, against which this epic was the cheap-now point).

## Handoff

The collision-**prevention** side is complete: the allocator now sees every locally-known
published id (local + remote branches + trunk), refreshable via `--fetch`. The residual
race — a teammate who has allocated but not pushed — stays `aiwf reallocate`'s to cure,
and the **cure** side (batch reallocation, G-0274; promote mis-attribution, G-0308)
remains open for a future epic. ADR-0001 is the structural alternative if team-scale
parallel-agent friction ever justifies it.
