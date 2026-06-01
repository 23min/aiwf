---
id: M-0104
title: aiwfx-start-epic sequencing fix (closes G-0116)
status: in_progress
parent: E-0030
depends_on:
    - M-0102
    - M-0103
tdd: required
acs:
    - id: AC-1
      title: Embedded snapshot reflects new step ordering
      status: met
      tdd_phase: done
    - id: AC-2
      title: Stale G-0059 paragraph removed; replacement names ADR-0010
      status: open
      tdd_phase: red
    - id: AC-3
      title: Workflow headings structurally appear in new order
      status: open
      tdd_phase: red
    - id: AC-4
      title: Preflight accepts --branch <future> from main (future-branch refinement)
      status: met
      tdd_phase: done
    - id: AC-5
      title: Skill body names --force --reason override at appropriate step
      status: open
      tdd_phase: red
---
## Goal

Reorder `aiwfx-start-epic` so the sovereign promote (`aiwf promote E-NNNN active`) and authorize (`aiwf authorize E-NNNN --to ai/<id> --branch epic/E-NNNN-<slug>`) commits fire on `main` *before* the worktree/branch is cut. Retire the stale "G-0059 frames the open question of which branch-model convention aiwf should bless" paragraph at step 6 — ADR-0010 is the answer. Closes [G-0116](../../gaps/G-0116-aiwfx-start-epic-creates-worktree-before-promote-authorize-on-trunk-based-repos.md).

## Context

G-0116 documented the sequencing inversion in today's `aiwfx-start-epic`: step 5 (worktree placement) precedes step 8 (sovereign promote) and step 9 (optional authorize). With M-0103's preflight active, the existing ordering would *fail* — the worktree-first cut hits the preflight before any ritual branch context exists.

This milestone fixes the ordering so the ritual works with the chokepoint. It also implements [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md)'s sequencing rule for opening an epic: state-announcement commits on main, *then* branch cut, *then* implementation work on the branch.

Ritual content edits land at the canonical authoring location per [ADR-0014](../../../docs/adr/ADR-0014-embed-and-materialize-rituals-distribution-retire-claude-marketplace.md) and [ADR-0016](../../../docs/adr/ADR-0016-retire-ai-workflow-rituals-upstream-channel-embedded-snapshot-canonical.md):

- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md` — the embedded snapshot that `aiwf init` / `aiwf update` materializes into the consumer repo's `.claude/skills/`. Per [G-0182](../../gaps/archive/G-0182-consolidate-testdata-ritual-fixtures-onto-the-embedded-snapshot-dedupe.md), this is also the path the per-AC content-assertion tests read (via `aiwfxStartEpicFixturePath` in `internal/policies/aiwfx_start_epic_test.go`).

One edit in one commit. The upstream `ai-workflow-rituals` repo was archived under ADR-0016 — no cross-repo coordination, no `make sync-rituals`, no `rituals.lock` to refresh.

## Pre-decided design

- **New step ordering** (the reorder):
  1. Preflight (existing steps 1–4).
  2. Delegation prompt (Q&A — promoted earlier so the operator's choice is known *before* the sovereign acts).
  3. Sovereign promote on `main` (or parent branch): `aiwf promote E-NNNN active`.
  4. *(if delegating)* Sovereign authorize on `main`: `aiwf authorize E-NNNN --to ai/<id> --branch epic/E-NNNN-<slug>`. The branch flag is required by M-0103's preflight; the named branch does not yet exist at this point — it is allowed because preflight's "branch exists" check is suppressed when both (a) the current checkout is `main` AND (b) the `--branch` value parses as a ritual shape per `internal/branchparse/` (`epic/`/`milestone/`/`patch/` + a canonical id). Implementation note: M-0103's preflight is refined here — see the carve-out at `internal/verb/authorize.go`. Ritual-shape (not arbitrary "valid ref name") is the stricter test the implementation adopted; without it the gate would become a no-op for any string under `--branch` from `main`. The cell for this is added to the consolidation milestone (M-0158); the carve-out's guard against the looser reading is `TestAuthorize_Open_AITarget_MainPlusNonRitualMissingBranch_Refuses`.
  5. Worktree placement + branch creation (Q&A). The operator picks an in-repo or sibling worktree, the branch is cut against the existing authorize trailer's binding.
  6. Hand-off.
- **Retire the G-0059 paragraph at the original step 6** — replace with a one-line *"per ADR-0010 §"Decision", the operator stays on `main` for the sovereign acts (steps 3–4) and the epic branch is cut afterwards at step 5."*.
- **Step 4's authorize-with-future-branch refinement** is a small extension to M-0103's preflight, not a separate milestone. The cell that proves it is "AI-actor authorize with `--branch epic/E-NNNN-<slug>` on a branch that doesn't yet exist, AND operator is on `main`" → preflight accepts (rationale: this is the ritual's well-formed pattern; cutting the branch is step 5's deliverable). Without this refinement, the ritual could not work — the authorize step would fail M-0103's "branch exists" check.

## Out of scope

- `aiwfx-start-milestone` (M-0105 — symmetric fix one level down).
- Kernel finding for post-hoc detection (M-0106).
- Spec-cell consolidation (the consolidation milestone).
- Other ritual surfaces beyond `aiwfx-start-epic`.

## Dependencies

- **M-0102** — the `--branch` flag and `internal/branchparse/` the new ordering invokes.
- **M-0103** — the preflight that makes the ordering necessary. M-0103's "branch exists OR --branch parses as a ritual shape AND current checkout is on `main`" refinement is *part of* this milestone's deliverable, not a separate prerequisite (the refinement is small enough to land alongside the ritual edit; cell coverage lives in the consolidation milestone).

## Acceptance criteria

<!-- Drafted at `aiwfx-start-milestone M-0104` time. AC seed set:
1. The embedded snapshot at `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md` reflects the new step ordering: preflight → delegation prompt → sovereign promote → sovereign authorize (if delegating) → worktree placement → hand-off.
2. The stale "G-0059 frames the open question" paragraph at the original step 6 is removed; the replacement names ADR-0010 explicitly.
3. The skill's `## Workflow` section's headings, parsed structurally (per CLAUDE.md §"Substring assertions are not structural assertions"), appear in the order specified above. A flat substring match is not sufficient — the assertion is structural.
4. M-0103's preflight accepts `aiwf authorize E-NNNN --to ai/<id> --branch epic/E-NNNN-<slug>` from a checkout on `main` even when the named branch doesn't yet exist, provided `--branch` parses as a ritual shape per `internal/branchparse/`. This is the "future branch" refinement; the cell is registered in the consolidation milestone. (The implementation tightened "valid ref name" from the seed wording to "ritual shape" to keep the gate from becoming a no-op for any string from main; rationale and guard test recorded in pre-decided-design point 4 above.)
5. The skill's "Workflow" prose names the override path (`--force --reason "..."`) at the appropriate step so an operator reading the skill body sees it.
-->

### AC-1 — Embedded snapshot reflects new step ordering

### AC-2 — Stale G-0059 paragraph removed; replacement names ADR-0010

### AC-3 — Workflow headings structurally appear in new order

### AC-4 — Preflight accepts --branch <future> from main (future-branch refinement)

### AC-5 — Skill body names --force --reason override at appropriate step

