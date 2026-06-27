---
id: M-0184
title: 'Reserved global area value: predicate, whitelist, and verb acceptance'
status: draft
parent: E-0044
tdd: required
acs:
    - id: AC-1
      title: IsValidAreaValue predicate classifies global, members, and unknown values
      status: open
      tdd_phase: red
    - id: AC-2
      title: area-unknown treats global as known, including under areas.required
      status: open
      tdd_phase: red
    - id: AC-3
      title: set-area accepts the reserved global value
      status: open
      tdd_phase: red
---
## Goal

Implement the reserved `global` area value (ADR-0021) as a value-acceptance layer: a single
SSOT predicate for "valid area value", the `area-unknown` whitelist (the load-bearing site),
acceptance in the tagging/creation verbs, and the reserved-member guard. This is the
prerequisite that lets the path oracle exclude `global` from its domain without special-casing.

## Context

ADR-0021 introduces `global` as an explicit, reserved value of the single-valued `area`
dimension — the named, affirmative not-1:1 escape valve for inherently-cross-cutting entities
(ADRs, decisions, seam contracts). `areas.required` stays total: `global` is a satisfying
assignment chosen affirmatively, never inferred from absence.

A reserved value must be accepted everywhere a declared member is validated today. The
oracle-dependent checks (the bijection/coverage check, mistag detection) handle their own
domain exclusion in their own milestones; this milestone delivers only the value-acceptance
surface they build on.

## Acceptance criteria

<!-- Candidate ACs, formalized via `aiwf add ac <id> --title "..."` at start-milestone. -->

Candidate behaviors to formalize at start-milestone:

- A single SSOT predicate `IsValidAreaValue(v, members) = v == "global" || isMember(v, members)`
  is the one definition of "valid area value"; the reserved token is defined once, not
  re-litigated per rule.
- The `area-unknown` present-⇒-declared check treats `global` as known — **including under
  `areas.required`**, where that finding escalates to error. The load-bearing AC: a
  `global`-tagged entity is NOT blocked under strict mode.
- `aiwf set-area <id> global` accepts the reserved value and tags the entity `global`.
- `aiwf add --area global` accepts the reserved value at creation (for the self-tagging root
  kinds), including under `areas.required`.
- `config.Areas.validate()` rejects a declared `areas.members` entry named `global` (the
  reserved-name guard), so a real project cannot shadow the sentinel.
- A `global`-tagged entity satisfies `areas.required` — pin that the present-at-all
  (`area-required`) finding does not fire for it.

## Constraints

- **`area` stays single-valued.** `global` is one more value of the dimension, not a second
  axis and not multi-value.
- **`global` is chosen affirmatively, never inferred from absence** — the whole point of the
  ADR-0021 decision; absence still errors under `areas.required`.
- **Token is `global`, a reserved member name.** The kernel rejects it as a declared member.
- **Default views never hide.** `global` entities surface under `--area global` and in every
  unscoped view.

## Out of scope

- The oracle domain-exclusion (excluding `global` from the bijection/coverage accounting) —
  that lands in the bijection-check milestone, which owns the path matching.
- Mistag-detection's skip of `global` entities — that lands in the mistag milestone.
- The stronger seam check (a `global` seam contract's paths within the bridged union) — a
  later idea, deferred in ADR-0021.

## Design notes

- The SSOT predicate's home is `internal/config` (or `internal/entity`) beside the area
  vocabulary; the `area-unknown` check, `set-area`, and `add --area` all route their
  "is this a valid area value?" decision through it — no parallel "is global" checks.
- The reserved-name guard is a small addition to `config.Areas.validate()`.
- Independent of the `paths:` schema — this is the area *value* layer, orthogonal to the
  member *location* schema; the two only meet in the consumers (bijection, mistag).

## Dependencies

- None (hard). The value layer does not need the `paths:` schema; it sequences naturally after
  M-0179 only because both touch `config.Areas`. The bijection/coverage milestone (M-0180)
  depends on THIS milestone so it can exclude `global` from its domain.

## References

- [ADR-0021](../../../docs/adr/ADR-0021-sanctioned-global-area-value-for-inherently-cross-cutting-entities.md) — the decision this implements.
- M-0179 — the dual-form member schema (the `config.Areas` code this sits beside).
- M-0180 — the bijection/coverage check that depends on this milestone.

### AC-1 — IsValidAreaValue predicate classifies global, members, and unknown values

### AC-2 — area-unknown treats global as known, including under areas.required

### AC-3 — set-area accepts the reserved global value

