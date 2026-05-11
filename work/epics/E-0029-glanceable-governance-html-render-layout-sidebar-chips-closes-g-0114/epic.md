---
id: E-0029
title: 'Glanceable governance HTML render: layout, sidebar, chips (closes G-0114)'
status: proposed
---
# E-0029 — Glanceable governance HTML render: layout, sidebar, chips

## Goal

Make the rendered governance site usable for current-state synthesis at a glance. The layout fills the viewport with the sidebar flush-left; the sidebar surfaces gaps with the active count; per-kind index pages collapse from active/all-pair to a single file with `:target`-driven filter chips at the top; within `gaps.html` the open subset pops visually rather than sitting equally-weighted with the addressed rows.

## Context

E-0009 (Iteration I3 — Governance HTML render, done, archived) shipped the static-site governance view: per-kind templates, dark mode, sidebar navigation, Playwright browser tests. The shape has held up — what's left is the quality-of-life polish a maintainer running the site daily picks up on.

G-0114 names three concrete friction points the maintainer hit while using the site for "what's open right now?":
- The sidebar shown on every page omits gaps entirely — readers have to scroll to the small "Browse by kind" block at the bottom of `index.html`, then pick between `gaps.html` and `gaps-all.html` via a tiny `all` sub-link.
- The active-vs-archived distinction is invisible from page chrome — landing on `gaps-all.html` (112 rows) accidentally looks like the same surface as `gaps.html` (35 rows) but isn't.
- Within `gaps.html` itself, open and addressed rows are equally weighted in a flat table; the per-row status badges exist but don't function as a glanceable organizer.

