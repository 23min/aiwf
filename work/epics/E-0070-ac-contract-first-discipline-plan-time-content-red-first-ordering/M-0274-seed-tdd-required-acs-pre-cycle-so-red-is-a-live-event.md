---
id: M-0274
title: Seed tdd:required ACs pre-cycle so red is a live event
status: in_progress
parent: E-0070
tdd: required
acs:
    - id: AC-1
      title: aiwf add ac seeds ACs at the pre-cycle empty phase, not red
      status: open
      tdd_phase: done
    - id: AC-2
      title: A live empty-to-red phase promote succeeds from the seeded state
      status: open
      tdd_phase: red
    - id: AC-3
      title: Empty-phase ACs raise no acs-shape or acs-tdd-audit finding
      status: open
      tdd_phase: red
    - id: AC-4
      title: wf-tdd-cycle makes the empty-to-red promote a live mandatory step
      status: open
      tdd_phase: red
    - id: AC-5
      title: The --tests flag at add time is reconciled with pre-cycle seeding
      status: open
      tdd_phase: green
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

### AC-1 — aiwf add ac seeds ACs at the pre-cycle empty phase, not red

`aiwf add ac <M>` against a `tdd: required` milestone seeds the new AC at the
pre-cycle empty phase (`tdd_phase: ""`), not `red`. `red` means "a failing test
exists"; a freshly-added AC has written no test yet, so its honest resting phase
is absent. The failing test is recorded later, by a live
`aiwf promote <M>/AC-<N> --phase red`.

Mechanical evidence: a verb-level test adds an AC under a `tdd: required`
milestone and asserts the resulting frontmatter `acs[]` entry carries an empty
`tdd_phase`.

### AC-2 — A live empty-to-red phase promote succeeds from the seeded state

From an AC seeded at the empty phase, `aiwf promote <M>/AC-<N> --phase red`
succeeds as a live `"" -> red` transition — the event that means "a failing test
has been written and shown to fail." Before this milestone the AC was born at
`red`, so this transition could never fire: the phase FSM refuses `red -> red`.

Mechanical evidence: a verb/integration test seeds an AC at the empty phase, runs
the `--phase red` promote, and asserts the transition succeeds with the AC then
resting at `red`.

### AC-3 — Empty-phase ACs raise no acs-shape or acs-tdd-audit finding

A `tdd: required` milestone whose ACs rest at the empty phase through both
`draft` and `in_progress` raises no `acs-shape` or `acs-tdd-audit` finding — an
absent phase is legal until the AC is promoted to `met`. This is the check-layer
behavior G-0286 already ratified; this milestone makes the seeder agree with it
rather than contradict it.

Mechanical evidence: a check-level test builds a `tdd: required` milestone with
empty-phase ACs, once in `draft` and once in `in_progress`, runs `aiwf check`,
and asserts neither `acs-shape` nor `acs-tdd-audit` fires.

### AC-4 — wf-tdd-cycle makes the empty-to-red promote a live mandatory step

The `wf-tdd-cycle` ritual no longer instructs the operator to skip the red
promote for `tdd: required` ACs. It names the `"" -> red` promote as a live,
mandatory RED step, run the moment the failing test is written and shown to
fail.

Mechanical evidence: a structural policy test under `internal/policies/` asserts
the embedded `wf-tdd-cycle` `SKILL.md` drives a live `--phase red` promote and
no longer carries "skip the red promote" guidance — the skill-edit
structural-test backstop.

### AC-5 — The --tests flag at add time is reconciled with pre-cycle seeding

The `--tests` flag on `aiwf add ac` (previously "only valid when seeding red")
is reconciled with pre-cycle seeding: because ACs are no longer born at `red`,
the flag's home moves to the `--phase red` promote or the flag is removed
outright. The chosen resolution is pinned by a verb-level test so it cannot
silently regress.

Mechanical evidence: a verb-level test exercises the reconciled behavior (test
metrics accepted at the `--phase red` promote, or the flag's removal refused at
`add`) and asserts the outcome.

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
