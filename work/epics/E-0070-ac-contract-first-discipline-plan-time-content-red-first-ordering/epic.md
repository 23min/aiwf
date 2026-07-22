---
id: E-0070
title: 'AC contract-first discipline: plan-time content, red-first ordering'
status: active
---

# E-0070 — AC contract-first discipline: plan-time content, red-first ordering

## Goal

Close three gaps in the kernel's "TDD discipline must not depend on the LLM's
behavior" guarantee: acceptance-criterion contracts land on main before
implementation starts; the TDD-phase model is corrected so an AC reaches
`red` only when a failing test exists (not at creation); and red-first
test-then-code ordering is then mechanically checked at that live `red`
event rather than trusted from a self-reported phase timeline.

## Context

Two related gaps surfaced from a single design conversation (recorded in
D-0047) about closing structural discipline holes the kernel's own ritual
text already names but doesn't enforce:

- **G-0252** — `wf-tdd-cycle` asks for a failing test before the
  implementation, but nothing enforces the ordering: the phase-transition
  commits (`--phase red`/`--phase green`/`--phase done`) are metadata-only,
  and the actual test-plus-implementation code lands in one combined commit
  at the end of the cycle. The skill's own text names the gap ("a phase
  ladder stamped in a batch later is indistinguishable from one
  back-stamped after the fact") without closing it.
- **G-0441** — the prerequisite the ordering gate depends on. `aiwf add ac`
  seeds `tdd: required` ACs directly at `tdd_phase: red`
  (`internal/verb/ac.go:122-124`), but `red` means "a failing test exists."
  A born-at-red AC has already spent its one `"" -> red` transition (the FSM
  refuses `red -> red`) and `wf-tdd-cycle` tells the operator to skip the red
  promote — so the "I wrote the failing test" event never fires on the
  honest path, leaving G-0252's ordering gate with no live promote to attach
  to. Correcting the seeding (ACs born at the pre-cycle `""` state) restores
  that event. This matches the model G-0286 (addressed) already ratified and
  the check layer already enforces; G-0286 fixed the check half and left the
  seeder untouched.
- **G-0440** — `aiwfx-plan-milestones` merges planning to main without ever
  calling `aiwf add ac` — that happens inside `aiwfx-start-milestone`'s
  preflight instead, one FSM stage after the milestone is already visible on
  main. The result: draft milestones sit on main with zero or unpopulated
  AC entities, invisible to any reader without the epic's worktree checked
  out.

This builds on the already-shipped `G-0216`/`D-0039` AC-completeness guard
family (`internal/check/acs.go`), which established the
block-at-transition/warn-at-rest pattern this epic's mechanisms extend one
FSM stage earlier (G-0440) and apply to a new dimension — ordering, not
just presence (G-0252). The three gaps are dependency-ordered: G-0441
(seeding correctness) enables G-0252 (the ordering gate on the now-live
`"" -> red` promote); G-0440 (plan-time AC content) is independent of both.
Full mechanism and rationale in D-0047.

## Scope

### In scope

- **Seeding correctness (G-0441):** `aiwf add ac` seeds `tdd: required` ACs
  at the pre-cycle `""` state, not `red`, so the `"" -> red` promote becomes
  a live event. Sweep the two born-at-red consequences: reverse
  `wf-tdd-cycle`'s "skip the red promote" guidance so the red promote is a
  live, mandatory step, and reconcile the `--tests`-at-`add` flag
  (`internal/verb/ac.go:106-108`).
- **Red-first ordering gate (G-0252):** a working-tree diff-shape check on
  the AC's TDD-phase promotes (`internal/verb/ac.go`), attached to the live
  `"" -> red` promote. `--phase red` refuses when the working-tree diff
  against HEAD touches any non-test path; `--phase green` refuses unless a
  non-test path is dirty now (a stateless check on the current diff — no
  red-time snapshot is kept or needed; ordering comes from the *pair* of
  gates). Test-path classification is a glob predicate over a **new config
  surface** — the `areamatch` matcher is reusable, but the test-path glob
  set is new (the areas `paths:` config maps source to workstreams, not
  test-vs-source).
- **Plan-time AC content (G-0440):** moving `aiwf add ac` and AC-body
  content-filling from `aiwfx-start-milestone`'s preflight into
  `aiwfx-plan-milestones`, before its merge-to-main step; plus a new
  warning-severity check-time finding (extending `internal/check/acs.go`
  alongside `milestoneDoneIncompleteACs`) surfacing a `draft` milestone with
  zero ACs or empty AC bodies.
- Updating `wf-tdd-cycle` and `aiwfx-plan-milestones`/`aiwfx-start-milestone`
  skill text to match the new mechanics (each embedded-rituals `SKILL.md`
  edit lands with its referencing structural test under `internal/policies/`
  per the skill-edit backstop).

