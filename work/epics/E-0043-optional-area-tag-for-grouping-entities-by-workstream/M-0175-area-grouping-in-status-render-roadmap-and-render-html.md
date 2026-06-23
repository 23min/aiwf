---
id: M-0175
title: Area grouping in status, render roadmap, and render html
status: draft
parent: E-0043
depends_on:
    - M-0171
tdd: required
acs:
    - id: AC-1
      title: a shared Partition helper groups items by area, complement always last
      status: open
      tdd_phase: red
    - id: AC-2
      title: status groups in-flight and planned epics by area when areas declared
      status: open
      tdd_phase: red
    - id: AC-3
      title: render roadmap groups its epics by area when areas declared
      status: open
      tdd_phase: red
    - id: AC-4
      title: render --format=html groups epic sections by area, asserted DOM-structurally
      status: open
      tdd_phase: red
---
## Goal

Add per-area grouping to the three presentation surfaces — `status`, `render roadmap`, and `render --format=html` — so each renders a section per declared area, with untagged entities collected under the `aiwf.yaml: areas` `default:` display label.

## Context

M-0171 exposes each entity's effective area; the filter milestone narrows a view to one area. This milestone partitions a *whole* view into per-area sections — the readable payoff of the feature for a monorepo or a co-developed tool. It is the heaviest milestone (three surfaces) and may be split at `aiwfx-start-milestone` if the grouping helper + three renderers exceed a single focused cycle.

## Acceptance criteria

<!-- ACs allocated at aiwfx-start-milestone via `aiwf add ac` against this milestone.
     Candidate AC titles, drafted here as prose hints (not yet kernel state): -->

- **AC-1 candidate** — A shared grouping helper partitions the loaded entities by effective area, with the untagged complement under the configured `default:` label (or a built-in fallback label when `default:` is unset).
- **AC-2 candidate** — `aiwf status` renders a section per area (plus the default complement) when an `areas` block exists; unchanged single-list output when it does not.
- **AC-3 candidate** — `aiwf render roadmap` groups its epic/milestone table by area.
- **AC-4 candidate** — `aiwf render --format=html` groups by area, asserted **DOM-structurally** (parse the HTML, assert the area sections/containers), not by flat substring match (per CLAUDE.md "substring assertions are not structural assertions").
- **AC-5 candidate** — Empty-area handling per Open Question 2 in the epic (lean: suppress an area section with zero entities; always show the default complement).
- **AC-6 candidate** — With no `areas` block, all three surfaces render exactly as today (zero-migration / inert).

## Constraints

- **One grouping helper, three consumers** — single source of truth for the partition logic; the three renderers format it, they don't each re-derive it.
- **Read-only.** No mutation.
- **Zero-migration parity** — untagged trees render byte-equivalently to today (modulo the optional default-complement heading when an `areas` block exists).

## Out of scope

- The `--area` filter (separate milestone; filter narrows, grouping partitions — they compose but ship separately).
- Grouping any surface beyond status / roadmap / html (e.g. `show`).

## Dependencies

- M-0171 — effective-area exposure on the loaded model.

## References

- [E-0043 epic](epic.md) · [G-0266](../../gaps/G-0266-optional-area-tag-for-grouping-entities-by-workstream.md)

### AC-1 — a shared Partition helper groups items by area, complement always last

### AC-2 — status groups in-flight and planned epics by area when areas declared

### AC-3 — render roadmap groups its epics by area when areas declared

### AC-4 — render --format=html groups epic sections by area, asserted DOM-structurally

