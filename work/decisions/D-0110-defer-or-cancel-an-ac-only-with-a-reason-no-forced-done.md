---
id: D-0110
title: Defer or cancel an AC only with a reason; no forced done
status: accepted
relates_to:
    - D-0109
---
> **Date:** 2026-11-18 · **Decided by:** human/peter

## Question

Two cases no record settles:

1. How does a milestone with an open AC reach `done`? The "Milestone-done implies AC progress"
   bullet in `docs/design/design-decisions.md`, the `milestone-done-incomplete-acs` hint in
   `internal/check/hint.go`, the `aiwf-promote` skill and the `aiwfx-wrap-milestone` ritual say
   `--force` takes it there while the standing check keeps reporting. `aiwf promote` refuses it:
   `Promote` in `internal/verb/promote.go` runs its projection check under `--force` as well, and
   the comment above `requireNonEmptyACBodiesAtMilestoneStart` there says force "relaxes
   FSM-transition legality and this specific verb-time guard, never check-time coherence".
2. Must an AC that leaves the milestone's contract say why? `aiwf promote <milestone>/AC-N
   deferred|cancelled` and `aiwf cancel <milestone>/AC-N` take `--reason` as optional. The
   `aiwfx-wrap-milestone` ritual asks for a note in the milestone body pointing a deferred AC at
   the milestone or gap that receives it, and nothing checks the note.

## Decision

- Finish a milestone that has an open AC by closing each open AC first: `met` with its
  evidence, `deferred` or `cancelled`. `--force` does not take a milestone to `done` while an AC
  is open.
- Require `--reason` on every move of an AC to `deferred` or `cancelled`, from `open` or from
  `met`, through `aiwf promote` or `aiwf cancel`. The reason lands in the commit body, as
  `--reason` does today.

## Reasoning

**No forced done.** An open AC on a `done` milestone is the error `milestone-done-incomplete-acs`
reports on every `aiwf check`, so a forced done would commit a state the kernel names an error.
Letting `--force` past this one finding would also make it relax a check, which the comment in
`promote.go` says force never does. D-0109's reasoning states that no verb leaves a finished
milestone with an open AC; a forced done would be one. The rule removes no route to an honest
state: deferring the open AC and then promoting the milestone, with no `--force`, took a scratch
milestone to `done` (both exit 0, with `aiwf` built from main at bd818e14d). The alternative was
a flag on the done promote that defers every open AC in the same commit. It lost because one
reason would cover several criteria, and a cut that should be `cancelled` would be recorded as
`deferred`.

**A reason to leave the contract.** `deferred` and `cancelled` each take a criterion off the
milestone's contract without claiming it succeeded (the comment on `acTransitions` in
`internal/entity/transition.go`), and the only record of why that a verb writes is the commit
body. Of the 81 commits reachable from ce4e6d89c that moved an AC to `deferred` or `cancelled`,
72 carried a reason in the body. The rule covers `met` as well as `open`, because withdrawing a
criterion already recorded as delivered needs its why at least as much as closing an undecided
one, and one rule is simpler to state and test than one with an exception. A check-time warning
instead of a refusal lost because the move would already be committed without its reason, and
nothing adds a reason to a commit once it has landed. A field naming where deferred work went
lost because it means nothing for a cancelled AC, and a reason can name the receiving milestone
or gap in prose.

## Consequences

- `promoteAC` and `cancelAC` in `internal/verb/ac.go` commit the move without a reason. Tests
  that defer or cancel an AC without one change with them.
- Every source that describes a forced done is now wrong. Among them is the
  `milestone-done-incomplete-acs` hint: M-0293/AC-2 appended to its `--force` remedy a caveat on
  what the override leaves standing, and `forceOfferingHintKeys` in
  `internal/check/hint_force_caveat_test.go` requires the hint to offer that remedy.
