---
id: M-0100
title: Sidebar adds gap entry + epic archive chip filter
status: in_progress
parent: E-0029
depends_on:
    - M-0099
tdd: required
acs:
    - id: AC-1
      title: Sidebar shows Gaps (N) entry with active count
      status: met
      tdd_phase: done
    - id: AC-2
      title: Chip strip with Active/All renders in sidebar
      status: open
      tdd_phase: green
    - id: AC-3
      title: Sidebar archive chip filter toggles epic visibility
      status: open
      tdd_phase: red
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

### AC-1 — Sidebar shows Gaps (N) entry with active count

**Pass criterion**: Every rendered page's sidebar (`<aside class="sidebar">`) includes a "Gaps (N)" link in its `.sidebar-top` section, where N is the count of non-archived gaps in the planning tree. Verified via Playwright on multiple page kinds (index.html, an epic page, a milestone page, an entity page, a kind-index page) — `aside.sidebar .sidebar-top a` matching text `/Gaps \(\d+\)/` exists. The count value matches the fixture tree's count of files under `work/gaps/` (not `work/gaps/archive/`). Clicking the link navigates to `gaps.html` (verified via `page.url()` after `.click()`).

**Edge cases**: A planning tree with zero non-archived gaps renders the entry as "Gaps (0)" — the entry is not suppressed. The count is recomputed on every render; no caching. When the current page is `gaps.html`, the entry carries `aria-current="page"` and renders with the active-link styling (`.sidebar a[aria-current="page"]` rule already in `style.css`). The entry sits below "Overview" in `.sidebar-top` and above the epic `<details>` list — visual position matches the existing top-section pattern.

**Code references**: `internal/htmlrender/embedded/_sidebar.tmpl` — new `<li>` in `.sidebar-top` between "Overview" and the epic loop; uses the existing aria-current pattern. `internal/htmlrender/pagedata.go` — `SidebarData` gains a `GapCount int` field. `internal/htmlrender/default_resolver.go` — `sidebar()` helper populates `GapCount` by counting `r.tree.ByKind(entity.KindGap)` entries whose path doesn't include `/archive/`. `cmd/aiwf/render_resolver.go` — cmd-side resolver mirrors. Test in `e2e/playwright/tests/render.spec.ts` under a new `sidebar — gap entry (M-0100/AC-1)` describe.

### AC-2 — Chip strip with Active/All renders in sidebar

**Pass criterion**: Every rendered page's sidebar contains a `<nav class="chip-strip">` with two chips: Active and All. Markup mirrors M-0099's kind-index chip strip: each chip is an `<a class="chip">` with matching `id` and `href` so `:target` CSS drives both the active-chip visual state and AC-3's epic filter. The chip strip uses the **`#sidebar-active`** and **`#sidebar-all`** fragments — distinct from M-0099's `#active`/`#all` so the sidebar archive filter and the kind-index page filter toggle independently. Asserted via Playwright: `aside.sidebar nav.chip-strip` exists; contains exactly two `a.chip` children; first has text "Active", id "sidebar-active", href "#sidebar-active"; second has text "All", id "sidebar-all", href "#sidebar-all".

**Edge cases**: The chip strip renders unconditionally — even in a tree with zero archived epics the strip appears. Position: between the top section (`Project status` / `Overview` / `Gaps (N)`) and the epic `<details>` list. The strip uses M-0099's existing `.chip-strip` and `.chip` CSS classes for visual styling — no new styling rules in this AC; the rules from M-0099 already handle pill shape, hover state, :target highlight, and default Active highlight via `:not(:has(.chip:target))`. Note: M-0099's default-highlight CSS uses `#active` — for the sidebar chip the rule's selector needs broadening (or a parallel rule for `#sidebar-active`) so the sidebar's Active chip highlights too. That's a small CSS adjustment landing in this AC.

**Code references**: `internal/htmlrender/embedded/_sidebar.tmpl` — chip strip markup added between `.sidebar-top` and the `{{range .Epics}}` loop. `internal/htmlrender/embedded/style.css` — broaden M-0099's default-highlight rule to cover both `#active` and `#sidebar-active`, or add a parallel rule. Test in `e2e/playwright/tests/render.spec.ts` under a new `sidebar — chip strip markup (M-0100/AC-2)` describe.

### AC-3 — Sidebar archive chip filter toggles epic visibility

**Pass criterion**: On any rendered page with no URL fragment, sidebar epics whose paths are under `work/epics/archive/` have `display: none` (verified via Playwright `getComputedStyle(epicElement).display`); non-archived epics are visible. Loading the same page with `#sidebar-all` reveals all sidebar epics (including archived) — no `display: none`. Every `<details class="sidebar-epic">` element carries `data-archived="true"` or `data-archived="false"` so the CSS filter can target archived epics specifically.

**Edge cases**: The active-vs-archived determination is path-based, not status-based — epics under `work/epics/archive/` are archived; epics outside that subtree are active (regardless of frontmatter status). Aligns with ADR-0004's archive convention. The CSS rule keys off `body:has(#sidebar-all:target)` to be specific — does not fire when the kind-index page's `#all` is targeted, so the two chip filters remain independent. Milestones nested inside archived epics ride with their parent's visibility (the `<details>` collapses; nested `<ul>` follows DOM hierarchy). When the current page is itself inside an archived epic, the epic's `<details>` would normally have `open` (per existing `IsActive` logic), but the filter rule hides the whole `<details>` regardless — the user has to switch to `#sidebar-all` to see the current page's parent epic in the sidebar.

**Code references**: `internal/htmlrender/embedded/_sidebar.tmpl` — each `<details class="sidebar-epic">` gains `data-archived="{{if .Archived}}true{{else}}false{{end}}"`. `internal/htmlrender/embedded/style.css` — new CSS rule `aside.sidebar .sidebar-epic[data-archived="true"] { display: none; }` and `body:has(#sidebar-all:target) aside.sidebar .sidebar-epic[data-archived="true"] { display: block; }` (or equivalent display value for `<details>`). `internal/htmlrender/pagedata.go` — `SidebarEpic` gains `Archived bool` field. `internal/htmlrender/default_resolver.go` and `cmd/aiwf/render_resolver.go` — populate `Archived` from `entity.IsArchivedPath(e.Path)`. Test in `e2e/playwright/tests/render.spec.ts` under a new `sidebar — archive chip filter (M-0100/AC-3)` describe; fixture needs at least one archived epic (the existing renderRichFixture has none — enrich similarly to M-0099/AC-3's gap-archive setup).

