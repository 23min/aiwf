---
id: M-0100
title: Sidebar adds gap entry + epic archive chip filter
status: draft
parent: E-0029
depends_on:
    - M-0099
tdd: required
---
# M-0100 — Sidebar adds gap entry + epic archive chip filter

## Goal

Expand the sidebar's information density on two fronts: (a) add a "Gaps (N)" entry where N is the count of non-archived gaps — closing the "no current-state surface in the sidebar" half of G-0114; and (b) add a `[Active] [All]` chip strip that filters archived epics out of the sidebar's epic list by default, reusing the same `:target`-driven pattern landed in M-0099. Both improvements appear in the sidebar on every rendered page.

## Context

Two adjacent issues surfaced during E-0029 review:

1. **Gaps invisible from the sidebar.** Today the sidebar surfaces Project status, Overview, and the epic/milestone hierarchy. Gaps — one of the project's primary current-state surfaces — are reachable only by scrolling to the small "Browse by kind" block at the bottom of `index.html`. G-0114 names this as a glanceability failure.

2. **All epics (including done ones) crowd the sidebar.** The current sidebar emits every epic in the planning tree as a `<details>` block, regardless of status. For the aiwf repo with 29 epics (most of them `done`), the active in-flight epics drown in the long tail of archived ones. Discovered during M-0099's visual review: a reader scanning the sidebar can't pick up "what's in flight right now" without scrolling past dozens of finished epics.

Both improvements share the sidebar surface and the test scaffolding (sidebar rendered on every page kind); folding them into one milestone keeps the work focused and lets the chip-strip pattern (M-0099) prove out across surfaces. The chip strip uses a different URL fragment (`#sidebar-all`) from the kind-index chip strip's `#all` so the two filters can be toggled independently — a reader on `gaps.html#all` doesn't also reveal archived epics in the sidebar.

M-0099 (kind-index chip filter) lands first so the `.chip-strip` / `.chip` CSS classes exist and can be reused; the sidebar's chip strip is structurally identical, just scoped to the sidebar.

## Acceptance criteria

ACs added via `aiwf add ac M-<id>` at start-milestone time. The observable-behavior space this milestone covers:

**Gap entry half (original M-0100 scope):**
- Every rendered page's sidebar includes a "Gaps (N)" entry where N is the count of non-archived gaps in the planning tree at render time.
- The entry sits in the sidebar's top section alongside "Project status" and "Overview" — above the epic list.
- The entry's link target is `gaps.html` (the chip-filtered single file from M-0099).
- The count N reflects gaps with paths under `work/gaps/` (not `work/gaps/archive/`); recomputed on every render.
- The entry renders even when the count is zero (consistent surface), displaying "Gaps (0)" rather than disappearing.

