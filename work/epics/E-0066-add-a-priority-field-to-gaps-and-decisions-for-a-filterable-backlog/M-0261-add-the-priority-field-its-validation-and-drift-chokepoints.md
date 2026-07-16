---
id: M-0261
title: Add the priority field, its validation, and drift chokepoints
status: draft
parent: E-0066
tdd: required
acs:
    - id: AC-1
      title: priority is an optional gap/decision field validated against its closed set
      status: open
      tdd_phase: red
    - id: AC-2
      title: priority on other kinds raises the priority-not-applicable finding
      status: open
      tdd_phase: red
    - id: AC-3
      title: drift chokepoints cover priority literals like status literals
      status: open
      tdd_phase: red
---

# M-0261 — Add the priority field, its validation, and drift chokepoints

## Goal

Define the `priority` frontmatter field on gap and decision, validate it on both axes (value-in-set and kind-scope), and extend the two literal-drift chokepoints so priority literals are protected like status literals. The foundation the write, read, and render surfaces all build on.

## Context

E-0066 adds `priority` to the two kinds where "which one do I work next" is an open question the kernel can't answer. After this milestone the field is defined and guaranteed but nothing sets or reads it yet — the writer and reader surfaces are separate milestones. The design mirrors the `area` feature: the field lives on the shared `Entity` struct and per-kind legality is enforced by check rules, not the type system.

## Acceptance criteria

<!-- Seeded via `aiwf add ac`; each starts at tdd_phase: red. -->

### AC-1 — priority is an optional gap/decision field validated against its closed set

### AC-2 — priority on other kinds raises the priority-not-applicable finding

### AC-3 — drift chokepoints cover priority literals like status literals

## Constraints

- The closed set (`urgent | high | medium | low`) is hardcoded in Go alongside kinds and statuses — no `aiwf.yaml` knob, because the set is genuinely closed (unlike `area`'s operator-declared members).
- `priority` sits on the shared `Entity` struct; per-kind legality is a `CarriesOwnPriority`-style predicate consulted by check rules, not a per-kind struct or a decode-time gate.
- Value validation is advisory (shape-only); scope validation is mechanical — "gap and decision only" must be an enforced fact, not prose.

## Design notes

- The scope rule (`priority-not-applicable`) is net-new check logic: the `area` precedent only ever gates *requiredness*, never *presence*, so nothing today rejects an out-of-scope field being present. Structure it off `internal/check/area_unknown.go` and pair it with a firing fixture (required by `firing_fixture_presence.go`).
- Chokepoint extensions: `enum_literal_adoption.go` harvests only `Status*`-prefixed constants today (an explicit "deliberate future-gap" note in-file) — widen to `Priority*`; `closed_set_status_constants.go` matches `Status:` / `.Status ==` / `TDDPhase:` contexts — add `Priority:` / `.Priority ==`.
- Whether the scope rule fires at warning or error severity is carried from the epic as an open question; default lean is warning, consistent with `area_unknown`.

## Surfaces touched

- `internal/entity/` — the `Priority` field, the gap/decision `OptionalFields` entries, the `CarriesOwnPriority` and closed-set-value predicates.
- `internal/check/` — the `priority-not-applicable` rule and its firing fixture.
- `internal/policies/` — `enum_literal_adoption.go`, `closed_set_status_constants.go`.

## Out of scope

- Any verb that writes the field, and any surface that reads it — those are the write-surface, read-surface, and render milestones under this epic.
- Sort ordering by priority — deferred to G-0420.

## Dependencies

- None — this is the foundation milestone; the other three depend on it.

## References

- G-0078 — the ratified design decisions this milestone executes.
- The `area` feature — `internal/check/area_unknown.go` / `area_required.go` and the `aiwf-area` skill — the design precedent.
