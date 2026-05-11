---
id: M-0098
title: 'Render-site layout overhaul: viewport-fill body, flush-left sidebar, prose cap'
status: in_progress
parent: E-0029
depends_on:
    - M-0102
tdd: required
acs:
    - id: AC-1
      title: Layout fills viewport at widths above 768px
      status: met
      tdd_phase: done
    - id: AC-2
      title: Sidebar width resolves to chosen target value
      status: met
      tdd_phase: done
    - id: AC-3
      title: Prose blocks cap at readable measure inside main
      status: open
      tdd_phase: red
    - id: AC-4
      title: Mobile collapse stacks sidebar below main below 768px
      status: open
      tdd_phase: red
    - id: AC-5
      title: Tab clicks do not scroll the page
      status: open
      tdd_phase: red
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

- **M-0102** (Repair Playwright e2e suite for current kernel state) — added mid-flight after AC-1's red phase surfaced that the Playwright suite had rotted across multiple kernel changes since E-0009 (repo reorg, ID width migration, `aiwf init` hook-write behavior). M-0098 cannot have its layout / CSS / viewport ACs tested via Playwright until M-0102 lands the suite green.

## References

- E-0029 (parent epic)
- G-0114 (gap closed by this epic)
- E-0009 (archived; original HTML render epic)
- `CLAUDE.md` — *Substring assertions are not structural assertions*, *Render output must be human-verified before the iteration closes*, *Test untested code paths before declaring code paths "done"*

## Work log

### AC-1 — Layout fills viewport at widths above 768px (red phase paused)

Red-phase test-authoring began against the existing Playwright suite; running `npx playwright test` against the unmodified suite surfaced rot from three independent kernel changes (path-rot from `a137132` reorg; `aiwf init`'s hook-installation behavior change; canonical 4-digit ID width migration from E-0023). M-0098/AC-1 cannot fail "for the right reason" against the current fixture. M-0102 allocated as a prerequisite milestone to repair the suite; M-0098 stays at `in_progress` but is blocked until M-0102 reaches `done`. Resume here after M-0102 wraps.

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)

### AC-1 — Layout fills viewport at widths above 768px

**Pass criterion**: At any viewport width above 768px, the layout fills the viewport horizontally with modest uniform edge padding (≤ 2rem each side, no centering gutter). The body element has no `max-width` cap (`getComputedStyle(document.body).maxWidth` is `"none"`). The sidebar's left edge sits within 32px of the viewport's left edge (`boundingBox.x <= 32`). The main panel's right edge sits within 32px of the viewport's right edge (`viewport.width - mainRight <= 32`). No horizontal scrollbar appears (`document.documentElement.scrollWidth <= window.innerWidth`). Verified via Playwright `boundingBox()` / `getComputedStyle()` queries at 1920×1080 viewport (the cap-visible failure-mode viewport).

**Edge cases**: A 2560px viewport must show the layout truly fluid, not capped at 78rem (the current default). The "modest edge padding" is the body's own `padding` (currently 1rem all around) — that's the difference between strict viewport-fill and the implemented behavior. The 32px threshold in the assertions accommodates 1rem (16px) of body padding plus sub-pixel rounding; if the padding grows beyond 1.5rem the threshold should widen accordingly. The `.layout` grid's gap (currently `1.5rem`) and any inner padding on main are preserved — the test asserts viewport-edge proximity, not zero spacing inside the panel.

**Code references**: Primary change in `internal/htmlrender/embedded/style.css` — body's `margin: 2rem auto; max-width: 78rem` and `.layout`'s grid shape need to flip to viewport-spanning. Test lives in `e2e/playwright/tests/render.spec.ts` under a new `test.describe("layout — viewport-fill", ...)` block adjacent to the existing `index.html` / tabs describes. Render fixture: existing `renderRichFixture()` in `e2e/playwright/fixture.ts` (no fixture changes expected for AC-1).

### AC-2 — Sidebar width resolves to chosen target value

**Pass criterion**: The sidebar's computed width is the chosen target value across viewport widths. The target is a single source of truth defined in CSS (e.g. a custom property `--sidebar-width: clamp(240px, 18vw, 320px)` or a fixed `280px`); the test asserts `getComputedStyle(sidebar).width` resolves correctly at 800, 1280, 1920, and 2560 viewport widths. If `clamp()` is used, the test verifies the resolved value sits between the clamp's min and max at each viewport (240px floor, 320px ceiling for the proposed `clamp(240px, 18vw, 320px)`). The chosen value is recorded in this milestone's wrap *Validation* section so M-0099/M-0100/M-0101 reference one source of truth.

**Edge cases**: The sidebar's content (epic titles, brand mark + wordmark, "Project status" / "Overview" links, the planned "Gaps (N)" entry from M-0100) must not overflow the chosen width — verify the longest epic title in the rich fixture fits without `text-overflow: ellipsis` triggering. The chosen value must also not be so wide that main becomes uncomfortably narrow on common laptop viewports (1280–1440px) — sketched options and the eyeball pass record the choice. If `clamp()` is chosen, verify it degrades sensibly below the floor (mobile collapse takes over at <768px per AC-4).

**Code references**: Likely `internal/htmlrender/embedded/style.css` — either a `--sidebar-width` custom property in `:root` consumed by `.layout`'s `grid-template-columns`, or a direct value in the grid declaration. Decision sketch (2–3 options) lives in this milestone's *Design notes* section, filled at red-phase start. Test in `e2e/playwright/tests/render.spec.ts` under "layout — sidebar width". Final value lands in *Validation* at wrap; downstream milestones reference it.