### Out of scope

- Running the test suite at promote-time to verify fail-then-pass (rejected
  in D-0047 on cost/language-coupling grounds, the same objection as
  D-0038).
- A `--evidence`-style flag or an `aiwf-red-commit` SHA trailer (rejected in
  D-0047 — self-reported claims with an "existence not relevance" gap).
- Any change to `wf-vacuity` or `wf-review-code` — the semantic "does this
  test actually matter" judgment stays there by design (D-0047).
- Blocking (as opposed to warning) a `draft` milestone with incomplete ACs —
  draft is a legitimate mid-planning state (D-0047, mirroring D-0039's
  `done`-side reasoning).

## Constraints

- `red` must mean "a failing test exists" — the gate attaches to the live
  `"" -> red` promote, and the seeding fix (G-0441) is a hard prerequisite:
  the ordering gate is meaningless until an AC can actually reach `red` via a
  live event rather than being born there.
- The diff-shape check must add zero friction to an honest cycle — it
  validates existing working-tree state, no new commit or trailer.
- Must remain stack-agnostic — test-path classification is glob-based, not
  toolchain-coupled (no `go test -list` equivalent).
- The AC-content-timing fix must not turn `aiwfx-plan-milestones` into a
  hard block — `draft` stays warn-only per D-0039's precedent.

## Success criteria

- [ ] `aiwf add ac` against a `tdd: required` milestone leaves the new AC at
      the pre-cycle `""` state (not `red`); the `"" -> red` promote is a live
      event an operator runs once the failing test exists.
- [ ] `aiwf promote M-NNN/AC-N --phase red` refuses when a non-test path is
      already dirty in the working tree; `--phase green` refuses when no
      non-test path is dirty.
- [ ] A milestone drafted via `aiwfx-plan-milestones` has its AC entities
      (`acs[]` + `### AC-N` bodies) created and populated before the
      ritual's merge-to-main step, not deferred to `aiwfx-start-milestone`.
- [ ] `aiwf check`/`aiwf status` surface a warning for any non-archived
      `draft` milestone with zero ACs or an empty AC body.
- [ ] G-0441, G-0252, and G-0440 are all promoted to `addressed`.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Exact test-path glob convention (per-language default vs. per-milestone declared) | yes | settled at milestone-planning time |
| Does the new warning finding need its own grandfather/archive-scoping, or reuse `entity.IsArchivedPath` like its siblings | no | almost certainly reuse, per D-0039 precedent; confirm at milestone-planning |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Test-path glob misclassifies a legitimate file (e.g. a shared fixture under a non-test-looking path), causing a false-positive refusal | medium | `--force --reason` override (human-only, sovereign), same as every other FSM guard; glob configurable per project |
| The verb's own milestone-file write dirties a non-test path (`work/**`) during the promote, tripping the red gate against itself | medium | pin the inspected universe at milestone-planning (likely source paths, excluding the verb's own entity write and `work/**`/`docs/**` planning files); lock the boundary with a docs-dirty fixture test |
| Existing ACs already born at `red` under `tdd: required` after the seeding change | low | no backfill needed — `red` stays a valid present phase; only new `aiwf add ac` calls change behaviour, and the check layer already tolerates both absent and `red` |
| An in-flight `tdd: required` milestone mid-cycle when this lands | low | confirm at milestone-planning time whether any are currently mid-cycle (today: zero, per the current tree) |

## Milestones

Execution order: M-0274 → M-0275 → M-0276. Only M-0276's edge is a hard
dependency; M-0275 is independent and sequenced second by soft preference
(risk ladder: safe foundation first, riskiest gate last).

- `M-0274` — Seed `tdd: required` ACs pre-cycle so `red` is a live event
  (closes G-0441) · depends on: —
- `M-0275` — Create AC content at plan time; warn on incomplete `draft`
  milestones (closes G-0440) · depends on: —
- `M-0276` — Gate red-first ordering via a working-tree diff-shape check
  (closes G-0252) · depends on: `M-0274`

## References

- D-0047 — Contract-first AC timing and red-first ordering enforcement
- G-0441 — aiwf add ac seeds tdd:required ACs at red before any test exists
  (the seeding-correctness prerequisite)
- G-0252 — wf-tdd-cycle red-first ordering unguarded for consumer
  tdd:required AC cycles
- G-0440 — AC entities not created until start-milestone; milestones land
  bare on main
- G-0286 — the accepted decision that `red` means "a failing test exists"
  (check-layer half of the seeding correction)
- G-0216 / D-0039 — the AC-completeness guard precedent this epic extends
