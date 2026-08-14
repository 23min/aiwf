---
id: E-0085
title: Measure the spec before the code; record what the measurement changed
status: active
---

## Goal

Give the step before implementation a method and an artifact: an entity's
factual claims are measured by running commands before any code is written, and
what the measurement changed is left on the record.

## Context

`aiwfx-start-milestone`'s first step asks the reader to confirm every acceptance
criterion is concrete and testable and supplies no method for deciding that. Its
other bullets check that the baseline builds and the tests pass — real, but
orthogonal to whether the spec is right. `wf-patch` step 1 has the same shape:
state the goal in your own words, and if you can't, the change isn't ready.

Measured across three milestones, the specification was wrong more often than
the implementation. M-0307 was cancelled at preflight — the defect was real but
worse than stated, the surface count three times what the spec named, and one
criterion could not be written without asserting that the guidance says what the
test says it says. Its premise came from G-0541, whose own title says the path
"resolves for two of six kinds"; measurement found it resolved for none. The
false claim originated in the gap, which is why a patch closing a gap carries
the same exposure as a milestone.

Review does not reach this class. M-0306 had six independent review passes, all
returning request-changes on the implementation; none questioned the spec,
because a reviewer checks work against a specification rather than a
specification against reality.

E-0081 set the disposal precedent this epic inherits. Eleven surfaces stated one
fact and several were wrong; the answer was a single `entity.RequiredSections`
that the rest derive from, not eleven corrections. M-0307's own surviving commit
took the same shape — a ban with both sides derived rather than an assertion.

## Scope

- A stack-agnostic ritual under the `wf-rituals` plugin: measure every factual
  claim by running commands, challenge each criterion for the failure it
  prevents and whether its letter can be satisfied vacuously, sweep related
  prose against current trunk.
- The recorded-outcome contract: the pass terminates in an `aiwf edit-body <id>`
  commit on the entity whose claims were measured, or an explicitly recorded
  no-change.
- Wiring into `aiwfx-start-milestone` step 1 and `wf-patch` step 1.
- `aiwfx-plan-milestones` stops asserting it produces testable acceptance
  criteria, a property it has no method to deliver.

## Out of scope

- Any kernel change. No check rule, no finding code, no config field, no schema
  entry ships with this epic.
- The structural direction — requiring a criterion to state its own
  falsification condition and checking that mechanically. It is the other answer
  to the same question in `quality-signal-and-cadence.md`'s Q6, it stays
  untaken, and it stays cheaper wherever a criterion can be made to carry it.
- G-0530's pruning of milestone-spec sections. Same artifacts, different
  subject: that is spec size, this is spec truth.
- E-0084's body-section membership enforcement. That decides whether a required
  heading is present; this asks whether what is under it is true.

## Constraints

- **The kernel does not change.** The enforcement this epic ships is a record in
  git, not a rule. A check rule over that record is a later question, and it
  waits until use has shown what the record should contain.
- **The embedded snapshot is the authoring location.** `.claude/skills/` and
  `.claude/templates/` are materialized by `aiwf update`; edits land under
  `internal/skills/embedded-rituals/`.
- **Every `SKILL.md` edit lands with its referencing structural test**, per the
  repo's backstop policy.
- **The sweep reads current trunk through an affordance, not a reminder.** Run
  against a stale branch it missed the most consequential finding of the day and
  produced one confident false one; an instruction to remember is the shape
  CLAUDE.md rules out as a guarantee.
- **Sweep findings are hypotheses until a command settles one.** The sweep's
  headline claim was wrong and one measurement settled it.
- **A fact stated across several surfaces gets an owner, not a correction per
  copy** — E-0081's disposal, D-0054's ban on duplication with no re-derivation.

## Success criteria

- [ ] Starting a milestone, or opening a patch that closes a gap, runs a named
      ritual that measures the entity's factual claims before any code is
      written.
- [ ] The ritual terminates in either an `aiwf edit-body <id>` commit or an
      explicitly recorded no-change, so a pass that was skipped is visible in
      history rather than indistinguishable from one that found nothing.
- [ ] No shipped surface claims to produce testable acceptance criteria without
      a method behind the claim.
- [ ] Every ritual or skill file this epic adds or edits carries its referencing
      structural test.
- [ ] The kernel is unchanged: `aiwf check` gains no finding code and the config
      schema gains no field.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| What shape does a "no-change" record take — a commit trailer, a body line, or an empty `edit-body`? | yes | Settled in the first milestone, before the ritual ships; the answer decides whether a later check rule has anything to read. |
| Does the sweep's fresh-trunk checkout reuse `aiwf worktree add` or something lighter and read-only? | resolved | Neither. No checkout is needed at all — a ref can be fetched and searched in place. `aiwf worktree add` additionally requires a branch name and materializes rituals, both wrong for a read-only search. |
| Does the ritual belong at epic start too? | no | Left unanswered deliberately — no evidence at epic scale yet. Revisit once the milestone and patch seams have run enough times to say. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| The pass is prose, and nothing makes an assistant run it | med | The recorded outcome makes omission visible in git history. This epic stops short of a check rule on purpose: the rule that would close the hole needs to know what the record contains, and that is learned from use. |
| A recorded no-change is cheap to produce without doing the work | med | Accepted and stated. The record catches omission, not laziness. Closing that gap is a kernel question this epic explicitly defers rather than half-answers. |
| The pass costs more per milestone than the defects it prevents | med | The parts run in yield order: measuring and challenging are cheap and caught every defect in the motivating episode. The sweep is the expensive one and is bounded — names rather than concepts, one hop on a borrowed claim — which a replay over five recorded drift cases put at 11 to 69 files to read, against 171 to 320 for the unbounded forms. |
| The sweep reaches a narrower class than the epic assumed | med | Stated rather than mitigated. The same replay missed two of five: prose that drifted in an earlier refactor names symbols the current change does not touch, and a defect of shape rather than wording names no symbol at all. Both are check-shaped, and the ritual says so where a reader meets one instead of implying a clean sweep is an all-clear. |

## Milestones

- `M-0308` — the ritual ships: measure, challenge and sweep, with the record it
  terminates in settled first · depends on: —
- `M-0309` — the seams invoke it: `aiwfx-start-milestone` and `wf-patch` route
  to it, and `aiwfx-plan-milestones` drops the claim it cannot deliver ·
  depends on: `M-0308`

## References

- G-0583 — the gap this epic closes
- G-0541, M-0307 — the measured episode; the gap's own claim was the wrong one
- E-0081 — the disposal precedent: one owner, not N corrections
- D-0054 — the ban on a fact copied into prose nothing re-derives
- `docs/initiatives/quality-signal-and-cadence.md` — Q6, and the structural
  direction left untaken
