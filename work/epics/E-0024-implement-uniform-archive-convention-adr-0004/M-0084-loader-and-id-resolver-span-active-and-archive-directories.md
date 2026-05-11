---
id: M-0084
title: Loader and id resolver span active and archive directories
status: done
parent: E-0024
tdd: required
acs:
    - id: AC-1
      title: PathKind/IDFromPath recognize archive paths per ADR-0004 storage table
      status: met
      tdd_phase: done
    - id: AC-2
      title: tree.Load walks <kind>/archive/ and yields archived entities
      status: met
      tdd_phase: done
    - id: AC-3
      title: refsResolve resolves id-form references targeting archived entities
      status: met
      tdd_phase: done
    - id: AC-4
      title: aiwf show <id> resolves an archived id without flag opt-in
      status: met
      tdd_phase: done
    - id: AC-5
      title: aiwf history <id> walks across an archive rename via existing trailers
      status: met
      tdd_phase: done
    - id: AC-6
      title: Loader cost is bounded; archive-empty trees pay no extra cost
      status: met
      tdd_phase: done
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

- **AC-1 (red→green).** Added `TestPathKind_Archive` and `TestIDFromPath_Archive` covering the full ADR-0004 storage table; updated `entity.PathKind` and `entity.IDFromPath` to accept archive-shaped paths via a small `stripArchiveSegment` helper.
- **AC-2 (red→green).** `tree.Load` already walks every kind directory recursively; the work was extending `PathKind` (above) so archive-located files classify. Added `TestLoad_ArchivedEntities` covering every storage-table row and asserting `Strays` stays empty.
- **AC-3 (red→green).** Active→archive refs resolve cleanly through the existing canonicalized id index; added `TestRefsResolve_ResolvesArchivedTargets` driving an active milestone whose parent is an archived epic, an active gap addressing an archived milestone, and an active ADR superseding an archived ADR.
- **AC-4 (red→green).** `aiwf show <id>` reads through `tree.Load + t.ByID`; once the loader picks up archive entities, `show` resolves them with no API changes. `TestRun_ShowResolvesArchivedID` drives the dispatcher and asserts on the JSON envelope's `path` field as the structural proof.
- **AC-5 (red→green).** `readHistoryChain` greps `git log` purely on trailer values, so the cross-rename walk works for free; `TestRun_HistoryAcrossArchiveRename` pins both halves of the rename sequence.
- **AC-6 (red→green).** Loader cost: the archive walk reuses `filepath.WalkDir`'s existing recursion. `TestLoad_ArchiveEmptyTreeBoundedCost` builds the same fixture twice — once with empty archive dirs, once without — and asserts identical loader output.
- **Surfaced for M-0086 follow-up.** AC-1 work flipped `TestRewidth_ArchivePreservedByteIdentical` (narrow archive entries trip `frontmatter-shape`). M-0086 scoped the shape-and-health rules to skip archive; the workaround in this test was reverted there.

## Decisions made during implementation

- **`stripArchiveSegment` is a helper, not an interface extension.** The active-shape switches in `PathKind` and `IDFromPath` stay unchanged — we strip the archive segment and run them as-is. Considered adding archive-aware branches per kind, but rejected: the storage table's shape is uniform (one `archive` segment at the per-kind position), and the helper keeps that uniformity visible.
- **Archive segment recognition is location-keyed, not name-keyed.** A directory literally named `archive` inside `work/notes/` (or any other non-recognized parent) is NOT classified as archive. Per ADR-0004 §"Storage": "one level deep `archive/` subdirectory" at the per-kind position only. Tested via the negative cases — `work/notes/archive/x.md` → false.

## Validation

- `go test -race ./...` — all packages green.
- `golangci-lint run` — 0 issues.
- `aiwf check` — 0 errors. Warnings are pre-existing pending-sweep advisories on the kernel's own tree.
- The downstream recovery test M-0086 added (`TestRewidth_ArchivePreservedByteIdentical` without the `--skip-checks` workaround) is green, confirming the archive-shape scoping landed cleanly.

## Deferrals

- (none)

## Reviewer notes