### AC-3 — Prose blocks cap at readable measure inside main

**Pass criterion**: At a viewport wide enough that `main` exceeds ~50rem (800px at default 16px font; verified at 1920×1080), prose blocks inside `main` are capped at the readable measure — a paragraph rendered on an epic page, milestone Overview tab, entity body section, or AC card description has `boundingClientRect.width <= 50rem`. Wide content inside the same `main` panel is unaffected — a `<table>` on a milestone Manifest tab, a `<pre>` code block, an AC card container (the `.ac` div), or the dependency DAG renders at the full main-panel width (`boundingClientRect.width > 50rem`). The cap is implemented as a CSS rule on prose-typed elements (likely a `.body-section`, `.ac > .ac-desc`, or `main p` selector — final selector decided in red phase), not by capping `main` itself.

**Edge cases**: Prose-cap must apply uniformly across page kinds — verify on epic / milestone / entity (gap/ADR/decision/contract) pages, not only one. Short prose blocks (one-sentence paragraphs) render at their natural width — the cap is a `max-width`, not a fixed width. Code blocks and `<pre>` tags inside prose paragraphs (`<p><code>...</code></p>`) follow the paragraph's cap, not their own scope. Tables nested inside body sections (rare but possible) fill the main width, not the prose cap. The cap respects rem units (so user font scaling extends the cap proportionally) rather than pixels.

**Code references**: New CSS rule in `internal/htmlrender/embedded/style.css` keyed off a prose-typed selector or class. Templates under `internal/htmlrender/embedded/*.tmpl` may need a small structural wrap (e.g. `<div class="body-section">` around the body markdown emit) if the existing markup doesn't already isolate prose. Test in `e2e/playwright/tests/render.spec.ts` under "layout — prose cap" — assert prose width on at least one page per kind, and assert table/code/`.ac` width exceeds the cap on at least one milestone page.

### AC-4 — Mobile collapse stacks sidebar below main below 768px

**Pass criterion**: At viewport widths below 768px (verified at 375 — iPhone SE width — 414, 600, and 767), the sidebar renders below `main` rather than beside it. Asserted via Playwright: `sidebar.boundingBox().y > main.boundingBox().y + main.boundingBox().height` (sidebar's top edge is below main's bottom edge), or — depending on the layout collapse mechanism — `sidebar.boundingBox().x === 0 && main.boundingBox().x === 0 && sidebar.boundingBox().y > main.boundingBox().y` (single-column stack). No horizontal scrollbar appears (`document.documentElement.scrollWidth <= window.innerWidth`). The existing `@media (max-width: 768px)` rule in `style.css` still fires and still flips `.layout`'s `grid-template-columns` to `1fr`; the sidebar's `position: static !important` override holds against the new sticky/flush-left layout.

**Edge cases**: At exactly 768px the rule fires (per the existing `max-width: 768px` definition); just above (e.g. 769px) it does not — verify both sides of the boundary so future tweaks to the breakpoint don't silently regress. Sidebar links still navigate when clicked in the collapsed layout (no overlap with main blocking the click target). The main panel's prose cap from AC-3 still applies in the collapsed view but never exceeds viewport width — main fills the available column. The brand mark + wordmark in the sidebar header don't overflow at 375px width.

**Code references**: Existing `@media (max-width: 768px)` block in `internal/htmlrender/embedded/style.css` (around line 101 today) — verify intact post-overhaul; may need a one-line tweak if the new layout's sticky/flush-left rules need explicit reset at the breakpoint. Test in `e2e/playwright/tests/render.spec.ts` under "layout — mobile collapse" — use `page.setViewportSize({ width: 600, height: 800 })` (and the other widths) to drive the assertion.

### AC-5 — Tab clicks do not scroll the page

**Pass criterion**: Clicking a tab link in a milestone page's `nav.tabs` (Overview, Manifest, Build, Tests, Commits, Provenance) does not scroll the page. After the click, `window.scrollY === 0` — the page is pinned at the top, regardless of which tab was clicked. Verified via Playwright by loading a milestone page, asserting `scrollY === 0` initially, clicking each tab in turn, and asserting `scrollY === 0` remains true after each click.

**Edge cases**: The behavior must hold at multiple viewport heights (tested at 1080 and 720 — both common laptop heights). The pin-to-top semantics apply even when a section is taller than the viewport (e.g. the Manifest tab with many ACs) — the scroll position stays at 0; the user can manually scroll within the section. The `:target`-driven CSS show/hide of sections remains intact (this AC is about scroll position, not visibility). Tab navigation via direct URL load (e.g. `M-0001.html#tab-build`) is also expected to land at scroll y=0, not scrolled-into-view of `#tab-build`.

**Code references**: The fix lives in `internal/htmlrender/embedded/style.css` — add `scroll-margin-top: 100vh` (or equivalent large value) to the `section[data-tab]` selector. The browser's "scroll the target into view" behavior on hash-change respects `scroll-margin-top`, treating it as a phantom top-margin; a value larger than the document height effectively means "the target is so far above the top that scrolling to it clamps at y=0." The existing `:target + :has()` show/hide rule at `style.css:359` is unchanged. Tests in `e2e/playwright/tests/render.spec.ts` extend the existing `milestone page — :target tab show/hide` describe or add a sibling layout describe.

**Why this surfaced now**: AC-1's body-padding refinement removed the original `margin: 2rem auto` from `body`, which had provided a 2rem buffer at the top of the page. The scroll-into-view jump on tab clicks was always present but visually buffered; without the top margin, the jump is more pronounced and the bug became user-visible during AC-1/AC-2 review.

