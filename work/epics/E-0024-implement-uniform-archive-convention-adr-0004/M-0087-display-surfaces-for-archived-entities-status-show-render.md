---
id: M-0087
title: Display surfaces for archived entities (status, show, render)
status: in_progress
parent: E-0024
depends_on:
    - M-0086
tdd: required
acs:
    - id: AC-1
      title: aiwf status surfaces sweep-pending count when non-zero
      status: open
      tdd_phase: red
---

# M-0087 — Display surfaces for archived entities (status, show, render)

## Goal

Wire archive awareness into the user-facing display surfaces: `aiwf status`'s tree-health section gains a sweep-pending line, `aiwf show` indicates archived state and resolves any id without flag opt-in, and `aiwf render --format=html` segregates per-kind index pages so the active-set is the default home view while the full set remains reachable and per-entity pages render regardless of status.

## Context

M-0084–M-0086 made archive load, sweep, and check correctly. The user-visible layer still treats the active dir as the only world. After this milestone, an operator scanning `aiwf status` sees pending sweeps inline; an operator looking up a closed gap by id gets the page without `--archived` ceremony; a render consumer browsing the site sees an active-default home with a one-click full-set escape hatch.

## Acceptance criteria

<!-- ACs added via `aiwf add ac M-0087 --title "..."` at start time. -->

Intended landing zone:

- `aiwf status` adds a tree-health one-liner *"Sweep pending: N terminal entities not yet archived (run `aiwf archive --dry-run` to preview)"* hidden when N is 0.
- `aiwf status` remains active-only — no `--archived` flag.
- `aiwf show <id>` resolves any id (active or archived) without flag opt-in; the rendered output indicates archived state visibly.
- `aiwf render --format=html` per-kind index pages render active-only by default (the page reachable from home nav).
- A separate `<kind>/all.html` page renders the full set; static `<a>` nav links between active-default and all-set indices.
- Per-entity HTML pages render regardless of status — deep links from external sources don't 404 on archived entities.

## Constraints

- `aiwf list` already supports `--archived` (per existing flag); this milestone does not change `list` semantics.
- No JavaScript layer for `aiwf render` — static `<a>` nav only. JS-driven filter chips / view-switching are explicitly deferred per ADR-0004.
- `aiwf history <id>` already follows path renames via the trailer model — no changes needed; verify the cross-archive case under test.

## Design notes

- Sweep-pending count comes from `archive-sweep-pending` (M-0086 finding); status hides the line when count is 0.
- Per-entity HTML render path is location-agnostic: the renderer reads from the loader's id-resolved view, not from a directory walk.

## Surfaces touched

- `internal/verb/status/`
- `internal/verb/show/`
- `internal/render/html/` (per-kind index, all-set index, per-entity page)

## Out of scope

- The `archive.sweep_threshold` config knob (M-0088).
- Embedded `aiwf-archive` skill (M-0088).
- CLAUDE.md amendment (M-0088).
- Filter-chip / JS view-switching for the render site (deferred per ADR-0004).

## Dependencies

- M-0086 — `archive-sweep-pending` finding produces the count consumed by `aiwf status`.
- ADR-0004 (accepted) — *Display surfaces* section.

## References

- [ADR-0004](../../../docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — *Display surfaces* section.

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

### AC-1 — aiwf status surfaces sweep-pending count when non-zero

