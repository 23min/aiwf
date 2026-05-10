---
id: M-0084
title: Loader and id resolver span active and archive directories
status: in_progress
parent: E-0024
tdd: required
---

# M-0084 — Loader and id resolver span active and archive directories

## Goal

Extend `tree.Tree` and the id-resolution paths so that entities under `<kind>/archive/` (and `docs/adr/archive/`) load and resolve identically to active entities. After this milestone, references like `Resolves: G-0018` work whether the target is active or archived; the rest of the kernel can rely on archive being readable but otherwise inert.

## Context

ADR-0004 makes `archive/` a legal location for terminal-status entities. Today the loader walks only the active directory per kind. Without this milestone, every other piece of the epic (the verb, the check rules, the display surfaces) would have to grow its own archive-aware traversal, and the references-stay-valid invariant from the ADR could not hold. This is foundational; nothing else in E-0024 lands cleanly without it.

## Acceptance criteria

<!-- ACs are added via `aiwf add ac M-0084 --title "..."` at `aiwfx-start-milestone` time per the skill's anti-pattern guidance: don't front-load AC detail before work begins. The shape below is the intended landing zone, not committed AC text. -->

Intended landing zone (refine via `aiwf add ac M-0084 --title "..."` when the milestone starts):

- `tree.Tree` reads `<kind>/archive/` for every kind whose ADR-0004 storage row populates an archive location, including `docs/adr/archive/`.
- `internal/check/check.go::refsResolve` and `internal/entity/refs.go::ForwardRefs` resolve ids across active+archive without flag opt-in.
- `aiwf show <id>` and `aiwf history <id>` resolve targets in `archive/` identically to active.
- Loader cost is bounded; no quadratic re-scan; archive-empty trees pay no extra cost.

## Constraints

- Loader behavior must be uniform across kinds — no per-kind conditional opt-in to archive traversal.
- `internal/entity/transition.go::IsTerminal` remains the single source of truth for terminal statuses; the loader does not duplicate that logic.
- No display-surface changes in this milestone — that's M-0087's scope. Listing/render layers may still default to active-only via their own filters; the *loader* must be archive-aware regardless.

## Design notes

- Archive paths follow the per-kind storage table in ADR-0004: directory-shaped kinds (`epic`, `contract`) move whole subtrees; flat-file kinds (`gap`, `decision`, `adr`) move individual files; milestones do not archive independently.
- ADR-0001 (proposed) reserves `inbox/` for pre-mint state; the loader's archive walk must not collide with inbox handling if/when that lands.

## Surfaces touched

- `internal/tree/tree.go` (loader entry points)
- `internal/check/check.go` (`refsResolve`)
- `internal/entity/refs.go` (`ForwardRefs`)

## Out of scope

- The `aiwf archive` verb (M-0085).
- New check-rule findings (M-0086).
- Display-surface filtering (M-0087).

## Dependencies

- ADR-0004 (accepted) — design pinned.
- None among E-0024's milestones (this is the foundational one).

## References

- [ADR-0004](../../../docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — Storage table, id-resolver section.
- `internal/entity/transition.go::IsTerminal`

---

## Work log

(populated during implementation)

## Decisions made during implementation

- (none)

## Validation

(populated at wrap)

## Deferrals

- (none)

## Reviewer notes

- (none)
