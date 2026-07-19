---
id: M-0268
title: 'AC-completeness guards: zero-AC and empty-body promote refusals'
status: draft
parent: E-0068
tdd: required
acs:
    - id: AC-1
      title: Zero-AC milestone refused at draft to in_progress promote
      status: open
      tdd_phase: red
    - id: AC-2
      title: Empty AC body refused at draft to in_progress promote
      status: open
      tdd_phase: red
    - id: AC-3
      title: Zero-AC done milestone surfaces a warning finding
      status: open
      tdd_phase: red
    - id: AC-4
      title: Empty AC body surfaces an error finding, archive-scoped
      status: open
      tdd_phase: red
---

# M-0268 — AC-completeness guards: zero-AC and empty-body promote refusals

## Goal

Make the two remaining "no real contract yet" states — a milestone with zero ACs, and a milestone with an AC whose body is empty prose — refuse to start `in_progress`, per the decisions locked in D-0039, instead of depending on the operator to remember to add or fill in ACs first.

## Context

`internal/verb/promote.go`'s `Promote` function already carries two structurally-similar guards: `EpicPromoteNonTerminalChildrenError` (an epic can't reach a terminal status while it owns a non-terminal milestone) and `MilestonePromoteNonTerminalACsError` (a milestone can't be cancelled while it has an open AC). Both explicitly run "unconditionally, even under `--force`" per their own code comments — `--force` in this codebase relaxes FSM-transition *legality*, not these structural preconditions.

This milestone's two new guards are a different kind of precondition and must **not** copy that unconditional pattern — see Design notes.

`internal/check/acs.go` already carries five rules following the `entity.IsArchivedPath(e.Path)` archive-scoping convention (`acsShape`, `acsTDDAudit`, `acsBodyCoherence`, `milestoneDoneIncompleteACs`, `milestoneCancelledIncompleteACs`); the two new check-time findings in this milestone extend that same convention, per D-0039 point 3.

## Acceptance criteria

### AC-1 — Zero-AC milestone refused at draft to in_progress promote

`aiwf promote M-NNN in_progress` on a milestone whose `acs[]` is empty exits non-zero, naming the milestone and pointing at adding an AC or `--force --reason`. `aiwf promote M-NNN in_progress --force --reason "..."` succeeds despite zero ACs. Per D-0039 point 1.

### AC-2 — Empty AC body refused at draft to in_progress promote

`aiwf promote M-NNN in_progress` on a milestone where at least one AC's body subsection (the prose between its `### AC-<N>` heading and the next heading or EOF) contains no non-heading prose exits non-zero, naming the specific AC. Same `--force --reason` override. Per G-0216.

### AC-3 — Zero-AC done milestone surfaces a warning finding

`aiwf check` emits a new warning-severity finding — extending the existing `milestone-done-incomplete-acs` pattern in `internal/check/acs.go` — when a non-archived milestone has `status: done` and an empty `acs[]`. This is check-time only: `aiwf promote M-NNN done` itself is **not** refused for a zero-AC milestone; there is no verb-time guard at this transition. Per D-0039 point 2.

### AC-4 — Empty AC body surfaces an error finding, archive-scoped

`aiwf check` emits a new error-severity finding when a non-archived milestone is `in_progress` or `done` and any AC's body subsection is empty. Scoped with `entity.IsArchivedPath(e.Path)`, the same guard as its five siblings in `acs.go` — an archived milestone never fires this finding regardless of body state, so this stays forward-only without any new grandfather or timestamp mechanism. Per G-0216 and D-0039 point 3.

## Constraints

- AC-1 and AC-2's refusals apply only to the `draft → in_progress` transition — they must not affect any other legal milestone transition.
- AC-3's finding is warning severity and check-time only; it must never become a verb-time refusal at `done` (D-0039 explicitly rejects a second hard block there).
- AC-4's finding is error severity and archive-scoped; it fires for `in_progress` and `done` alike, but never for an archived milestone.

## Design notes

- **`--force` bypasses AC-1 and AC-2, deliberately diverging from the adjacent unconditional-guard pattern.** `MilestonePromoteNonTerminalACsError` and `EpicPromoteNonTerminalChildrenError` (both in the same `Promote` function, just above where these two guards land) run even under `--force`, because they protect a genuine consistency invariant: a *terminal* status must never be reached while a child/AC is still non-terminal. AC-1 and AC-2 guard something different — whether real work has a contract *before it starts* — and D-0039 point 2 explicitly permits a zero-AC milestone to reach `done` (with only a warning), so "permanently AC-less" is itself a legitimate end state, not an inconsistency force would be papering over. The correct precedent to follow is the `if !force { ... }` pattern used by the resolver-requirement checks earlier in `Promote` (e.g. the `gap-addressed-has-resolver` / `adr-supersession-mutual` overrides at promote.go:465/469) — a sovereign, human-only, reason-carrying override of a soft precondition — not the unconditional structural guards. Get this pattern right at implementation time; copying the nearby non-terminal-ACs guard by proximity would silently make the milestone un-force-able, contradicting D-0039.
- AC-3 and AC-4 both key their archive scoping off `entity.IsArchivedPath(e.Path)`, identical to the five existing rules in the same file — no new helper.

## Surfaces touched

- `internal/verb/promote.go` — the two new verb-time refusals (AC-1, AC-2), following the `if !force { ... }` resolver-requirement pattern, not the unconditional structural-guard pattern.
- `internal/check/acs.go` — the two new check-time findings (AC-3 extends `milestoneDoneIncompleteACs`; AC-4 is new, mirroring the file's existing rule shape).

## Out of scope

- G-0252 (red-first TDD ordering enforcement) — out of scope for the whole epic (see E-0068's spec), not just this milestone.
- Any change to the existing `MilestonePromoteNonTerminalACsError` / `EpicPromoteNonTerminalChildrenError` guards — they stay unconditional as-is; this milestone only adds new, separately-gated guards.

## Dependencies

- None on M-0267 (different code paths in the same file; independently shippable in either order).
- [D-0039](../../decisions/D-0039-ac-completeness-guards-block-empty-start-warn-at-done-archive-scoped-check.md) — accepted, the authoritative source for AC-1's and AC-3's behavior.

## References

- [D-0039](../../decisions/D-0039-ac-completeness-guards-block-empty-start-warn-at-done-archive-scoped-check.md)
- [G-0216](../../gaps/G-0216-empty-ac-body-blocks-milestone-draft-to-in-progress-promote.md), [G-0334](../../gaps/G-0334-milestone-can-start-and-finish-with-zero-acceptance-criteria-no-guard.md)
