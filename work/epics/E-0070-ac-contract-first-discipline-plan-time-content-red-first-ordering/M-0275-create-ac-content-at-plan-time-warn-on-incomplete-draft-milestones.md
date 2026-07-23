---
id: M-0275
title: Create AC content at plan time; warn on incomplete draft milestones
status: in_progress
parent: E-0070
tdd: required
acs:
    - id: AC-1
      title: Draft milestone with zero ACs raises a warning-severity finding
      status: met
      tdd_phase: done
    - id: AC-2
      title: Draft milestone with an empty AC body raises the finding
      status: met
      tdd_phase: done
    - id: AC-3
      title: The draft-AC finding is archive-scoped (silent on archived milestones)
      status: open
    - id: AC-4
      title: aiwfx-plan-milestones adds and body-fills ACs before its merge-to-main step
      status: open
    - id: AC-5
      title: aiwfx-start-milestone preflight reframes ACs as expected-to-pre-exist
      status: open
---

# M-0275 — Create AC content at plan time; warn on incomplete draft milestones

## Goal

Close G-0440: move AC-entity creation and body-filling into
`aiwfx-plan-milestones` (before its merge-to-main step) so milestones don't
land on main with empty or missing ACs, and add a warning-severity check-time
finding that surfaces a `draft` milestone with zero ACs or empty AC bodies.

## Context

`aiwfx-plan-milestones` merges planning to main without ever calling
`aiwf add ac` — that happens later, inside `aiwfx-start-milestone`'s preflight,
one FSM stage after the milestone is already visible on main. A reader without
the epic's worktree checked out then can't see what the milestone requires.
G-0216/D-0039 already built the "contract before code" guard, but it fires at
`draft -> in_progress` — one stage after this visibility gap. This milestone
extends that family one stage earlier, as a warning (not a block), because
`draft` is a legitimate mid-planning state. Full rationale in D-0047.

## Acceptance criteria

### AC-1 — Draft milestone with zero ACs raises a warning-severity finding

A new warning-severity check-time finding fires for a non-archived `draft`
milestone whose `acs[]` is empty. Warning, not error — `draft` is a legitimate
mid-planning state, so the finding surfaces the missing-contract gap without
blocking the milestone from resting in `draft`.

Mechanical evidence: a check-level test builds a non-archived `draft` milestone
with zero AC entities, runs the check, and asserts the new finding fires at
warning severity (and does not fire for a `draft` milestone that has ACs).

### AC-2 — Draft milestone with an empty AC body raises the finding

The finding (or a paired subcode) fires for a non-archived `draft` milestone
with any AC whose body subsection carries no non-heading prose — the same
empty-body condition `acs-empty-body` guards at `in_progress` (G-0216/D-0039),
surfaced one FSM stage earlier at `draft` as a warning.

Mechanical evidence: a check-level test builds a non-archived `draft` milestone
with an AC whose `### AC-N` body is empty, runs the check, and asserts the
finding fires.

### AC-3 — The draft-AC finding is archive-scoped (silent on archived milestones)

The finding is archive-scoped via the existing `entity.IsArchivedPath` guard:
an archived `draft` milestone (zero ACs or an empty AC body) does not fire it,
matching every sibling shape/health rule in `internal/check`.

Mechanical evidence: a check-level test places the same zero-AC / empty-body
`draft` milestone under an archive path and asserts the finding stays silent.

### AC-4 — aiwfx-plan-milestones adds and body-fills ACs before its merge-to-main step

The `aiwfx-plan-milestones` ritual gains a step that calls `aiwf add ac` and
fills each AC's body *before* its merge-to-main step, so a milestone never
lands on main with missing or empty ACs — closing the visibility gap G-0440
names.

Mechanical evidence: a structural policy test asserts the embedded
`aiwfx-plan-milestones` skill drives `aiwf add ac` + body-fill and that this
step precedes the merge-to-main step (the skill-edit structural-test backstop).

### AC-5 — aiwfx-start-milestone preflight reframes ACs as expected-to-pre-exist

The `aiwfx-start-milestone` preflight reframes ACs as expected to already exist
(added at plan time), retaining the "add them now" step as a recovery fallback
for hand-written specs rather than the default path.

Mechanical evidence: a structural policy test asserts the embedded
`aiwfx-start-milestone` preflight names ACs as expected-to-pre-exist and keeps
the fallback wording.

## Constraints

- Warn, never block, at `draft` — mirrors D-0039's block-at-transition /
  warn-at-rest split; no change to the existing `draft -> in_progress` block.
- The new finding reuses the file's existing archive-scoping convention — no
  new grandfather/timestamp mechanism.

## Design notes

- Implements D-0047 point 2; closes G-0440.
- Extends the `internal/check/acs.go` finding family alongside
  `milestoneDoneIncompleteACs`.

## Surfaces touched

- `internal/check/acs.go` (new warning finding)
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-milestones/SKILL.md`
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md`
- `internal/policies/` (structural tests for the skill edits)

## Out of scope

- The seeding fix (M-0274) and the ordering gate (M-0276).
- Blocking (as opposed to warning) an incomplete `draft` milestone.

## Dependencies

- None (independent of M-0274 and M-0276). Sequenced after M-0274 by soft
  preference so plan-time AC creation, once it is the practice, produces
  ACs born correctly at `""`.

## References

- G-0440 — the gap this milestone closes.
- D-0047 — Contract-first AC timing and red-first ordering enforcement.
- G-0216 / D-0039 — the AC-completeness guard precedent this extends.

## Work log

### AC-1 — Draft milestone with zero ACs raises a warning-severity finding

Added `milestoneDraftIncompleteACs` (finding `milestone-draft-incomplete-acs`,
subcode `zero-acs`, warning severity) in `internal/check/acs.go`, wired into
`check.Run` after `milestoneDoneIncompleteACs`. It fires for a non-archived
`draft` milestone whose `acs[]` is empty and stays silent once the milestone has
ACs; archive-scoped via `entity.IsArchivedPath`. The code carries a hint
(`internal/check/hint.go`) and an `aiwf-check` SKILL.md doc row, satisfying the
finding-code discoverability policies. Pinned by
`TestCheckRun_DraftMilestoneZeroACsWarns` (`internal/check/`), whose three
assertions — fires on a zero-AC draft, silent on a draft-with-ACs, silent on an
archived zero-AC draft — walk the `check.Run` aggregate. A follow-on commit
reconciled the finding's blast radius: the clean and verb-projection fixtures
gained a genuine AC, the cmd/aiwf goldens
(`internal/cli/integration/testdata/m0089/`) and the check-summary
code-set/order/footer record the new output, and three stress scenarios
(force-override-durability, concurrent-move, verb-sequence) added it to their
expected-warnings baselines as a documented setup side effect. · commit 2dc06b98
(blast-radius reconcile d8de2c99)
