---
id: E-0068
title: Mechanical AC/milestone-completeness guards
status: active
---

# E-0068 — Mechanical AC/milestone-completeness guards

## Goal

Close three places where the kernel currently depends on operator vigilance instead of a mechanical chokepoint for AC/milestone completeness discipline — a milestone starting with an empty AC body, a milestone starting or finishing with zero ACs at all, and an over-strict `tdd_phase` requirement — so the AC-evidence discipline holds without relying on a human or an LLM remembering the rules.

## Context

`internal/check/acs.go` already carries five rules that shape and audit a milestone's `acs[]` list (`acsShape`, `acsTitleProse`, `acsTDDAudit`, `milestoneDoneIncompleteACs`, `milestoneCancelledIncompleteACs`, `acsBodyCoherence`), but three real holes remain, each traced and confirmed against the current codebase:

- **G-0216** — a milestone can go `draft → in_progress` while its AC bodies are empty prose. The contract-first TDD discipline (write the AC's contract before the test) depends on the human remembering to fill it in first. This already bit M-0159 and M-0160, whose AC bodies were backfilled post-hoc at wrap — the exact "AC gaming" failure mode the discipline exists to prevent.
- **G-0334** — a milestone can traverse `draft → in_progress → done` with zero acceptance criteria, tripping no finding. The AC-evidence discipline is vacuous for a zero-AC milestone; there is no milestone-level sibling to the epic-level `epic-active-no-drafted-milestones` guard.
- **G-0286** — under `tdd: required`, `acs-shape/tdd-phase` forces every AC to carry a `tdd_phase` the instant it exists, which is stricter than what the design actually commits to (CLAUDE.md #8: `AC met` requires `tdd_phase: done`, not "every AC is phase-tracked from creation"). Strengthening a milestone `advisory → required` currently reddens the tree for every pre-existing AC.

The design forks in G-0216 and G-0334 were reviewed together (they share the same file and the same "operator vigilance vs. mechanical guarantee" theme) and settled in [D-0039](../../decisions/D-0039-ac-completeness-guards-block-empty-start-warn-at-done-archive-scoped-check.md) (`status: accepted`), which is the authoritative source for the guard behavior this epic implements. G-0286 needed no design decision — it is a scope-relaxation bug fix, independently specified by its own gap.

## Scope

### In scope

- Verb-time refusal on `draft → in_progress` when a milestone has zero ACs, with `--force --reason "..."` override (D-0039 point 1).
- Verb-time refusal on `draft → in_progress` when any AC's body subsection is empty, same override shape (G-0216).
- A new warning-severity check-time finding extending the existing `milestone-done-incomplete-acs` pattern to also fire when a `done` milestone has an empty AC set, not just open ACs (D-0039 point 2).
- A new check-time finding for empty AC bodies on `in_progress`/`done` milestones, scoped with the existing `entity.IsArchivedPath` archive guard already used by every sibling rule in `internal/check/acs.go` — no new grandfather/timestamp mechanism (D-0039 point 3).
- Relaxing `acs-shape/tdd-phase` so an absent `tdd_phase` is legal until an AC reaches `met`, at which point the existing `acs-tdd-audit` rule ("`met` requires `tdd_phase: done`") still applies unchanged (G-0286).

### Out of scope

- **G-0252** (red-first TDD ordering enforcement — confirming a failing test *preceded* the implementation, not just that it exists). Already deferred by `docs/initiatives/tdd-cycle-subagent-boundaries.md` as a meaningfully heavier lift (candidate mechanisms there require either running the test suite at promote time or a new ordering-pinning trailer); not reopened by this epic.
- Any change to the `wf-tdd-cycle` skill's advisory guidance. This epic ships kernel chokepoints; ritual-content changes are a separate concern.

## Constraints

- The new verb-time refusals follow the FSM's existing `--force --reason "..."` override shape; no new override mechanism is introduced.
- The new check-time findings follow the existing archive-scoping convention (`entity.IsArchivedPath`) already used by every sibling rule in `internal/check/acs.go`; no new grandfather/timestamp machinery.
- Per D-0039, the `done`-transition guard for a zero-AC milestone is check-time and warning-severity only — it must not become a second verb-time hard block.

## Success criteria

- [ ] Every guard and relaxation listed under *In scope* above is implemented, tested per the branch-coverage discipline, and observable in `aiwf check` / `aiwf promote` output against a fixture tree.
- [ ] G-0216, G-0286, and G-0334 are each promoted to `addressed` with the implementing milestone/commit cited.
- [ ] `aiwf check` against this repo's own tree remains clean (no new findings introduced by the guards themselves against current state).

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Exact finding codes/subcodes for the two new check-time findings (the zero-AC-at-`done` warning extending `milestone-done-incomplete-acs`, and whether the empty-AC-body finding extends `entity-body-empty` or is a new code) | no | Decided during milestone implementation; naming doesn't affect the guard behavior D-0039 already settled. |

## Milestones

- [M-0267](M-0267-relax-acs-shape-tdd-phase-to-allow-absent-phase-until-ac-met.md) — Relax `acs-shape/tdd-phase` so an absent phase is legal until `met` (G-0286) · depends on: —
- [M-0268](M-0268-ac-completeness-guards-zero-ac-and-empty-body-promote-refusals.md) — AC-completeness guards: zero-AC and empty-AC-body refusals plus their check-time findings (G-0216 + G-0334, per D-0039) · depends on: —

## References

- [D-0039](../../decisions/D-0039-ac-completeness-guards-block-empty-start-warn-at-done-archive-scoped-check.md) — the accepted decision settling the guard-severity forks this epic implements.
- [G-0216](../../gaps/G-0216-empty-ac-body-blocks-milestone-draft-to-in-progress-promote.md), [G-0286](../../gaps/G-0286-acs-shape-tdd-phase-over-demands-a-phase-on-every-ac-under-tdd-required.md), [G-0334](../../gaps/G-0334-milestone-can-start-and-finish-with-zero-acceptance-criteria-no-guard.md) — the source gaps.
- `docs/initiatives/tdd-cycle-subagent-boundaries.md` — the wider initiative this cluster was reviewed alongside; names G-0252 as deferred and out of scope here.