**Epic archive filter half (broadened scope per user visual review of M-0099):**
- The sidebar's epic list defaults to showing non-archived epics only (statuses `proposed`, `active`). Archived epics (status `done` or `cancelled` with paths under `work/epics/archive/`) are hidden by default via CSS.
- A `[Active] [All]` chip strip renders in the sidebar (placement: after the top section's links, before the epic list) with the same `.chip-strip` / `.chip` markup as M-0099's kind-index chip strip.
- The chip strip uses `#sidebar-active` / `#sidebar-all` URL fragments — different from M-0099's `#active` / `#all` so the two filters toggle independently.
- The CSS filter rule keys off `body:has(#sidebar-all:target)` to reveal archived epics in the sidebar; the kind-index chip filter rule (which keys off `#all:target`) is unaffected.
- Each `<details class="sidebar-epic">` element carries `data-archived="true|false"` so the CSS rule can target archived epics specifically.
- The Active-chip-by-default visual state matches M-0099's pattern: `.chip-strip:not(:has(.chip:target)) #sidebar-active` highlights when no chip is :target.

**Shared:**
- All existing sidebar tests pass; new **Playwright** tests in `e2e/playwright/tests/` verify both halves on every page kind (index, epic, milestone, entity, kind-index, status). For the gap entry: assert presence + count + click-through. For the archive filter: assert sidebar chip strip presence; assert archived epics have `display: none` by default; assert they become visible under `#sidebar-all`.
- CI integration deferred per the epic Constraints; Playwright runs locally.

A render-against-real-fixture human-verification pass closes the milestone per CLAUDE.md *Render output must be human-verified before the iteration closes* — open multiple page kinds, verify the gap entry, click through; verify the sidebar chip strip toggles archived epics in/out.

## Constraints

- **Both halves share the same sidebar partial** (`_sidebar.tmpl`). Edits are coordinated; the sidebar's structural shape gains a new top-section entry (gaps) and a new chip strip (epic filter) but stays one template.
- **Sidebar chip strip uses `#sidebar-active` / `#sidebar-all` fragments** — deliberately different from M-0099's `#active` / `#all`. The two chip filters operate independently; a reader can toggle archive view on the kind-index page without affecting sidebar epic visibility.
- **Archived epics determined by path, not status.** Epics under `work/epics/archive/` are archived; epics outside that subtree are active. Aligns with ADR-0004's archive convention — status is decoupled from filesystem location; this milestone reads the filesystem indicator.
- **Active count only on the gap entry.** Matches M-0099's "default chip view" semantic. Total and archived breakdowns are visible via the home page's kind-index nav.
- **No JS.** Both halves use `:target`-driven CSS, same pattern as M-0099 and the milestone-page tabs.
- **No new entry per kind beyond gaps.** Decisions / ADRs / contracts stay reachable via the home page's "Browse by kind" block. Per the epic's *Out of scope*.

## Design notes

- The gap entry's position above the epic list (top section) matches the existing pattern: Project status and Overview sit in `.sidebar-top`. The new entry slots in after Overview.
- The sidebar chip strip's exact placement (between top section and epic list vs. inside a new sub-section heading) is a small visual choice to be made at red phase; the fragment naming and CSS shape are pinned above.
- The `SidebarEpic` struct gains an `Archived bool` field (or equivalent) so the template can emit `data-archived` per epic.
- The `SidebarData` struct gains `GapCount int` (or equivalent — final field naming decided at red phase).
- The cmd-side resolver and default resolver both update; existing tests for the sidebar should still pass once the new attribute is rendered consistently.

## Surfaces touched

- `internal/htmlrender/embedded/_sidebar.tmpl` (primary — gap entry in top section; chip strip near top; `data-archived` on each `<details class="sidebar-epic">`)
- `internal/htmlrender/embedded/style.css` (sidebar chip strip rules — re-use `.chip-strip` / `.chip` from M-0099; new `:target`-driven filter rule scoped to `.sidebar`)
- `internal/htmlrender/pagedata.go` (`SidebarData.GapCount`; `SidebarEpic.Archived`)
- `internal/htmlrender/default_resolver.go` (populate the new fields)
- `cmd/aiwf/render_resolver.go` (cmd-side resolver — same)
- `e2e/playwright/tests/` (primary test surface — extend `render.spec.ts` with sidebar gap-entry tests + sidebar chip filter tests)
- `internal/htmlrender/htmlrender_test.go` (sidebar emit-shape tests — complementary)
- `cmd/aiwf/render_archive_visibility_test.go` (sidebar archive state reflects path — complementary)

## Out of scope

- Same chip filter treatment for milestones in the sidebar — only epics get the filter. Milestones are scoped to their epic's `<details>` and inherit the parent's visibility.
- Sub-list of recent / open gaps inside the sidebar — just the gap entry + count, no enumeration.
- Per-kind sidebar entries for decisions / ADRs / contracts — defer until the gap entry pattern proves out.
- In-page status hierarchy in gaps.html — M-0101.
- Surfacing the gap count anywhere else (page header, status report) — sidebar only.
- Persistence of chip state across page navigations — fragment-only, no localStorage.

## Dependencies

- **M-0099** (kind-index chip filter) — depends_on. The sidebar gap entry's link target is the chip-filtered single `gaps.html`; the sidebar chip strip re-uses the `.chip-strip` / `.chip` CSS classes from M-0099.

## References

- E-0029 (parent epic)
- G-0114 (gap closed by this epic)
- M-0099 (chip-strip pattern this milestone re-uses)
- `internal/htmlrender/embedded/_sidebar.tmpl` — existing sidebar partial
- `internal/htmlrender/embedded/style.css` — chip-strip styling at the `Chip strip` section (added in M-0099)
- `CLAUDE.md` — *Substring assertions are not structural assertions*, *Render output must be human-verified before the iteration closes*

## Work log

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
