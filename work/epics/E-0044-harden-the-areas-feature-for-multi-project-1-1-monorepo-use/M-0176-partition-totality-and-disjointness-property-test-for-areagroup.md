---
id: M-0176
title: Partition totality and disjointness property test for areagroup
status: draft
parent: E-0044
tdd: advisory
---

## Goal

Mechanically guarantee that `internal/areagroup.Partition` never silently drops or duplicates an item: for any input, every item lands in exactly one output group. Turns the view-layer drop failure from "hoped-for" into "impossible" — the Tier-0 floor under E-0044's trust claim.

## Context

E-0043 shipped `areagroup.Partition` as the single source of the area-partition logic shared by `status`, roadmap, and HTML renders. Its correctness is currently pinned only by example-based tests; a refactor that drops an item into neither bucket — or into both — would pass them. This milestone replaces that hope with a generative property, per the `wf-property-test` skill. No production change is expected unless the property surfaces a real defect.

## Acceptance criteria

<!-- Candidate ACs, formalized via `aiwf add ac M-0176 --title "..."` at start-milestone. -->

Candidate properties to formalize at start-milestone:

- **Totality + disjointness** — every input item appears in exactly one output group (count-in == count-out; no item in two groups; none dropped).
- **Complement correctness** — the complement group (Area "") holds exactly the items whose area is "" or not a declared member, and nothing else.
- **Declared order + suppression** — declared areas appear in `members` order; an empty declared area is suppressed; the complement is always emitted last.

## Constraints

- Pure test addition on `internal/areagroup`; no change to `Partition`'s signature or behavior unless the property catches a real bug — in which case the fix lands here with its own regression test.
- The generator covers arbitrary item slices, arbitrary `areaOf` mappings (including "", declared, and undeclared values), and arbitrary `members` / `defaultLabel`.

## Out of scope

- The `paths:` oracle and any path-based checks (Tier 1+).
- Redesigning `Partition`'s ordering or emptiness policy — those are pinned, not changed.

## Dependencies

- None. Independent Tier-0 hardening; parallel with the other Tier-0 milestones.

## References

- `internal/areagroup/areagroup.go` — the `Partition` helper under test.
- `wf-property-test` skill — the generative-property discipline this milestone applies.
