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
      status: met
      tdd_phase: done
    - id: AC-4
      title: aiwfx-plan-milestones adds and body-fills ACs before its merge-to-main step
      status: met
      tdd_phase: done
    - id: AC-5
      title: aiwfx-start-milestone preflight reframes ACs as expected-to-pre-exist
      status: met
      tdd_phase: done
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

### AC-2 — Draft milestone with an empty AC body raises the finding

Extended `milestoneDraftIncompleteACs` so a draft milestone that has ACs no
longer returns early: it reads the file, parses the AC body sections
(`entity.ParseACSections`), and fires subcode `empty-body` (warning, keyed to
the composite AC id) for any non-cancelled AC whose `### AC-N` body carries no
non-heading prose — the draft-rung, warning-severity mirror of
`acsEmptyBodyOnStart`'s in_progress/done error, sharing that rule's
missing-heading / cancelled-AC / empty-id carve-outs. A subcode-specific hint
points at `aiwf edit-body` (the AC exists; only its body is missing), and the
`aiwf-check` SKILL.md row now documents both subcodes. Pinned by
`TestCheckRun_DraftMilestoneEmptyACBodyWarns` (fires + warning severity +
composite id) and `TestCheckRun_DraftMilestoneEmptyACBody_CarveOuts` (the three
skip branches); the fires path was confirmed non-vacuous by the live red→green
transition and a targeted severity mutation. No blast radius outside
`internal/check` — no fixture, golden, or stress baseline changed, since
empty-body fires only on a draft milestone whose ACs have empty bodies, a shape
none of them carry. · commit 1b28bb10

### AC-3 — The draft-AC finding is archive-scoped (silent on archived milestones)

Pinned by `TestCheckRun_DraftMilestoneIncompleteACs_ArchiveScoped`
(`internal/check/`): a table over both subcodes asserts each fires on an
active-tree draft milestone (the non-vacuity guard) yet stays silent on the same
shape under an archive path, exercising the shared `entity.IsArchivedPath` guard
for `zero-acs` and `empty-body` together. No production change — the guard
predates this AC (it has capped the rule since AC-1); AC-3 promotes
archive-scoping from an incidental sub-assertion inside the AC-1/AC-2 tests to a
first-class named property. RED was genuine: with the guard temporarily removed,
both archived subcodes fired and both silent-assertions failed; restoring it (a
byte-exact revert, empty acs.go diff) returned the test to green. · commit
191e1120

### AC-4 — aiwfx-plan-milestones adds and body-fills ACs before its merge-to-main step

Folded AC-entity creation into the embedded `aiwfx-plan-milestones` step 5
(authoring the milestone spec): the "Acceptance criteria" template bullet now
redirects to a dedicated block that runs `aiwf add ac` per criterion and fills
each scaffolded `### AC-N` body via `aiwf edit-body`, carrying the rationale that
doing this before the merge-to-main step is what keeps a milestone off main with
zero ACs or empty AC bodies (the `milestone-draft-incomplete-acs` gap AC-1/AC-2
surface), and that `aiwfx-start-milestone`'s preflight then expects the ACs to
pre-exist. Folded into step 5 rather than a new numbered step to avoid
renumbering the two "step N" cross-references. Pinned by
`TestAiwfxPlanMilestones_CreatesACsBeforeMerge_M0275` (`internal/policies/`),
which reads the embedded skill bytes and asserts — content-driven — that `aiwf
add ac` plus a co-located `aiwf edit-body` appear in `## Workflow` and precede
the merge-to-main step; the file's existing path constant satisfies the
skill-edit structural-test backstop. RED was genuine — the assertion failed
against the unedited skill (no `aiwf add ac` in the workflow). · commit 2c946eb0

### AC-5 — aiwfx-start-milestone preflight reframes ACs as expected-to-pre-exist

Reframed the embedded `aiwfx-start-milestone` preflight's AC bullet: instead of
"confirm the spec has its ACs landed … if hand-written, add them now," it now
states ACs are expected to already exist — created and body-filled at plan time
by `aiwfx-plan-milestones` (AC-4) — demoting on-the-spot `aiwf add ac` to an
explicit recovery fallback for a hand-written spec, and tying the empty-spec case
to the `milestone-draft-incomplete-acs` warning. Pinned by
`TestAiwfxStartMilestone_PreflightExpectsACsPreExist_M0275`
(`internal/policies/`), heading-scoped to the preflight subsection: it asserts
the plan-time reframe, the fallback framing, and the retained `aiwf add ac`
recovery command. RED was genuine — the preflight carried neither "plan time"
nor "fallback" framing before the edit. · commit 1ef79d50
