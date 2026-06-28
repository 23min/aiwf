---
id: M-0184
title: 'Reserved global area value: predicate, whitelist, and verb acceptance'
status: done
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
      status: met
      tdd_phase: done
    - id: AC-7
      title: read-filter note and --area completion recognize the global value
      status: met
      tdd_phase: done
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

`entity.IsValidAreaValue(v, members)` returns true for `global` and any declared member when a
block is declared, false otherwise — including `global` when no members are declared, so the
predicate itself enforces Position A (the dimension is inert until a block exists). Pinned by a
table-driven unit test in `internal/entity`. Decision D-0026 documents why no literal-adoption
policy is added now.

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
  area value?" decision through the predicate — no parallel `== "global"` checks. The predicate
  also enforces Position A at the SSOT: with no declared members the dimension is inert (M-0171),
  so it returns false for every value including `global`. The call sites keep their own no-block
  pre-guards for clearer messages, but correctness no longer depends on each caller remembering
  to gate (closes the footgun the design review surfaced).
- The reserved-name guard lives in `config.Areas.validate()` (rejecting a declared member
  named `global`) and is mirrored in `aiwf rename-area`'s newName guard — the second write path
  that could otherwise inject a `global` member behind `validate()`'s back.
- SSOT enforcement depth (AC-1): the predicate is unit-tested behaviorally; a literal-adoption
  policy (à la `enum_literal_adoption` / `closed_set_status_constants`) is deliberately *not*
  added now — one reserved token, one definition site, an area comparison has no clean
  syntactic marker to scan. Decision D-0026 captures the rationale and the revisit trigger
  (a third reserved area value → add the policy).
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

## Work log

All seven ACs landed in the single feature commit `5a89eea9`; `0c6586fc` hardened the predicate
afterward (Position A folded into the SSOT, plus two review-flagged test improvements). The
per-AC TDD phase timeline is in `aiwf history M-0184/AC-<N>`.

- **AC-1** — `entity.AreaGlobal` + `entity.IsValidAreaValue`; table-driven unit test. `5a89eea9`,
  hardened in `0c6586fc` (empty members → false, global included).
- **AC-2** — `check.AreaUnknown` routes through the predicate; load-bearing test pairs a global
  entity with a real mistag and asserts no escalation under `required:true`. `5a89eea9`.
- **AC-3** — `verb.SetArea` accepts `global`; verb test + full-dispatcher end-to-end success
  test (added in `0c6586fc`). `5a89eea9`.
- **AC-4** — `cli/add.validateAreaMember` accepts `global` with a block (usage error with no
  block); dispatcher-driven test. `5a89eea9`.
- **AC-5** — reserved-name guard in `config.Areas.validate()` and `verb.RenameArea` (both write
  paths); config + rename-area tests. `5a89eea9`.
- **AC-6** — pin-only (no production change): a `global`-tagged entity gets no `area-required`
  finding under `required:true`. `5a89eea9`.
- **AC-7** — `UndeclaredAreaNote` + `--area` completion recognize `global` (offered for
  add/set-area/read-filters, not `rename-area <old>`); cliutil + completion tests. `5a89eea9`.

## Decisions made during implementation

- **D-0026 — Defer literal-adoption policy for the `global` sentinel.** The SSOT predicate is
  pinned by a behavioral unit test; a chokepoint forbidding bare `== "global"` is deferred until
  a third reserved area value appears (one token, one definition site, no clean syntactic marker
  to scan today).
- **Position A (feature-gated `global`), chosen in conversation.** With no declared areas block
  the dimension is inert (M-0171), so `global` is unavailable consistently — not a universal
  always-on sentinel. Captured in the commit messages and Design notes; folded into the predicate
  itself in `0c6586fc` so correctness does not depend on per-caller guards.

## Validation

- `make ci` — green (full CI-parity gate; `aiwf doctor --self-check` 29 steps pass).
- `go test ./...` — green (full suite, race-clean under CI).
- `golangci-lint run` — 0 issues.
- Independent two-lens review before wrap: `wf-review-code` (APPROVE — every load-bearing claim
  confirmed by measurement, no blocking findings) and `wf-rethink` (design sound; the one
  surfaced footgun was fixed in `0c6586fc`, not deferred).

## Deferrals

None new. The oracle domain-exclusion, mistag skip, stronger seam check, and the grouping/render
resolver are scoped to M-0180/M-0181 and ADR-0021 (see "Out of scope"); no fresh gap opened.

## Reviewer notes

- The Position-A asymmetry the design review flagged (predicate accepted `global` regardless of
  `members`, with Position A held only by per-caller guards) was judged a *correctness* gap, not
  a deferral: the predicate is the SSOT for "valid area value", so under Position A it must
  return false for `global` with no block. Fixed in `0c6586fc`; the per-caller no-block guards
  remain for their clearer messages.
- Two non-blocking review items resolved inline: a cross-reference comment pairing the strict-mode
  area-unknown test with its firing sibling, and a full-dispatcher integration test for the
  `set-area <id> global` success path.
- Deliberate omission: no kind-level default tagging ADR/decision `global` (YAGNI per ADR-0021);
  the explicit `area: global` tag serves all cross-cutting kinds uniformly.

