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
