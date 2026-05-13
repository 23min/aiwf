# Epic wrap — E-0029

**Date:** 2026-05-12
**Closed by:** human/peter (via ai/claude under authorized scope opened at bd7e49b)
**Integration target:** main
**Epic branch:** epic/E-0029-glanceable-render
**Merge commit:** _(filled at step 5)_

## Milestones delivered

- M-0102 — Repair Playwright e2e suite for current kernel state (commit `18e5a25` promote done)
- M-0098 — Render-site layout overhaul: viewport-fill body, flush-left sidebar, prose cap (commit `9dc45a0` promote done)
- M-0099 — Kind-index chip filter: single emitted file per kind with :target chips (wrapped during epic, on epic branch)
- M-0100 — Sidebar adds gap entry + epic archive chip filter (wrapped during epic, on epic branch)

## Milestones deferred

- M-0101 — In-page status hierarchy in `gaps.html` (status: `cancelled` — deferred at user direction; the sort-vs-CSS-reorder-vs-grouped-sections mechanism choice deserves more design thought than fit this epic's remaining scope. Design notes preserved in the cancelled milestone body for a future iteration.)

## Summary

E-0029 closes [G-0114](../../gaps/archive/G-0114-html-render-gap-surface-status-and-archive-state-not-glanceable-from-sidebar.md) by making the rendered governance site usable for current-state synthesis at a glance. The body's `max-width: 78rem` cap is gone; the layout fills the viewport with modest 1rem padding and a wider 285px sidebar (M-0098). Per-kind index pages collapsed from active/all-pair to a single emitted file per kind, with a `:target`-driven `[Active] [All]` chip strip at the top — `gaps-all.html` and its cousins no longer exist; the chip handles the toggle client-side via CSS (M-0099). The sidebar gains a `Gaps (N)` entry showing the non-archived gap count plus its own `[Active] [All]` chip strip (with the distinct `#sidebar-all` fragment so the two filters operate independently) that hides archived epics by default — closing the user-reported "all 29 epics drown the in-flight ones" half of the glanceability failure (M-0100).

The epic also surfaced a separate concern that became its own prerequisite milestone: the Playwright e2e suite had silently rotted across three independent kernel changes (repo reorg `a137132`, ID width migration E-0023, and an `aiwf init` hook-installation behavior change) since the original E-0009 governance-render epic shipped. M-0102 repaired the suite end-to-end (path fix, hook strategy, canonical-ID assertions, `--tdd none` fixture verb update for G-0055's chokepoint) so all five milestones in E-0029 could rely on Playwright as the load-bearing test surface for layout / CSS / browser-state ACs. The suite's final state: 55 passing tests covering layout, chips, sidebar filtering, no-scroll-on-click behavior.

Two design discoveries shaped the implementation pattern across multiple ACs:
1. **Fragment-namespace discipline** — the sidebar chip filter and the kind-index chip filter use distinct fragments (`#all` vs `#sidebar-all`) so they toggle independently. This sets a precedent for future `:target`-driven UI: when two state machines coexist on one page, their fragments name their scope.
2. **`scroll-margin-top: 100vh` pins page at top on hash change** — applied first to `section[data-tab]` for tabs (M-0098/AC-5), then to `.chip` for chips (M-0100/AC-4). Same bug class, same fix; both surfaced from user visual review, not test scaffolding.

## ADRs ratified

_(none — by user direction at wrap. ADR candidates considered: fragment-namespace discipline for independent `:target` state machines, Playwright as chokepoint for browser-state ACs. Both stayed in the wrap.md narrative + commit messages rather than promote to standalone ADRs; the patterns are recoverable from the rendered output and the spec bodies.)_

## Decisions captured

_(none — by user direction at wrap. One candidate considered: `scroll-margin-top: 100vh` as the canonical "pin page at y=0 on hash change" recipe; chose to leave it as a code-comment-level note in `style.css` adjacent to both occurrences instead of promoting to D-NNN.)_

## Follow-ups carried forward

- G-0115 — `aiwf render roadmap --write` rewrites entity refs in epic prose to broken paths (filed mid-epic; blocks the roadmap-regen step at wrap)
- G-0116 — aiwfx-start-epic creates worktree before promote/authorize on trunk-based repos (filed mid-epic; rituals-plugin ordering observation)

## Doc findings

`wf-doc-lint` scoped to E-0029's change-set since branch base `bd7e49b`:

- **Broken code references:** none. No `docs/` references to the removed `gaps-all.html` / `decisions-all.html` / `adrs-all.html` / `contracts-all.html` filenames, the `.all-link` CSS class, or the "View all" cross-link text.
- **Removed-feature docs:** none.
- **Orphan files:** N/A — no `docs/` files in the change-set.
- **Documentation TODOs:** pre-existing TODOs in `docs/explorations/` and `docs/pocv3/migration/` predate E-0029; not introduced by this epic.

Clean for this epic's scope.

## Handoff

E-0029's chip-filter pattern is now established across two surfaces (kind-index pages, sidebar). Future iterations on the rendered site that introduce new `:target`-driven UI state can reuse `.chip-strip` / `.chip` markup and pick a distinct fragment name. The deferred M-0101 (in-page status hierarchy in `gaps.html`) is ready to pick up whenever the sort-vs-CSS-reorder-vs-grouped-sections design discussion lands; the milestone's body already enumerates the mechanism options with trade-offs.

The dead-code paths in `KindIndexData` (`IncludeArchived`, `ActiveFileName`, `AllFileName`) and the resolver method's `includeArchived bool` parameter survived M-0099's hybrid approach (M-0099/AC-1 dropped only the emit loop; AC-3 changed the call-site to always pass `true`). Cleanup is a follow-up — a small refactor PR or part of a future render-polish epic.