The layout half of this epic is the maintainer's own request and an observation while sitting with the rendered site: the body's `max-width: 78rem` cap centers everything in a fixed-width column, which wastes screen real estate on wide monitors and makes the 220px sidebar look cramped. The sidebar wants to be wider to host the new gaps block (with active count) comfortably, and flush-left feels right for a navigation surface. The main panel should fill the rest of the viewport — fluid — but prose blocks need to retain a readable measure (long lines on a 27" monitor are uncomfortable).

The chip filter design re-uses the rendered site's existing no-JS CSS state-machine pattern. The milestone-page tabs (Overview / Manifest / Build / Tests / Commits / Provenance) already drive stateful UI via `:target + :has()` in `style.css`. Filter chips fit the same shape: `<a href="#all">` / `<a href="#active">` plus a `:target`-keyed CSS rule that hides archived rows when the active chip is current. No JS, URLs stay shareable (`gaps.html#all` bookmarkable), single emitted file per kind.

Closes G-0114.

## Scope

### In scope

- **Layout overhaul.** The body's `max-width: 78rem` / `margin: 2rem auto` cap is removed; the layout grid fills the viewport. The sidebar widens to a ~260–300px target and sits flush against the left edge. The main panel runs fluid for tables/code/wide content; prose blocks (epic/milestone Overview sections, entity body sections, AC card prose) cap at a readable measure (~72ch / 50rem). No new media-query breakpoints — the existing <768px sidebar-collapse stays.
- **Sidebar surfaces gaps.** A new sidebar entry "Gaps (N)" — where N is the active count — appears alongside the existing Status / Overview / Epics entries. Clicking lands on the per-kind index page's default (active) view. The entry appears on every rendered page; no per-page conditional.
- **Kind-index chip filter.** Each kind's index page (`gaps.html`, `decisions.html`, `adrs.html`, `contracts.html`) collapses from the current active/all-pair to a single emitted file. Atop the page sits a chip strip `[Active] [All]` driven by `:target`; default state = active (no fragment) shows only non-archived rows; the `[All]` chip is a `<a href="#all">` that flips a CSS rule to reveal archived rows. The chip strip is the same component across all four kinds. Each row carries `data-archived="true|false"`. The home page's "Browse by kind" block loses its `all` sub-link (one entry per kind suffices).
- **In-page status hierarchy in `gaps.html`.** Open and addressed rows are organized so a reader skimming sees what's in flight without scanning the full list. Mechanism left to milestone-level design (grouping by status, ordering, or per-status visual weight) — the success bar is "open subset pops visually."
- **Renderer data-shape simplification.** `KindIndexData.IncludeArchived` / `.ActiveFileName` / `.AllFileName` collapse to a single emit shape; `KindIndexLink.AllFileName` and its sub-link in the home-page nav drop; `default_resolver.KindIndexData` no longer dispatches on `includeArchived` boolean.

### Out of scope

- **G-0113 (rendered HTML site has no publish path).** Separate hosting concern, separate epic.
- **Same sidebar treatment for decisions / ADRs / contracts.** The chip filter is shared across kind-index pages; the sidebar surfaces only gaps in this epic. Extension to other kinds defers until the gaps pattern proves out in use.
- **New media-query breakpoints / drawer sidebar / hamburger.** The existing <768px collapse stays; no tablet-range intermediate breakpoint.
- **Search / faceting / additional filter dimensions.** The chip filter is binary (active / all). No status filter, no parent filter, no full-text search on kind-index pages.
- **Promoting the chip strip to a global page-chrome element above the sidebar.** Chips stay on the kind-index pages where they belong.

## Constraints

- **No JavaScript.** The rendered site is server-emitted static HTML + CSS. The chip filter uses `:target` and CSS rules, same pattern as the existing milestone-page tabs. No `<script>` tags introduced.
- **No new external dependencies.** CSS stays hand-written; no framework. The layout's `clamp()` / `max-content` / `:target` / `:has()` features used today are already in every supported browser (Chrome 111+, Safari 16.2+, Firefox 113+ per the existing `style.css` comment).
- **Prose-cap is internal to `main`, not the body.** The body fills the viewport; prose readability comes from a CSS rule that caps the *width of prose blocks inside main*, not the width of `main` itself. Tables, code, and wide entity views still get the full panel width.
- **Single canonical emitted file per kind.** The `-all.html` filenames go away as part of the chip migration. Existing URLs that pointed at `gaps-all.html` break; the kernel's render output is not a stable URL contract (the rendered site is regenerated on every `aiwf render` run), so this is acceptable. The change is mechanical and one-shot, not a deprecation window.
- **Sidebar gaps entry shows active count, not total count.** "Gaps (33)" means 33 non-archived gaps — matches what the chip's default view will show. The all-count (112) is only visible after the user toggles the `[All]` chip.
- **In-page status hierarchy preserves all rows visible by default.** No collapse of addressed rows behind a toggle; the goal is "open pops," not "addressed hides." A reader scanning `gaps.html` should still see addressed rows in their peripheral vision.

## Success criteria

- A reader landing on any rendered page sees the sidebar flush against the left edge of the viewport, with the main panel filling the rest at any viewport width above the existing <768px collapse threshold.
- Prose body text on epic, milestone, and entity pages stays comfortably readable on a 27" monitor — line length doesn't exceed ~72ch even when the main panel is wider.
- A reader on any rendered page can reach the gaps index from the sidebar; the sidebar entry shows the active count.
- A reader on `gaps.html` (and every other kind-index page) sees a chip strip at the top indicating which view they're on; flipping the chip switches the visible row set without a page reload.
- A reader skimming `gaps.html` picks up the open rows without scanning the addressed rows first.
- The renderer emits one file per kind for the kind-index family; `gaps-all.html` and its cousins no longer exist.
- Every existing render test (`cmd/aiwf/render_*_test.go`, `internal/htmlrender/htmlrender_test.go`) passes; new tests cover the chip-filter markup, the sidebar gaps entry's count, the row `data-archived` attribute, and the layout's flush-left + prose-cap CSS rules through structural assertions (per the *Substring assertions are not structural assertions* rule in `CLAUDE.md`).

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Exact sidebar width target — 260px vs 280px vs 300px vs `clamp(240px, 18vw, 320px)` | no | Decide during the layout milestone with a real-content visual check; document the chosen value in the milestone's wrap spec. |
| In-page status hierarchy mechanism — grouped sections, sort order, or per-row visual weight (e.g. fading addressed rows) | no | Decide during the gaps-page milestone; sketch 2–3 options in the milestone spec, pick one. |
| Should the chip strip appear when a kind has zero archived entries (e.g. `decisions.html` early in life)? | no | Render the chip strip unconditionally for shape consistency, OR suppress when archived count is zero. Decide during the chip milestone; lean toward unconditional for consistency. |
| Backwards-compat: do consumers have external pages linking to `gaps-all.html`? | no | The rendered site is regenerated; no external link contract. Confirm during wrap that the project's own narrative docs don't carry `*-all.html` references; fix if any. |

## Milestones

<!-- Bulleted list, ordered by execution sequence. Status is NOT carried here. -->

<!-- To be filled by `aiwfx-plan-milestones`. Sketch:
- M-NNNN — Layout overhaul: body fills viewport, sidebar wider + flush-left, main panel fluid with prose cap
- M-NNNN — Kind-index chip filter: collapse to single emitted file per kind, `:target`-driven [Active]/[All] chips
- M-NNNN — Sidebar surfaces gaps with active count + in-page status hierarchy in gaps.html
-->

## References

- G-0114 — `work/gaps/G-0114-html-render-gap-surface-status-and-archive-state-not-glanceable-from-sidebar.md` (the gap this epic closes)
- E-0009 — `work/epics/archive/E-0009-iteration-i3-governance-html-render/epic.md` (the original HTML render epic)
- `internal/htmlrender/embedded/_sidebar.tmpl` — current sidebar partial
- `internal/htmlrender/embedded/kind_index.tmpl` — current kind-index template
- `internal/htmlrender/embedded/style.css` — render stylesheet; `:target + :has()` tabs pattern at line ~333
- `internal/htmlrender/default_resolver.go` + `internal/htmlrender/pagedata.go` — render data shape; `KindIndexData` / `KindIndexLink`
- `CLAUDE.md` — *Substring assertions are not structural assertions*, *Render output must be human-verified before the iteration closes*, *Test untested code paths before declaring code paths "done"* (apply directly to this epic's tests)
