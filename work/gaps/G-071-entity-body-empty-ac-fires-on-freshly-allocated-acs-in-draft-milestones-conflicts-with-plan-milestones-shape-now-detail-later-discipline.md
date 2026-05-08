---
id: G-071
title: entity-body-empty/ac fires on freshly-allocated ACs in draft milestones; conflicts with plan-milestones 'shape now, detail later' discipline
status: open
discovered_in: E-20
---
## What's missing

`internal/check/entity_body.go`'s `entity-body-empty/ac` rule fires on every AC whose body section under `### AC-N` is whitespace-only, regardless of the parent milestone's status or the AC's TDD phase. The check is shape-aware (heading must be present, in canonical form) but lifecycle-blind: an AC allocated 30 seconds ago via `aiwf add ac` produces a warning the moment the next `aiwf check` runs.

Concretely, allocating M-072..M-074 for E-20 via `aiwfx-plan-milestones` produced 24 `entity-body-empty/ac` warnings on a freshly-rebuilt binary — one per allocated AC. The plan-milestones skill explicitly mandates this state: *"Does not draft individual milestone specs in deep detail — that happens just-in-time when each milestone is started."* Detail under each `### AC-N` is supposed to fill at `aiwfx-start-milestone`, not at allocation. The skill's discipline and the kernel's check rule are at odds.

## Why it matters

The rule was scoped at M-066/AC-1 to catch *shipped* empty bodies — entities promoted to terminal status without prose. The current implementation generalizes that intent to every state, including `draft`, which catches the planning-phase backlog as noise. Three downstream costs:

1. **Plan-milestones output is dirty by design.** Every epic broken into N milestones with M ACs each surfaces N×M warnings on the next `aiwf check`. For E-20 that's 24; a 5-milestone epic with 5 ACs each yields 25. The right baseline after planning is "tree clean except provenance"; the rule makes that unattainable without filling stub prose.
2. **The "fix" defeats planning hygiene.** Stubbing one-line prose under each AC heading at allocation time produces exactly the rotting AC bodies the plan-milestones anti-pattern warns against (*"AC definitions written 6 weeks before the work starts are usually wrong"*). Operators are nudged toward writing throwaway placeholder prose to silence warnings.
3. **Strict mode amplifies the cost.** `tdd.strict` escalates `entity-body-empty` to error via `ApplyTDDStrict` in `entity_body.go`. On a strict-mode repo, planning a multi-milestone epic would block `aiwf check` until every AC body is filled — at exactly the moment the operator should be deferring detail.

Two complementary fix shapes worth considering:

- **Status gating.** Fire `entity-body-empty/ac` only when the parent milestone's `status` is `in_progress` or `done`. A `draft` milestone is pre-implementation; empty AC bodies are the expected planning output. Coarser but doesn't require TDD configuration.
- **Phase gating.** When the parent milestone is `tdd: required` and an AC has `tdd_phase: red` with `status: open`, treat the empty AC body as expected. Fire once the AC is `tdd_phase: green` (implementation has begun) or the milestone is `in_progress`. Phase data is already in the AC's frontmatter; the rule just needs to consult it.

Status gating is the simpler primitive; phase gating is the more precise. Both could coexist (status-gated by default, phase-aware when `tdd: required`).

Surfaced during E-20 planning when `aiwfx-plan-milestones` produced 24 warnings against the just-allocated M-072/M-073/M-074 — explicitly the shape the skill is supposed to leave the tree in. Documented as a follow-up rather than scoped into E-20 because the fix is a kernel-discipline concern in `internal/check/entity_body.go`, not in the verb being added.
