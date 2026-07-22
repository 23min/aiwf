---
id: G-0441
title: aiwf add ac seeds tdd:required ACs at red before any test exists
status: open
priority: high
---
## What's missing

`aiwf add ac` seeds every acceptance criterion under a `tdd: required`
milestone directly at `tdd_phase: red` (`internal/verb/ac.go:122-124`). But
`red` means "a failing test has been written and shown to fail" — an AC at
creation has no test, so `red` misrepresents it. ACs should be born at the
pre-cycle `""` state and reach `red` only via a live promote once the failing
test exists.

This contradicts the model the kernel already committed to and enforces
elsewhere:

- **G-0286 (addressed)** ratified that `red` means "a failing test exists,"
  that the phase enum has no "not started" member, and that `tdd: required`
  commits only to "every AC reaches `done` before `met`" — not "every AC is
  phase-tracked from creation." The strict born-at-red reading was the one
  option it explicitly rejected.
- The check layer already treats an **absent** phase (`""`) as the honest
  resting state: `internal/check/acs.go:154-160` errors only on a
  present-but-invalid phase, never on absence; `acsTDDAudit` requires `done`
  only at `met`. The FSM documents `""` as the "pre-cycle entry state"
  (`internal/entity/transition.go:226`) with a legal live `"" -> red`
  transition.

G-0286 fixed the **check** half and was scoped to only that half. The
**seeder** was never updated to match, leaving the kernel internally
contradictory: the check says "absent is honest, `red` means a test failed,"
while the seeder stamps `red` on a testless AC.

## Why it is a bug, not just friction

Born-at-red destroys the one honest, live event in the TDD cycle: the
`"" -> red` promote that means "I wrote the failing test." Because the AC is
born at `red`, that transition is already spent (the FSM refuses `red -> red`),
and `wf-tdd-cycle` guidance tells the operator to skip the red promote. The
result is that no live event marks "the test now exists and fails" — which is
exactly the event any red-first ordering guarantee must attach to. The
workflow cannot be made mechanically red-first while ACs are born red.

## Corrected workflow

    aiwf add ac                -> AC born at ""  (pre-cycle; no test yet)
    write the failing test     -> shown to fail
    aiwf promote --phase red   -> live: "test written, it fails"
    write the implementation
    aiwf promote --phase green -> test passes
    (aiwf promote --phase refactor)
    aiwf promote --phase done
    aiwf promote met

## Fix and consequences to sweep together

- Seed `tdd: required` ACs at `""`, not `red`, in `aiwf add ac`.
- Reverse `wf-tdd-cycle`'s "the AC was already seeded at red; skip this step"
  guidance — the `"" -> red` promote becomes a live, mandatory step. The skill
  edit needs its referencing structural test under `internal/policies/` per
  the skill-edit backstop.
- Reconcile the `--tests` flag on `aiwf add ac`
  (`internal/verb/ac.go:106-108`, "only valid when seeding red"): recording
  test metrics at creation, before a test exists, is the same born-at-red
  category error — it moves to the red promote or is removed.
- Audit existing tests that assume an AC is born at `red` under
  `tdd: required`.

## Relationship

- **G-0286** — did the check-layer half of this correction; this gap is the
  seeder half it left out of scope.
- **G-0252 / D-0047** — the red-first ordering gate; this seeding fix is its
  prerequisite (it creates the live `"" -> red` event the gate attaches to).
