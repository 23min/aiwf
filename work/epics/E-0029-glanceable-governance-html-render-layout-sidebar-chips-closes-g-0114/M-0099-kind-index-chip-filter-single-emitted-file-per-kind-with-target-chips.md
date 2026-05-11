---
id: M-0099
title: 'Kind-index chip filter: single emitted file per kind with :target chips'
status: draft
parent: E-0029
depends_on:
    - M-0098
tdd: required
---
# Kind-index chip filter: single emitted file per kind with :target chips

## Goal

Collapse the per-kind index pages from active/all-pair to a single emitted file per kind, with a `[Active] [All]` chip strip at the top of each page driven by `:target`. Default state shows non-archived rows; the `[All]` chip flips a CSS rule to reveal archived rows in the same view.

## Context

Today the renderer emits two files per kind that participates in archive segregation — `gaps.html` (active only) and `gaps-all.html` (all); same for `decisions`, `adrs`, `contracts`. The active/all distinction is invisible from page chrome (only the page title says which is which), and the all-set is reachable only via a small `all` sub-link inside the home page's "Browse by kind" block. G-0114 names this as a glanceability failure.

The chip filter design re-uses the rendered site's existing no-JS CSS state-machine pattern: the milestone-page tabs (Overview / Manifest / Build / Tests / Commits / Provenance) drive stateful UI via `:target + :has()` in `style.css`. The chip filter follows the same shape — `<a href="#all">` / `<a href="#active">` plus a `:target`-keyed CSS rule that hides archived rows in the default view and shows them under the `#all` fragment. URLs stay shareable (`gaps.html#all` is bookmarkable); the toggle is instant (no page reload); markup is a single emitted file per kind.

M-α (layout overhaul) lands first so the chip strip can be styled against the wider sidebar / fluid main panel; the chip strip's visual placement assumes the new layout.

## Acceptance criteria

ACs added via `aiwf add ac M-<id>` at start-milestone time. The observable-behavior space this milestone covers:

- The renderer emits **one file per kind** for the kind-index family: `gaps.html`, `decisions.html`, `adrs.html`, `contracts.html`. The `*-all.html` cousins are no longer emitted.
- The home page's "Browse by kind" block drops the `all` sub-link; one entry per kind in the nav list.
- Each kind-index page renders a chip strip at the top with two chips: `[Active]` (default, no fragment) and `[All]` (`<a href="#all">`). The chip strip uses a structural CSS class (`.chip-strip`) and the chips themselves are `.chip` anchors so other surfaces (sidebar in M-γ) can reuse the styling.
- A `:target`-keyed CSS rule hides archived rows by default and reveals them when the `#all` fragment is current. The chips' visual active state mirrors the same rule (active chip highlights when `:target` matches).
- Each table row carries `data-archived="true"` or `data-archived="false"` so the CSS rule can target archived rows specifically.
- The page emits the **full row set** (active + archived) regardless of the chip state — the chip filter is CSS-driven, not server-side, so the rendered markup is one source of truth.
- The chip strip renders unconditionally on every kind-index page, even for kinds with zero archived entries (consistent shape across pages).
- The `KindIndexData` struct loses `IncludeArchived`, `ActiveFileName`, `AllFileName`; the `KindIndexLink` struct loses `AllFileName`. The default resolver's `KindIndexData` method no longer takes an `includeArchived` boolean (single signature, single emit).
- All existing render tests pass after the migration; new **Playwright** tests in `e2e/playwright/tests/` exercise the chip filter end-to-end — clicking `[All]` flips the fragment to `#all`, archived rows become visible via computed style, the active-chip visual state mirrors the URL fragment. Parsed-HTML / parsed-CSS checks in Go remain for emit-shape assertions (chip-strip presence, `data-archived` attribute on rows, `:target` rule in `style.css`) but the `:target`-driven behavior is verified in a real browser, paralleling the existing tabs tests in `render.spec.ts`. CI integration is deferred per the epic Constraints; Playwright runs locally.

