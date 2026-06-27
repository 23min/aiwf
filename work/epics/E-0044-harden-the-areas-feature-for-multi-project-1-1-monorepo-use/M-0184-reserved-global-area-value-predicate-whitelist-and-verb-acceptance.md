---
id: M-0184
title: 'Reserved global area value: predicate, whitelist, and verb acceptance'
status: in_progress
parent: E-0044
tdd: required
acs:
    - id: AC-1
      title: IsValidAreaValue predicate classifies global, members, and unknown values
      status: met
      tdd_phase: done
    - id: AC-2
      title: area-unknown treats global as known, including under areas.required
      status: met
      tdd_phase: done
    - id: AC-3
      title: set-area accepts the reserved global value
      status: met
      tdd_phase: done
    - id: AC-4
      title: add --area accepts the reserved global value
      status: met
      tdd_phase: done
    - id: AC-5
      title: validate() and rename-area reject a declared member named global
      status: met
      tdd_phase: done
    - id: AC-6
      title: a global-tagged entity satisfies areas.required
      status: open
      tdd_phase: red
    - id: AC-7
      title: read-filter note and --area completion recognize the global value
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

Each AC pins one observable behavior; the mechanical evidence that fails if the claim breaks is
named per AC. AC-1's predicate is the SSOT keystone — AC-2/3/4/7 route their "is this a valid
area value?" decision through it.

### AC-1 — IsValidAreaValue predicate classifies global, members, and unknown values

`entity.IsValidAreaValue(v, members)` returns true for `global` and for any declared member,
false otherwise. Pinned by a table-driven unit test in `internal/entity`. A `D-NNNN` recorded
during implementation documents why no literal-adoption policy is added now.

### AC-2 — area-unknown treats global as known, including under areas.required

`check.AreaUnknown` emits no finding for a `global`-tagged entity, so `ApplyAreaRequiredStrict`
has nothing to escalate — a `global` entity is not blocked under `areas.required: true`. Pinned
by a check test with `required:true` asserting zero findings for the global entity.
**Load-bearing.**

### AC-3 — set-area accepts the reserved global value

`aiwf set-area <id> global` tags the entity `global`, routing its declared-member check through
the predicate. Pinned by a verb/integration test.

### AC-4 — add --area accepts the reserved global value

`aiwf add --area global` creates a root-kind entity tagged `global`, including under
`areas.required` (the predicate is the validation site). Pinned by an add-cmd test.

### AC-5 — validate() and rename-area reject a declared member named global

`config.Areas.validate()` rejects a declared `areas.members` entry named `global`, and
`aiwf rename-area <old> global` refuses up front (the symmetric write path). Pinned by a config
unit test and a rename-area verb test.

### AC-6 — a global-tagged entity satisfies areas.required

The present-at-all `area-required` finding does not fire for a `global`-tagged entity (its area
is non-empty). Behavior already holds (`AreaRequired` skips non-empty areas); pinned by a check
test so a regression reddens.

### AC-7 — read-filter note and --area completion recognize the global value

`UndeclaredAreaNote` returns no "not a declared area" note for `global` (`aiwf list/show/status
--area global`), and `--area` completion offers `global` for `add`/`set-area` (not for
`rename-area <old>`). Pinned by cliutil + completion tests.

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
- The grouping/render resolver: a `global`-tagged entity falls into the "Uncategorized"
  complement in grouped views (it is not a declared member), so a global ADR renders there
  until M-0180/M-0181 give the oracle a domain that excludes `global` cleanly. A recognized,
  scoped limitation — not fixed in this milestone.

## Design notes

- The SSOT predicate's home is `internal/entity`: `entity.AreaGlobal = "global"` plus
  `entity.IsValidAreaValue(v, members)`. It must be `entity`, not `config` — the pure
  `internal/check` is config-agnostic by contract (M-0171/AC-4), so the `area-unknown` site
  cannot reach a token defined in `config`. `entity` is the lowest tier every consumer already
  reaches (`check`, `add`, `set-area` import it; `config` imports it sideways for the
  reserved-name guard — verified acyclic, `entity` imports only `codes`). The `area-unknown`
  check, `set-area`, `add --area`, and the read-filter note all route their "is this a valid
  area value?" decision through the predicate — no parallel `== "global"` checks.
- The reserved-name guard lives in `config.Areas.validate()` (rejecting a declared member
  named `global`) and is mirrored in `aiwf rename-area`'s newName guard — the second write path
  that could otherwise inject a `global` member behind `validate()`'s back.
- SSOT enforcement depth (AC-1): the predicate is unit-tested behaviorally; a literal-adoption
  policy (à la `enum_literal_adoption` / `closed_set_status_constants`) is deliberately *not*
  added now — one reserved token, one definition site, an area comparison has no clean
  syntactic marker to scan. A `D-NNNN` recorded during implementation captures the rationale
  and the revisit trigger (a third reserved area value → add the policy).
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

