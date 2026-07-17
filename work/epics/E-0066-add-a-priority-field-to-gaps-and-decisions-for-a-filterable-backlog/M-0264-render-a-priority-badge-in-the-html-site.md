---
id: M-0264
title: Render a priority badge in the HTML site
status: in_progress
parent: E-0066
depends_on:
    - M-0261
tdd: advisory
acs:
    - id: AC-1
      title: aiwf render emits a priority badge for each gap/decision with a value
      status: met
    - id: AC-2
      title: the badge is asserted structurally, not by substring, and human-verified
      status: met
---

# M-0264 — Render a priority badge in the HTML site

## Goal

Surface each gap's and decision's `priority` as a badge in the `aiwf render` HTML site, so the backlog's importance is visible at a glance in the rendered governance views.

## Context

The field exists (field milestone) and is set and filterable via the other surface milestones. This milestone adds the visual read path. The HTML renderer has no generic per-entity metadata/column abstraction to reuse, so the badge is bespoke template work — hence `tdd: advisory`: the deliverable is visual, human-verification is the real gate, and a structural HTML assertion is the mechanical backstop.

## Acceptance criteria

<!-- Seeded via `aiwf add ac`; each starts at tdd_phase: red. -->

### AC-1 — aiwf render emits a priority badge for each gap/decision with a value

### AC-2 — the badge is asserted structurally, not by substring, and human-verified

## Constraints

- The badge appears only for gaps and decisions carrying a value; an unset priority renders nothing (no empty badge).
- AC evidence is a **structural** assertion — parse the HTML and assert the badge inside the entity's section/attribute, not a substring grep (per the repo's "substring assertions are not structural assertions" rule).
- The render is verified by eye against the kernel's own planning tree before the milestone closes; the test does not stand in for the look.

## Design notes

- No column/badge abstraction exists — the `area` tag reaches templates via a bespoke `data-area` construct, not a reusable component. Keep the priority badge minimal and self-contained.

## Surfaces touched

- `internal/htmlrender/` — the template(s) and page-data plumbing for the badge.

## Out of scope

- Text/JSON surfaces (the read-surface milestone); writing the field (the write-surface milestone).
- Sort ordering — G-0420.

## Dependencies

- M-0261 — the field must exist first. Independent of the write and read surface milestones (fixtures set the field directly).

## References

- G-0078 — the ratified design decisions (HTML badge in scope, sort deferred).

## Work log

### AC-1 — aiwf render emits a priority badge for each gap/decision with a value

`Priority` field added to `htmlrender.KindIndexEntry` and `htmlrender.EntityRef`, populated from `entity.Entity.Priority` at both resolver sites — the cmd-side `internal/cli/render.Resolver` (production `aiwf render`) and the in-package `defaultResolver` (the nil-resolver fallback and the package's own test baseline). `kind_index.tmpl` gained a `Priority` table column rendering `{{if .Priority}}<span class="priority priority-{{.Priority}}">{{.Priority}}</span>{{end}}` per row (gaps.html/decisions.html); `entity.tmpl` gained the same conditional badge next to the status pill on the individual gap/decision detail page. No kind-gate needed — the same "no separate carrying-kind check" mechanism established for the list/status filter (M-0263) applies here: a kind that never carries a priority (epic, milestone, ADR, contract) simply has an always-empty `Priority` field, so its row/page never renders the badge, the identical path an untagged gap/decision takes.

`style.css` gained one token per priority level (`--priority-urgent/high/medium/low`, light and dark) plus four `.priority-<level>` pill-color rules, reusing the existing generic `.status, .phase, .tdd, .scope-state, .policy` pill mechanism (now `.priority` joins that selector) rather than a new bespoke pill implementation — urgent/high sit on the same warm red/orange tokens as `tdd-required`/`tdd-advisory`, medium on the accent, low muted.

commit f1910741 · tests: 3 new structural tests in `internal/cli/integration/priority_badge_html_test.go` (kind-index rows, entity detail page, non-carrying-kind never renders) plus one assertion added to the package-level `TestRender_FixtureTree_FilesAndLinks` (`internal/htmlrender/htmlrender_test.go`) exercising the `defaultResolver` wiring directly · `wf-vacuity` mutation probe: 4/4 mutants killed (kind-index conditional removed, entity-page conditional removed, cmd-side resolver hardcoded to a wrong level, in-package resolver hardcoded to empty) — each confirmed red under the mutation and restored byte-identical via captured content, never `git stash`/`checkout`.

### AC-2 — the badge is asserted structurally, not by substring, and human-verified

Every badge assertion is scoped by containment, not a page-wide substring match: `priorityRowSlice` isolates one entity's own `<tr>...</tr>` inside the `<table class="kind-index">` body (deliberately excluding the sidebar, which also carries `href="<id>.html"` links for epics — a page-wide search false-matched there before this scoping was added, caught while writing the non-carrying-kind test), and `entityHeaderSlice`/`entityHeaderBlock` isolate the detail page's header block between `</h1>` and the first body `<section>`. Both mirror the repo's established `areaGroupSlice`/`sectionByTab` containment-scoping idiom rather than introducing `golang.org/x/net/html` as a new dependency the repo doesn't otherwise carry.

Human verification: rendered the real `aiwf` binary against a git snapshot of this repo's own planning tree (`aiwf set-priority` tagging three real entities — a gap urgent, a gap high, a decision medium — then `aiwf render --format html`), and read the emitted `gaps.html`/`decisions.html`/detail-page/`assets/style.css` output directly — badge present with the right level on tagged rows and the detail-page header, absent (empty `<td></td>`, no empty span) on an untagged gap's row, and the new CSS tokens/rules present in the emitted stylesheet. No browser was available in this environment to confirm rendered color/layout beyond the raw markup and CSS.

commit f1910741 · same test suite as AC-1 (the two ACs share one implementation pass across the same set of files, mirroring M-0263's combined-AC-implementation precedent).
