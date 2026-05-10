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
      status: met
      tdd_phase: done
    - id: AC-2
      title: aiwf status hides sweep-pending line when count is zero
      status: met
      tdd_phase: done
    - id: AC-3
      title: aiwf status exposes no --archived flag and remains active-only
      status: met
      tdd_phase: done
    - id: AC-4
      title: aiwf show resolves any id (active or archived) without flag opt-in
      status: met
      tdd_phase: done
    - id: AC-5
      title: aiwf show renders a visible archived-state indicator in text and JSON
      status: met
      tdd_phase: done
    - id: AC-6
      title: aiwf render index page links active-default and full-set per-kind pages
      status: met
      tdd_phase: done
    - id: AC-7
      title: aiwf render emits per-kind all.html showing the full active+archived set
      status: open
      tdd_phase: done
    - id: AC-8
      title: aiwf render emits per-entity pages for archived entities so deep links resolve
      status: open
      tdd_phase: red
    - id: AC-9
      title: aiwf history walks across an aiwf-verb=archive sweep rename
      status: met
      tdd_phase: done
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

When the loaded tree carries one or more entities whose status is terminal-for-kind but whose path is in the active dir (the same condition the M-0086 `archive-sweep-pending` finding fires on), `aiwf status`'s output gains a one-line entry inside the **Health** section naming the count and the remediation verb: *"Sweep pending: N terminal entities not yet archived (run `aiwf archive --dry-run` to preview)"*.

The aggregate is lifted out of the general `warnings[]` list into a dedicated `sweep_pending` field on the status report — per ADR-0004 §"Display surfaces", the one-liner belongs in the tree-health section. The text renderer prints it inside `Health\n`; the JSON envelope exposes it as a top-level field with `count` and the formatted `message`. Per-file `terminal-entity-not-archived` warnings remain in the warnings stream alongside other findings.

Seam test drives `runStatusCmd` against a fixture with one terminal-in-active gap and asserts `"Sweep pending: 1"` appears inside the Health section. Structural assertion: the substring search is scoped to the post-`Health\n` portion of the output, not flat over the full text — a plain match would not distinguish "line in Health" from "line in Warnings."

### AC-2 — aiwf status hides sweep-pending line when count is zero

When no entity is terminal-in-active, the sweep-pending entry is omitted entirely — not "Sweep pending: 0", just absent. The existing `archive-sweep-pending` finding rule returns nil at zero, so `statusReport.SweepPending` stays nil and the renderer's `if r.SweepPending != nil` guard skips the line.

Two coverage branches pinned: a unit test that asserts `r.SweepPending == nil` against a fixture with only active-status entities (and an already-archived terminal, to exercise the path-filter exclusion), and a render-side test that asserts `"Sweep pending"` does not appear anywhere in the output when the field is nil.

### AC-3 — aiwf status exposes no --archived flag and remains active-only

`aiwf status --help` and the Cobra flag-set list **no** `--archived` flag. The verb's snapshot is forward-looking; archived inspection lives in `aiwf list --archived`. Verified via the completion-drift test (which enumerates every flag of every verb) plus a direct assertion on `newStatusCmd()`'s flag set.

### AC-4 — aiwf show resolves any id (active or archived) without flag opt-in

`aiwf show G-0099` resolves an archived entity identically to an active one — same JSON envelope shape, same text layout, no `--archived` flag required. M-0084/AC-4 already pinned the loader and dispatcher seam through `TestRun_ShowResolvesArchivedID`; this AC re-pins the contract specifically as a milestone-level acceptance check and ensures the assertion remains green after the indicator change in AC-5.

### AC-5 — aiwf show renders a visible archived-state indicator in text and JSON

When the resolved entity's path is under `<kind>/archive/`, the rendered output indicates that visibly:

- **JSON envelope:** a top-level `archived: true` field on `ShowView` (boolean; omitted on active entities via `omitempty`).
- **Text:** a terse `· archived` marker appended to the existing header line (no separate row, no badge — one word on the line the operator already reads). Active entities render unchanged.

Test the seam: JSON structural assertion on `archived == true` for an archived id and absence (or `archived == false`) for an active id; text structural assertion that scopes the substring match to the header line (first line of show output), not flat over the full body, so the marker can't accidentally pass by appearing elsewhere.

### AC-6 — aiwf render index page links active-default and full-set per-kind pages

`aiwf render --format=html` writes per-kind index pages keyed on the active set by default. The index references each per-kind page from the sidebar / home nav. For the four high-volume kinds (epic, gap, decision, ADR), the per-kind index renders only entities whose path is **not** archived. Each per-kind index page carries a plain `<a href="<kind>/all.html">` link to the full-set view (AC-7) so the navigation between active-default and full-set is reachable.

Structural assertion: parse the rendered index / per-kind pages, walk the DOM, and assert (a) the active-default page lists only non-archived entities, (b) the page contains an anchor whose `href` resolves to the all-set page. No JavaScript / no view-switching.

### AC-7 — aiwf render emits per-kind all.html showing the full active+archived set

For each kind that participates in archive segregation, `aiwf render` writes an additional `<kind>-all.html` page that lists the **full set** — active and archived entities together. The all-set page is reachable from the per-kind active-default page (AC-6) and links back to it via a `<a href="<kind>.html">` nav. Plain static `<a>` between the two views — no JS, no filter chips.

Structural assertion: parse all.html, walk the listed entities, and assert it contains both an active id (only listed in the active set) and an archived id (would otherwise not appear). Plus a back-link to the active-default page.

### AC-8 — aiwf render emits per-entity pages for archived entities so deep links resolve

A per-entity HTML page is written for every entity in the tree regardless of status — active or archived. External deep links to `G-0018.html` resolve whether the target is active or in `<kind>/archive/`. The page surface is identical except for a small archived-state marker (paralleling AC-5's `show` indicator) so the reader can see the entity is archived.

Test: render against a fixture carrying both an active and an archived gap, assert both `<active-id>.html` and `<archived-id>.html` exist on disk, and assert the archived page carries the archived-state marker via structural assertion.

### AC-9 — aiwf history walks across an aiwf-verb=archive sweep rename

After an `aiwf archive --apply` sweep produces a single commit trailered `aiwf-verb: archive` with the swept ids, `aiwf history <id>` returns a continuous timeline that includes events from before the move (the entity's add/promote commits in the active path) and the archive sweep commit itself. M-0084/AC-5 already exercised the trailer-based cross-rename walk; this AC re-pins the regression specifically with the `aiwf-verb: archive` trailer (rather than the synthetic shape M-0084 used) so the M-0085 verb's trailer keys are honored.

Test: synthesize the commit sequence (add → archive sweep) in a fixture, drive `aiwf history --format=json <id>`, assert the events array contains both the add event and the archive event in chronological order, with both events bearing the same entity id.

