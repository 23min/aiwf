---
id: M-0101
title: In-page status hierarchy in gaps.html
status: draft
parent: E-0029
depends_on:
    - M-0099
tdd: required
---
# In-page status hierarchy in gaps.html

## Goal

Within `gaps.html`, organize rows so a reader skimming sees the open subset at a glance without scanning the addressed rows first. The mechanism — grouped sections, ordering, or per-row visual weight — is decided during implementation; the success bar is "open subset pops visually" while all rows remain visible by default (no collapse).

## Context

Today `gaps.html` (post-M-β: the single chip-filtered file) renders rows as a flat table sorted by id. Per-row status badges (`open`, `addressed`) exist in the markup but don't function as a glanceable organizer — open and addressed rows sit equally weighted, so a reader looking for "what's actually in flight right now" has to read row-by-row. With 33 open and 2 addressed gaps in the active subset today, this matters less in absolute terms, but the addressed count will grow over time and the glanceability cost compounds.

G-0114 names this as the third sub-problem. M-δ is independent of M-γ (sidebar gaps entry) — they touch different surfaces and have different test sets — but both depend on M-β so the rows ship with the right attributes (`data-archived` from M-β; `data-status` likely added here).

The mechanism choice is deliberately deferred to start-milestone time so the implementer can sketch 2–3 options against real content and pick the one that reads cleanest. Constraint: all rows visible by default (no collapse behind a toggle).

## Acceptance criteria

ACs added via `aiwf add ac M-<id>` at start-milestone time. The observable-behavior space this milestone covers:

- A reader skimming `gaps.html` picks up open rows before addressed rows — open rows have visual prominence (color, weight, ordering, grouping, or section heading) over addressed rows.
- All rows remain visible by default; no collapse, no hide-behind-toggle, no chip filter on status (status filter is out of scope per the epic).
- Each row carries a `data-status` attribute (or equivalent structural marker) so the CSS rule providing the hierarchy can target rows by status, and tests can assert hierarchy structurally.
- The treatment applies specifically to the gaps kind-index page. Other kind-index pages (decisions, ADRs, contracts) are unchanged in this milestone — the gap surface is the high-value current-state target per G-0114; extension to other kinds is deferred.
- A render-against-real-fixture human-verification pass closes the milestone per CLAUDE.md *Render output must be human-verified before the iteration closes*: open `gaps.html` against the kernel's own planning tree (33 open + 2 addressed at planning time), confirm open rows pop on first scan, confirm addressed rows still visible peripherally.
- Tests assert the hierarchy via **Playwright** in `e2e/playwright/tests/` — open `gaps.html`, verify open rows render with the distinguishing computed-style (and/or earlier in DOM order, if the chosen mechanism reorders), addressed rows render with the de-emphasized computed-style. Parsed-HTML / parsed-CSS checks in Go remain for emit-shape (the `data-status` attribute on rows, the gap-hierarchy CSS rule's presence in `style.css`) but the visual hierarchy is browser-verified, since opacity / order / color decisions only become observable after browser layout. CI integration deferred per the epic Constraints.

The specific mechanism (grouped sections under `### Open` / `### Addressed` headings; CSS-driven row reordering via `order:` on flex/grid rows; per-row opacity / muted color for addressed; or a hybrid) is decided at start-milestone time. The milestone's *Design notes* section (in this spec, below) is filled in at start-milestone time with the chosen approach; the wrap *Validation* records what was visually verified.

## Constraints

- **All rows visible by default.** No collapse, no "Show addressed (N)" toggle. The goal is "open pops," not "addressed hides." A reader scanning should still see addressed rows in their peripheral vision.
- **No status filter chip.** The chip filter from M-β is binary (active / all archive state). Adding a status chip on top is out of scope.
- **Gaps kind specifically.** Other kinds unchanged. Same pattern can extend later if it proves out; the epic explicitly defers.
- **Structural test discipline.** Per CLAUDE.md, the hierarchy is asserted via parsed-HTML structural checks. Substring-only tests are rejected.

## Design notes

- Mechanism options to weigh at start-milestone time:
  - **(a) Grouped sections.** Render `### Open (N)` and `### Addressed (N)` subheadings with their own tables. Heaviest visual signal; clear "above the fold" placement of open. Downside: doubles the table render and the addressed table can look orphaned when the count is low.
  - **(b) CSS ordering.** Single table, but CSS `order:` (with display: flex/grid on `<tbody>`) sorts rows server-side as open-first / addressed-last. Mid-weight signal; preserves the flat table feel.
  - **(c) Visual weight.** Single table, addressed rows render at reduced opacity / muted color via a `tr[data-status="addressed"]` CSS rule. Lightest-weight signal; risk of being too subtle.
  - **(d) Hybrid.** Sort open-first server-side (option b), and apply muted styling to addressed rows (option c). Combines two signals.
- The kernel's existing pill styling for status (`.status-open` is amber `#bf8233`, `.status-cancelled` line-through) provides a starting palette; addressed-rows treatment should fit the existing token system.
- The same `data-status` attribute can also benefit M-γ's sidebar entry (e.g. a future iteration showing "Gaps (12 open, 21 addressed)") — but adding that breakdown to the sidebar is out of scope for this epic.

## Surfaces touched

- `internal/htmlrender/embedded/kind_index.tmpl` (primary — emit `data-status` on rows; conditional grouping if option (a) chosen)
- `internal/htmlrender/embedded/style.css` (status-hierarchy CSS rule for the gaps page)
- `internal/htmlrender/pagedata.go` (if option (a): add status grouping shape to `KindIndexData`)
- `internal/htmlrender/default_resolver.go` and `cmd/aiwf/render_resolver.go` (if option (a): bucket entries by status)
- `e2e/playwright/tests/` (primary test surface — extend `render.spec.ts` with gap-page hierarchy visual-state assertions, or add a sibling spec)
- `internal/htmlrender/htmlrender_test.go` (emit-shape test for `data-status` attribute and rule presence; complementary)
- `cmd/aiwf/render_archive_visibility_test.go` or a new `cmd/aiwf/render_gaps_hierarchy_test.go` (gaps-specific emit-shape assertions; complementary)

## Out of scope

- Same treatment for decisions / ADRs / contracts kind-index pages.
- Status filter chip (open / addressed / all).
- Collapse / hide-behind-toggle for addressed rows.
- Surfacing the same hierarchy in any other surface (sidebar, status page) — gaps page only.
- Sorting / grouping by other dimensions (parent, discovered-in, age).

## Dependencies

- M-β (chip filter) — depends_on. The `data-archived` attribute lands in M-β; this milestone adds `data-status` alongside it and the gap-hierarchy CSS rule operates on the post-M-β markup.

## References

- E-0029 (parent epic)
- G-0114 (gap closed) — names this sub-problem specifically
- `internal/htmlrender/embedded/kind_index.tmpl` — current per-kind index template
- `internal/htmlrender/embedded/style.css` — existing status-pill token system
- `CLAUDE.md` — *Substring assertions are not structural assertions*, *Render output must be human-verified before the iteration closes*

## Work log

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