A render-against-real-fixture human-verification pass closes the milestone per CLAUDE.md *Render output must be human-verified before the iteration closes* — exercise each kind-index page, click the chip, verify URL fragment, verify archived rows appear/disappear.

## Constraints

- **No JavaScript.** Chip filter is `:target`-driven CSS only.
- **Single emitted file per kind.** No `*-all.html` cousins.
- **`data-archived` attribute on every row.** Required for the CSS rule to target archived rows; structural test assertion uses it.
- **URLs aren't a stable contract.** Existing pages or external links pointing at `gaps-all.html` (etc.) break with this milestone. The render output is regenerated on every `aiwf render` run; this is acceptable, but if narrative docs in this repo reference `*-all.html` filenames, fix them in the same milestone.
- **No status filter, no parent filter, no search.** The chip filter is binary (active / all). Other filter dimensions are out of scope for E-0029.

## Design notes

- The chip strip is positioned above the entity table inside `main`; not above the sidebar, not page-chrome above main. (The "promote chip strip to global page chrome" option was deliberately excluded from the epic.)
- The chip strip's reusable `.chip-strip` / `.chip` CSS classes will be picked up by M-γ for the sidebar gaps entry's visual treatment (optional — M-γ may use different styling).
- The `IncludeArchived` boolean removal simplifies the `kindPluralToKind` / `titleForKindIndex` / `buildKindIndexLinks` helpers in `default_resolver.go`. The `Title` field is now always the kind's plural name ("Gaps", "Decisions") — no "All gaps" variant.
- The `KindIndexLink.ArchivedCount` field stays useful: the home page's "Browse by kind" block can still show `(33 active, 79 archived)` next to each kind entry, so the reader knows what the chip's `[All]` view contains before clicking.

## Surfaces touched

- `internal/htmlrender/embedded/kind_index.tmpl` (primary — chip strip markup, `data-archived` on rows)
- `internal/htmlrender/embedded/style.css` (chip-strip rules, `:target` filter rule)
- `internal/htmlrender/embedded/index.tmpl` (drop the `all` sub-link)
- `internal/htmlrender/default_resolver.go` (collapse `KindIndexData` signature, drop the all-path emit)
- `internal/htmlrender/pagedata.go` (remove `IncludeArchived` / `ActiveFileName` / `AllFileName` from `KindIndexData`; remove `AllFileName` from `KindIndexLink`)
- `internal/htmlrender/htmlrender.go` (whichever caller dispatches the two-file emit per kind)
- `cmd/aiwf/render_resolver.go` (cmd-side resolver — same shape changes)
- `e2e/playwright/tests/` (primary test surface — extend `render.spec.ts` with chip-filter scenarios alongside the existing tabs tests)
- `cmd/aiwf/render_archive_visibility_test.go`, `cmd/aiwf/render_archive_test.go` (drop the `*-all.html` assertions, replace with chip / data-archived emit-shape assertions; complementary)
- `internal/htmlrender/htmlrender_test.go` (chip-strip markup structural test; complementary)

## Out of scope

- Sidebar gaps entry — M-γ.
- In-page status hierarchy in gaps.html — M-δ.
- Layout / CSS shape beyond chip-strip styling — depends on M-α already landed.
- Search, faceting, status filter, parent filter.
- Animation / transitions on chip toggle.

## Dependencies

- M-α (layout overhaul) — depends_on. The chip strip's visual styling assumes the wider sidebar and fluid main panel.

## References

- E-0029 (parent epic)
- G-0114 (gap closed)
- `internal/htmlrender/embedded/style.css` — existing `:target + :has()` tabs pattern at the *Tabs* section (line ~333)
- `CLAUDE.md` — *Substring assertions are not structural assertions*, *Render output must be human-verified before the iteration closes*

## Work log

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