- **Helper placement.** `stripArchiveSegment` and `removeAt` are package-private to `entity`. `IsArchivedPath` (added in M-0086) consumes the same helpers; this M-0084 placement made the re-use trivial.
- **`PathKind`'s kind argument is empty for the archive-segment recognition** — the function accepts an empty kind in `stripArchiveSegment` so the path-shape walk classifies uniformly before the per-kind switch. `IDFromPath` passes its kind argument so the strip only applies at the expected per-kind position.

### AC-1 — PathKind/IDFromPath recognize archive paths per ADR-0004 storage table

`internal/entity/entity.go::PathKind` and `entity.IDFromPath` accept the per-kind archive shapes from the ADR-0004 storage table (`work/<kind>/archive/...` and `docs/adr/archive/...`) alongside their active counterparts. The classification logic is uniform across kinds — a small `stripArchiveSegment` helper normalizes the archive form to active form before the existing shape switches run, so no per-kind opt-in branches were introduced. Mechanical evidence: `TestPathKind_Archive` and `TestIDFromPath_Archive` in `internal/entity/entity_test.go` enumerate every storage-table row and a representative set of negative cases (deeper nesting, no-id directory names).

### AC-2 — tree.Load walks <kind>/archive/ and yields archived entities

Once `PathKind` accepts archive shapes, `filepath.WalkDir` (already recursive) classifies archive files as recognized entities, populates `Tree.Entities`, and exposes them via `tr.ByID`. Mechanical evidence: `TestLoad_ArchivedEntities` in `internal/tree/tree_test.go` seeds an active+archive fixture covering every ADR-0004 archive row, asserts entity-count and per-id presence, and confirms `tr.Strays` stays empty (archive paths must not surface under tree-discipline).

### AC-3 — refsResolve resolves id-form references targeting archived entities

Active → archived references resolve cleanly through the existing canonical-id index. The seam is tree.Load → entity classification → reverse-ref index → refsResolve lookup; with AC-1 and AC-2 in place, archive entities sit in the same index as active ones. Mechanical evidence: `TestRefsResolve_ResolvesArchivedTargets` in `internal/check/check_test.go` builds an on-disk fixture with an active milestone whose `parent` points at an archived epic, an active gap whose `addressed_by` points at an archived milestone, and an active ADR whose `supersedes` points at an archived ADR — `refsResolve` returns zero findings.

### AC-4 — aiwf show <id> resolves an archived id without flag opt-in

`aiwf show <id>` reads through `tree.Load` + `tr.ByID`; once the loader picks up archive entities, `show` resolves them with no API surface changes. Mechanical evidence: `TestRun_ShowResolvesArchivedID` in `cmd/aiwf/show_cmd_test.go` drives the in-process dispatcher (`run([]string{"show", ...})`) against an on-disk archive fixture and asserts on the JSON envelope's `id`, `status`, and `path` fields — the path field is the structural proof that the lookup landed on the archived file.

### AC-5 — aiwf history <id> walks across an archive rename via existing trailers

`readHistoryChain` greps `git log` purely on `aiwf-entity:` / `aiwf-prior-entity:` trailers — it is path-agnostic. The seam test pins both halves: pre-rename commit (file in active dir) and post-rename commit (file in archive dir, after `git mv`) both produce trailered events under the same id; history returns both. Mechanical evidence: `TestRun_HistoryAcrossArchiveRename` in `cmd/aiwf/show_cmd_test.go` constructs the rename sequence directly and asserts on the two-event JSON envelope (verb sequence: `add`, `archive`).

### AC-6 — Loader cost is bounded; archive-empty trees pay no extra cost

The archive walk is not a separate scan — it leverages `filepath.WalkDir`'s existing recursion. A tree without archive content (or with empty `<kind>/archive/` directories) yields the same `Entities`, `Strays`, and `LoadError` counts as a tree without those subdirectories. Mechanical evidence: `TestLoad_ArchiveEmptyTreeBoundedCost` in `internal/tree/tree_test.go` builds the same active fixture twice — once with empty archive directories present, once without — and asserts the loader output is identical across the pair.

