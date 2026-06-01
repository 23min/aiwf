---
id: M-0105
title: aiwfx-start-milestone sequencing alignment
status: in_progress
parent: E-0030
depends_on:
    - M-0102
    - M-0103
tdd: required
acs:
    - id: AC-1
      title: Embedded snapshot reflects new step ordering
      status: open
      tdd_phase: done
    - id: AC-2
      title: Skill asserts tightened parent-epic-branch precondition
      status: open
      tdd_phase: red
    - id: AC-3
      title: Silent fallthrough to checkout -b epic/<slug> if missing removed
      status: open
      tdd_phase: red
    - id: AC-4
      title: Workflow headings structurally appear in new order
      status: open
      tdd_phase: red
    - id: AC-5
      title: Skill body names --force --reason override at appropriate step
      status: open
      tdd_phase: red
    - id: AC-6
      title: Milestone scope aiwf-branch trailer records milestone branch
      status: met
      tdd_phase: done
---
## Goal

Align `aiwfx-start-milestone`'s step order with M-0104's epic-side fix: `aiwf promote M-NNNN draft → in_progress` lands on the parent epic branch (which already exists at this point from `aiwfx-start-epic`), then — *if* the work is being delegated — `aiwf authorize M-NNNN --to ai/<id> --branch milestone/M-NNNN-<slug>` lands on the same parent epic branch, then the milestone work branch is cut off the parent. Tighten the "must be on parent epic branch" precondition so silent fallthrough to `git checkout -b epic/E-NNNN-<slug> if missing` is removed — missing parent epic branch is a hard precondition failure pointing the operator at `aiwfx-start-epic`.

## Context

M-0104 establishes the pattern for `aiwfx-start-epic`; this milestone applies the same shape one level down at the milestone-start ritual. [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md)'s symmetric rule for milestones: the promote-to-in_progress is a state-announcement that belongs on the parent epic branch (which already exists at this point), not on the milestone work branch (which hasn't been cut yet).

Today's embedded `aiwfx-start-milestone` step 2 *does* promote before branch setup — which matches ADR-0010's order — but step 3 contains a silent fallthrough (`git checkout -b epic/E-NNNN-<slug> origin/main # if missing`) that masks the precondition failure case. This milestone removes the fallthrough and adds the explicit "epic branch must exist; if it doesn't, run `aiwfx-start-epic` first" check.

Ritual content edits land at the canonical authoring location (per [ADR-0014](../../../docs/adr/ADR-0014-embed-and-materialize-rituals-distribution-retire-claude-marketplace.md) and [ADR-0016](../../../docs/adr/ADR-0016-retire-ai-workflow-rituals-upstream-channel-embedded-snapshot-canonical.md)):

- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md`

One edit in one commit. The upstream `ai-workflow-rituals` repo was archived under ADR-0016 — no cross-repo coordination.

## Pre-decided design

- **New step ordering** (matches the symmetric pattern from M-0104):
  1. Preflight (existing — includes "parent epic branch must exist and be currently checked out"; this is the tightened precondition).
  2. Delegation prompt (Q&A — promoted earlier so the operator's choice is known before the sovereign acts).
  3. `aiwf promote M-NNNN in_progress` on the parent epic branch (existing step 2's content, now explicitly named as "lands on parent epic branch").
  4. *(if delegating)* `aiwf authorize M-NNNN --to ai/<id> --branch milestone/M-NNNN-<slug>` on the parent epic branch. Same "future branch" refinement from M-0104's preflight applies: the named milestone branch doesn't yet exist; M-0103's preflight accepts it because the operator is on a recognized ritual shape (the parent epic branch) and `--branch` parses as a valid ref.
  5. Cut the milestone branch off the parent epic branch (`git checkout -b milestone/M-NNNN-<slug>`).
  6. Hand off to `wf-tdd-cycle` for each AC (existing).
- **Tightened precondition (step 1):** the current checkout must be the parent epic branch identified by `aiwf show M-NNNN`'s parent field. If the parent epic branch doesn't exist locally, the ritual stops and points at `aiwfx-start-epic E-NNNN`. The silent `git checkout -b epic/E-NNNN-<slug> origin/main # if missing` fallthrough is removed.
- **Scope inheritance:** the milestone's `aiwf authorize` opens a **new** scope independent of the epic's (the current kernel semantics — one scope per entity, no cross-entity coordination). The milestone scope's `aiwf-branch:` records the milestone branch; the epic scope's `aiwf-branch:` records the epic branch. M-0106's finding rule walks back to the nearest active scope on the entity in question (the milestone for milestone-entity commits, the epic for epic-entity commits) — no special cross-scope logic. (Conceptual framing per [ADR-0009](../../../docs/adr/ADR-0009-orchestration-substrate-vs-driver-split.md) substrate/driver split, but no mechanical dependency on ADR-0009 ratifying.)
- **Override path naming** in skill body: same shape as M-0104 — the skill body names `--force --reason "..."` at the relevant step so operators see it.

## Out of scope

- `aiwfx-start-epic` (M-0104, sibling).
- Kernel finding (M-0106).
- AC-level branch behavior — ACs ride on the milestone branch alongside test/code commits per ADR-0010; no separate AC-branch convention is in scope here.
- Spec-cell consolidation (the consolidation milestone).

## Dependencies

- **M-0102** — `--branch` flag and `internal/branchparse/` helpers.
- **M-0103** — preflight refuses dispatch without ritual branch context; the "future branch" refinement (added under M-0104) makes the `--branch milestone/M-NNNN-<slug>` call on a yet-uncut branch acceptable.

## Acceptance criteria

<!-- Drafted at `aiwfx-start-milestone M-0105` time. AC seed set:
1. The embedded snapshot at `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md` reflects the new step ordering: tightened preflight → delegation prompt → promote on parent epic branch → authorize on parent epic branch (if delegating) → cut milestone branch → wf-tdd-cycle hand-off.
2. The tightened precondition is explicit: skill body asserts "parent epic branch must exist and be the current checkout; if missing, run `aiwfx-start-epic E-NNNN` first."
3. The silent `git checkout -b epic/E-NNNN-<slug> origin/main # if missing` fallthrough is removed from the skill body.
4. The skill's `## Workflow` section, parsed structurally, presents the steps in the order specified above.
5. The skill body names the override path (`--force --reason "..."`) at the relevant step.
6. The milestone scope's `aiwf-branch:` trailer records the milestone branch (verified via an end-to-end fixture: run the ritual against a fixture epic, inspect the resulting authorize commit's trailer).
-->

### AC-1 — Embedded snapshot reflects new step ordering

### AC-2 — Skill asserts tightened parent-epic-branch precondition

### AC-3 — Silent fallthrough to checkout -b epic/<slug> if missing removed

### AC-4 — Workflow headings structurally appear in new order

### AC-5 — Skill body names --force --reason override at appropriate step

### AC-6 — Milestone scope aiwf-branch trailer records milestone branch

