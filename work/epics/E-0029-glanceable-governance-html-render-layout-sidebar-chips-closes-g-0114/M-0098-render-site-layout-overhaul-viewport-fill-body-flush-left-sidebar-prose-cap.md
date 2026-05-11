---
id: M-0098
title: 'Render-site layout overhaul: viewport-fill body, flush-left sidebar, prose cap'
status: draft
parent: E-0029
tdd: required
---
# Render-site layout overhaul: viewport-fill body, flush-left sidebar, prose cap

## Goal

Make the rendered governance site fill the viewport at any width. The body's max-width cap is removed; the sidebar widens and aligns flush against the viewport's left edge; the main panel is fluid; prose blocks inside `main` retain a readable measure via an inner cap rather than a global one.

## Context

E-0009 (Iteration I3 — Governance HTML render) shipped the static-site view with `body { margin: 2rem auto; max-width: 78rem }` and `.layout { grid-template-columns: 220px 1fr }`. The fixed body cap wastes screen real estate on wide displays (the layout centers in a ~1248px column regardless of viewport width). The 220px sidebar is cramped for the new gaps-block addition planned for M-γ (active count + future kind links). The layout overhaul lands first so M-β / M-γ / M-δ build on the new shape — chip strips, sidebar entries, and in-page status hierarchy all look better against a wider sidebar and fluid main panel than against the current centered column.

The prose-cap is intentionally an inner constraint (CSS rule on prose-typed blocks within `main`) rather than a cap on `main` itself, so tables, code blocks, AC cards, and milestone-tab content still fill the available horizontal space while body text on epic / milestone / entity pages stays readable.

## Acceptance criteria

ACs added via `aiwf add ac M-<id>` at start-milestone time. The observable-behavior space this milestone covers:

- The body's `max-width: 78rem` cap is removed; the layout grid spans the full viewport width.
- The sidebar sits flush against the viewport's left edge — no margin between the viewport edge and the sidebar's left border.
- The sidebar widens to a target around 280px (final value decided during implementation against real content and recorded in this milestone's wrap *Validation* section).
- The main panel is fluid — it occupies the remaining horizontal space at any viewport width above the existing <768px collapse threshold.
- Prose blocks inside `main` cap at ~72ch (~50rem) for readability. Applies to: epic Overview prose, milestone tab body prose, entity body sections on gap / ADR / decision / contract pages, AC card description text. Implemented as a CSS rule on prose-typed blocks (`.body-section`, `.ac > .ac-desc`, or equivalent), not by capping `main`.
- Tables, code blocks, AC cards, and milestone-tab content (Manifest table, Tests table, Provenance scope table, dependency DAG, commits list) fill the full main-panel width — the prose-cap rule does not affect them.
- The existing <768px mobile collapse continues to work: the sidebar drops below main; no horizontal scroll appears on phone-width viewports; no broken layout at common phone widths (375, 414, 768).

Each AC is asserted via **Playwright browser tests** in `e2e/playwright/tests/` (extending the existing `render.spec.ts` or in a sibling spec added for this milestone). Computed-style verification — `getComputedStyle`, `getBoundingClientRect`, viewport-resize behavior — is the load-bearing check; parsed-CSS / parsed-HTML checks in Go remain useful for structural shape but cannot reliably assert `clamp()`-resolved widths, viewport-dependent layout, or the `@media (max-width: 768px)` collapse. CI integration for the Playwright suite is **deferred for E-0029** per the epic Constraints; the run is local until the follow-up wires it. A render-against-real-fixture human-verification pass closes the milestone per CLAUDE.md *Render output must be human-verified before the iteration closes*; the chosen sidebar width value is recorded in *Validation* at wrap.

## Constraints

- **No JavaScript introduced.** All layout changes are CSS-only.
- **No new media-query breakpoints.** The existing <768px sidebar-collapse stays as the single responsive break.
- **Prose-cap is internal to main, not the body.** The body fills the viewport. Readability comes from a CSS rule that caps the width of prose-typed blocks inside `main`. Tables and wide content still get the full panel width.
- **Sidebar width is documented at wrap.** The final pixel value (or `clamp()` expression) chosen during implementation lands in the milestone's *Validation* section so future milestones reference one source of truth.

## Design notes

- The chip filter (M-β), gaps-block (M-γ), and in-page status hierarchy (M-δ) all build on this milestone's layout. Don't pre-emptively wire any of those here — keep the change scoped to layout / CSS shape.
- `style.css` already uses CSS custom properties for color tokens and `color-mix()` for derived shades; follow the same pattern for any new tokens this milestone introduces (sidebar width, prose-cap width).
- The existing `.layout` grid uses `grid-template-columns: 220px 1fr`. The new shape is `<sidebar-width> 1fr`; the 1fr column hosts main, prose-cap applies inside main.

## Surfaces touched

- `internal/htmlrender/embedded/style.css` (primary — body, .layout, .sidebar, prose-cap rule)
- `internal/htmlrender/embedded/_sidebar.tmpl`, `internal/htmlrender/embedded/index.tmpl`, `internal/htmlrender/embedded/epic.tmpl`, `internal/htmlrender/embedded/milestone.tmpl`, `internal/htmlrender/embedded/entity.tmpl`, `internal/htmlrender/embedded/kind_index.tmpl`, `internal/htmlrender/embedded/status.tmpl` (minimal — possibly wrap prose blocks in a class for the prose-cap selector)
- `e2e/playwright/tests/` (primary test surface — extend `render.spec.ts` or add a milestone-scoped spec; load-bearing computed-style assertions live here)
- `cmd/aiwf/render_*_test.go` and `internal/htmlrender/htmlrender_test.go` (complementary — emit-shape and CSS-rule-presence checks; not load-bearing for layout ACs)

## Out of scope

- Sidebar gaps entry — lands in M-γ.
- Kind-index chip filter — lands in M-β.
- In-page status hierarchy in gaps.html — lands in M-δ.
- New media-query breakpoints, drawer-style sidebar, hamburger menu.
- Color / typography changes beyond what the layout overhaul implies.

## Dependencies

- None. M-α is the foundation milestone for E-0029.

## References

- E-0029 (parent epic)
- G-0114 (gap closed by this epic)
- E-0009 (archived; original HTML render epic)
- `CLAUDE.md` — *Substring assertions are not structural assertions*, *Render output must be human-verified before the iteration closes*, *Test untested code paths before declaring code paths "done"*

## Work log

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
