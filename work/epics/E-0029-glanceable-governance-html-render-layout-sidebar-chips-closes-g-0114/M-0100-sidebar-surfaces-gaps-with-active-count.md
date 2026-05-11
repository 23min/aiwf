---
id: M-0100
title: Sidebar surfaces gaps with active count
status: draft
parent: E-0029
depends_on:
    - M-0099
tdd: required
---
# Sidebar surfaces gaps with active count

## Goal

The sidebar (rendered on every page) gains a "Gaps (N)" entry where N is the count of non-archived gaps. Clicking it lands on the chip-filtered `gaps.html` default (active) view. The entry appears on every page; no per-page conditional.

## Context

Today the sidebar surfaces Project status, Overview, and the epic/milestone hierarchy. Gaps — one of the project's primary current-state surfaces ("what's open right now?") — are reachable only by scrolling to the small "Browse by kind" block at the bottom of `index.html`. G-0114 names this as a glanceability failure: a reader landing on any page can't pick up the gap surface without hunting.

M-β (chip filter) lands first so the sidebar gap entry targets the unambiguous single `gaps.html` file (post-migration, `gaps-all.html` no longer exists; the chip handles the active/all toggle). M-α (layout overhaul) lands first so the wider sidebar comfortably hosts the new entry with its active-count display.

The active count specifically (not the total) is the right signal for the sidebar: it matches what the page's default chip view shows. Total / archived breakdown is visible via the home page's "Browse by kind" block and the chip's `[All]` view.

## Acceptance criteria

ACs added via `aiwf add ac M-<id>` at start-milestone time. The observable-behavior space this milestone covers:

- Every rendered page's sidebar includes a "Gaps (N)" entry where N is the count of non-archived gaps in the planning tree at render time.
- The entry sits in the sidebar's top section alongside "Project status" and "Overview" — above the epic list, not inside it.
- The entry's link target is `gaps.html` (the chip-filtered single file from M-β).
- The count N reflects gaps with paths under `work/gaps/` (not `work/gaps/archive/`); the count is recomputed on every render.
- The entry uses the sidebar's existing aria-current pattern: when the current page is `gaps.html`, the entry carries `aria-current="page"` and renders with the active-link styling.
- The `SidebarData` struct gains a `GapCount` field (or equivalent — final field naming decided in implementation); the default resolver populates it; the sidebar template reads it.
- The entry renders even when the count is zero (consistent surface), displaying "Gaps (0)" rather than disappearing.
- All existing sidebar tests pass; new **Playwright** tests in `e2e/playwright/tests/` verify the gap-entry presence across page kinds (index, epic, milestone, entity, kind-index, status) and assert the count value matches the fixture tree's active-gap count. Clicking the sidebar entry navigates to `gaps.html` and the resulting page shows the active subset. Parsed-HTML assertions in Go remain for emit-shape (sidebar entry is present in every rendered page's markup) but the user-visible navigation behavior is browser-verified. CI integration deferred per the epic Constraints; Playwright runs locally.

A render-against-real-fixture human-verification pass closes the milestone per CLAUDE.md *Render output must be human-verified before the iteration closes* — open multiple page kinds, verify the sidebar entry appears with the correct count, verify clicking it lands on `gaps.html` with the active subset visible.

## Constraints

- **Active count only, not total.** The sidebar shows the count matching the page's default chip view (active). Total and archived breakdowns are visible via the home page's kind-index nav and the chip's `[All]` view.
- **Count recomputed at render time.** No cached or stamped count; the in-memory tree count is the source of truth.
- **No new entry per kind.** Only gaps gets a sidebar entry in this epic. Decisions / ADRs / contracts stay reachable via the home page's "Browse by kind" block. Per the epic's *Out of scope*.
- **Entry rendered even at zero count.** Consistent surface shape; the count display "Gaps (0)" is the truthful state, not absence.

## Design notes

- The entry's position above the epic list (top section) matches the existing pattern: Project status and Overview sit in `.sidebar-top` and the epic `<details>` blocks follow.
- The count display is parenthetical to match how other sidebar entries elsewhere in the kernel surface counts (e.g. the home page's `(33 active, 79 archived)` line — though that's not parenthetical, the parenthetical-N form is the lighter visual choice for a sidebar where every row is tight).
- The cmd-side resolver populates `GapCount` from the same tree-walk it already does; no new IO. The default resolver does the same for its tests.

## Surfaces touched

- `internal/htmlrender/embedded/_sidebar.tmpl` (primary — new `<li>` in `.sidebar-top` with "Gaps (N)")
- `internal/htmlrender/pagedata.go` (add `GapCount` to `SidebarData`)
- `internal/htmlrender/default_resolver.go` (populate `GapCount` in `sidebar()` helper)
- `cmd/aiwf/render_resolver.go` (cmd-side resolver — same)
- `e2e/playwright/tests/` (primary test surface — extend `render.spec.ts` with sidebar gap-entry presence + count + click-through tests)
- `internal/htmlrender/htmlrender_test.go` (sidebar gap-entry emit-shape; complementary)
- `cmd/aiwf/render_archive_visibility_test.go` (sidebar count reflects archive state; complementary)

## Out of scope

- Same sidebar treatment for decisions / ADRs / contracts. Defer until the gaps pattern proves out.
- Sub-list of recent / open gaps inside the sidebar — just the entry + count, no enumeration.
- Per-kind sidebar entries with chips embedded in the sidebar.
- In-page status hierarchy in gaps.html — M-δ.
- Surfacing the count anywhere else (page header, status report) — sidebar only.

## Dependencies

- M-β (chip filter) — depends_on. The entry's link target is the chip-filtered single `gaps.html`; M-β must land first so the target is unambiguous.

## References

- E-0029 (parent epic)
- G-0114 (gap closed)
- `internal/htmlrender/embedded/_sidebar.tmpl` — existing sidebar partial
- `CLAUDE.md` — *Substring assertions are not structural assertions*, *Render output must be human-verified before the iteration closes*

## Work log

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
