---
id: M-0267
title: Relax acs-shape/tdd-phase to allow absent phase until AC met
status: in_progress
parent: E-0068
tdd: required
acs:
    - id: AC-1
      title: 'Absent tdd_phase is legal on a non-met AC under tdd: required'
      status: open
      tdd_phase: green
    - id: AC-2
      title: 'Regression: tdd-phase closed-set and met-requires-done checks unchanged'
      status: open
      tdd_phase: red
---

# M-0267 — Relax acs-shape/tdd-phase to allow absent phase until AC met

## Goal

Stop `acs-shape/tdd-phase` from forcing every AC in a `tdd: required` milestone to carry a `tdd_phase` from the moment it's created — an absent phase should be legal until the AC reaches `status: met`, matching what the design actually commits to.

## Context

`internal/check/acs.go`'s `acsShape` function (~line 155) currently fires `CodeACsShape`/`tdd-phase` whenever `ac.TDDPhase == "" && tddRequired`, regardless of the AC's own status. That's stricter than CLAUDE.md #8 and `design-decisions.md`, which commit only to "`AC met` requires `tdd_phase: done`" when the milestone is `tdd: required` — not "every AC is phase-tracked from creation." The practical effect (G-0286): strengthening a milestone `advisory → required` reddens the tree for every pre-existing AC, and the only way to clear it today is to seed untouched ACs to `red`, which misrepresents them (`red` means "a failing test exists").

## Acceptance criteria

### AC-1 — Absent tdd_phase is legal on a non-met AC under tdd: required

Under a `tdd: required` milestone, an AC with `status` anything other than `met` and an absent `tdd_phase` produces no `acs-shape/tdd-phase` finding. This is the core relaxation: `tdd_phase` becomes optional until the AC is claimed `met`, not mandatory from creation. Covers `status: open` explicitly (the case G-0286 traces — a milestone freshly upgraded `advisory → required` with pre-existing open ACs that have never had a phase set).

### AC-2 — Regression: tdd-phase closed-set and met-requires-done checks unchanged

Two existing behaviors must survive the relaxation untouched: (1) a *present* `tdd_phase` value outside the closed set (`red|green|refactor|done`) still fires `acs-shape/tdd-phase` exactly as today; (2) `acsTDDAudit`'s rule — an AC with `status: met` under `tdd: required`/`advisory` still requires `tdd_phase: done`, firing at error/warning severity respectively when it doesn't — is completely unaffected by this milestone's change. This AC exists because the relaxation touches the same conditional block as the closed-set check, and the audit rule (in a different function, `acsTDDAudit`) is the property this milestone must not weaken.

## Constraints

- The `acsTDDAudit` function itself is out of scope for this milestone — no code changes there, only regression coverage confirming it still fires as before.
- No change to `entity.IsAllowedTDDPhase` or the phase enum (`red|green|refactor|done`) — this milestone doesn't add a "not started" phase value; absence itself now means "not started."

## Design notes

- Per G-0286's fork: the "strict reading" (every AC is phase-tracked from creation) is explicitly rejected in favor of the "committed reading" (every AC reaches `done` before `met`) — see G-0286's own body for the full argument. No new decision entity was needed for this milestone; the gap itself is the settled design.

## Surfaces touched

- `internal/check/acs.go` — `acsShape` (the `tdd-phase` subcode's conditional).

## Out of scope

- G-0168's set-policy verb (auto-seed-vs-refuse on `advisory → required`) — this milestone only fixes the check-layer rule; per G-0286, relaxing this rule means that verb no longer needs an auto-seed decision at all, but building the verb itself is tracked separately in G-0168.
- Any change to `wf-tdd-cycle` or other ritual-content guidance on when to set `tdd_phase`.

## Dependencies

- None. Independent of M-0268 and of D-0039 (M-0267 predates neither).

## References

- [G-0286](../../gaps/G-0286-acs-shape-tdd-phase-over-demands-a-phase-on-every-ac-under-tdd-required.md) — source gap, fully specifies the fix and the design fork's resolution.
- CLAUDE.md §"What aiwf commits to", item 8 — the committed reading this milestone brings the check in line with.
