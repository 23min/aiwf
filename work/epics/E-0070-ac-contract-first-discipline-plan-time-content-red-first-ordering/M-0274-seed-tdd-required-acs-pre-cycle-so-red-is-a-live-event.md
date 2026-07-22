---
id: M-0274
title: Seed tdd:required ACs pre-cycle so red is a live event
status: draft
parent: E-0070
tdd: required
---

# M-0274 — Seed tdd:required ACs pre-cycle so red is a live event

## Goal

Fix the born-at-red seeding bug (G-0441): `aiwf add ac` on a `tdd: required`
milestone seeds the new AC at the pre-cycle `""` state, not `red`, so the
`"" -> red` promote becomes a live event that means "a failing test has been
written." This is the prerequisite that gives the M-0276 ordering gate a real
promote to attach to.

## Context

`internal/verb/ac.go:122-124` stamps `tdd_phase: red` on every AC born under a
`tdd: required` milestone. But `red` means "a failing test exists" — G-0286
(addressed) already ratified that meaning and the check layer already enforces
it (`internal/check/acs.go` treats an absent phase as legal until `met`).
G-0286 fixed only the check half; the seeder still contradicts it. Born-at-red
also spends the AC's one `"" -> red` transition (the FSM refuses `red -> red`)
and `wf-tdd-cycle` tells the operator to skip the red promote — so no live
event marks "the test now fails," which is the event any red-first ordering
guard must attach to. Full rationale in D-0047.

## Acceptance criteria

## Constraints

- `red` must mean "a failing test exists" — no state may be auto-assigned that
  claims a test that does not exist.
- Existing ACs already born at `red` stay valid — no backfill/migration; only
  new `aiwf add ac` calls change behavior (the check layer already tolerates
  both absent and `red`).

## Design notes

- Implements D-0047's seeding-correctness prerequisite; closes G-0441.
- Follows the model G-0286 ratified (the check-layer half of this correction).

## Surfaces touched

- `internal/verb/ac.go` (AddAC seeding path; `--tests` handling)
- `internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-tdd-cycle/SKILL.md`
- `internal/policies/` (structural test for the skill edit)

## Out of scope

- The ordering gate itself (M-0276) and the plan-time AC-content changes
  (M-0275).

## Dependencies

- None. This is the epic's foundational milestone.

## References

- G-0441 — the seeding-correctness gap this milestone closes.
- D-0047 — Contract-first AC timing and red-first ordering enforcement.
- G-0286 — the accepted decision that `red` means "a failing test exists."
