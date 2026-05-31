---
id: M-0104
title: aiwfx-start-epic sequencing fix (closes G-0116)
status: draft
parent: E-0030
depends_on:
    - M-0102
    - M-0103
tdd: required
---

## Goal

Reorder `aiwfx-start-epic` so the sovereign promote (`aiwf promote E-NNNN active`) and authorize (`aiwf authorize E-NNNN --to ai/<id> --branch epic/E-NNNN-<slug>`) commits fire on `main` *before* the worktree/branch is cut. Retire the stale "G-0059 frames the open question of which branch-model convention aiwf should bless" paragraph at step 6 — ADR-0010 is the answer. Closes [G-0116](../../gaps/G-0116-aiwfx-start-epic-creates-worktree-before-promote-authorize-on-trunk-based-repos.md).

## Context

G-0116 documented the sequencing inversion in today's `aiwfx-start-epic`: step 5 (worktree placement) precedes step 8 (sovereign promote) and step 9 (optional authorize). With M-0103's preflight active, the existing ordering would *fail* — the worktree-first cut hits the preflight before any ritual branch context exists.

This milestone fixes the ordering so the ritual works with the chokepoint. It also implements [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md)'s sequencing rule for opening an epic: state-announcement commits on main, *then* branch cut, *then* implementation work on the branch.

Cross-repo edits land at **both** authoring locations per CLAUDE.md §"Cross-repo plugin testing" and the embed-and-materialize landing ([ADR-0014](../../../docs/adr/ADR-0014-embed-and-materialize-rituals-distribution-retire-claude-marketplace.md), E-0038):

- `internal/policies/testdata/aiwfx-start-epic/SKILL.md` — the AC fixture, the canonical authoring location for content claims tested by `internal/policies/`.
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md` — the embedded snapshot that `aiwf init` / `aiwf update` materializes into the consumer repo's `.claude/skills/`.

Both edits land in the same commit. `TestRituals_VendoredMatchesUpstream` (M-0148) validates the embedded snapshot against the upstream `ai-workflow-rituals` repo at the pinned `rituals.lock` ref; the per-AC fixture test validates the content claim. After the milestone commits, the upstream `ai-workflow-rituals` repo is updated to match, the lock is refreshed via `make sync-rituals`, and that update commit's SHA is recorded in the milestone's Validation section.

## Pre-decided design

- **New step ordering** (the reorder):
  1. Preflight (existing steps 1–4).
  2. Delegation prompt (Q&A — promoted earlier so the operator's choice is known *before* the sovereign acts).
  3. Sovereign promote on `main` (or parent branch): `aiwf promote E-NNNN active`.
  4. *(if delegating)* Sovereign authorize on `main`: `aiwf authorize E-NNNN --to ai/<id> --branch epic/E-NNNN-<slug>`. The branch flag is required by M-0103's preflight; the named branch does not yet exist at this point — it is allowed because preflight's "branch exists" check applies to the operator's *current* checkout's ritual shape, not to the `--branch`-named target (when both signals are present, `--branch` names the *future* binding and is validated against `git check-ref-format` rather than existence). Implementation note: M-0103's preflight will need a small refinement here — when `--branch` is supplied AND the current checkout is `main`, the named branch is allowed to be future (will be cut at step 5). The cell for this is added to the consolidation milestone.
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
- **M-0103** — the preflight that makes the ordering necessary. M-0103's "branch exists OR --branch names a syntactically-valid future ref AND current checkout is on `main`" refinement is *part of* this milestone's deliverable, not a separate prerequisite (the refinement is small enough to land alongside the ritual edit; cell coverage lives in the consolidation milestone).

## Acceptance criteria

<!-- Drafted at `aiwfx-start-milestone M-0104` time. AC seed set:
1. The testdata fixture at `internal/policies/testdata/aiwfx-start-epic/SKILL.md` reflects the new step ordering: preflight → delegation prompt → sovereign promote → sovereign authorize (if delegating) → worktree placement → hand-off.
2. The embedded snapshot at `internal/skills/embedded-rituals/.../aiwfx-start-epic/SKILL.md` matches the testdata fixture byte-for-byte modulo the embedded-rituals-specific frontmatter.
3. The stale "G-0059 frames the open question" paragraph at the original step 6 is removed; the replacement names ADR-0010 explicitly.
4. The testdata fixture's "## Workflow" section's headings, parsed structurally (per CLAUDE.md §"Substring assertions are not structural assertions"), appear in the order specified above. A flat substring match is not sufficient — the assertion is structural.
5. M-0103's preflight accepts `aiwf authorize E-NNNN --to ai/<id> --branch epic/E-NNNN-<slug>` from a checkout on `main` even when the named branch doesn't yet exist, provided `--branch` parses as a valid ref name. This is the "future branch" refinement; the cell is registered in the consolidation milestone.
6. The skill's "Workflow" prose names the override path (`--force --reason "..."`) at the appropriate step so an operator reading the skill body sees it.
7. The `make sync-rituals` step is documented in the milestone's Validation section once the upstream commit lands; the resulting `rituals.lock` SHA matches the embedded snapshot.
-->
