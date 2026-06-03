---
id: G-0217
title: aiwf status WRAP PENDING conflates wrap-ritual and trunk-merge pending
status: open
discovered_in: M-0160
---

## What's missing

`aiwf status --worktrees` emits a `WRAP PENDING` label on worktrees whose milestone-branch is ahead of trunk (`refs/remotes/origin/main` per `aiwf.yaml.allocate.trunk` default). The full message:

    WRAP PENDING — driver done but branch ahead of trunk by N commits; merge to trunk before removing

The label and the prose conflate **two distinct states** that have different operator responsibilities:

1. **Wrap-ritual pending** — the milestone is `in_progress` and the operator has work to do: finish ACs, run `aiwfx-wrap-milestone`, fill wrap-side sections, `aiwf promote done`. The operator's action is "do the wrap."

2. **Trunk-merge pending** — the milestone is `done`, the wrap ritual is complete, the milestone branch is merged into its epic-integration branch (per `aiwfx-wrap-milestone` step 11), but trunk has not yet received the work because the epic itself is still in flight. The operator's action is **none** — the epic catches up to trunk via `aiwfx-wrap-epic` when its last milestone wraps.

The current message fires in BOTH cases with identical wording, and the phrase "WRAP PENDING" strongly suggests case 1 (operator forgot to wrap). For case 2 the label is misleading; the milestone IS wrapped.

## Why it matters

Per CLAUDE.md "Kernel functionality must be AI-discoverable": every status surface aiwf exposes must be unambiguous to an LLM operator. The `WRAP PENDING` label fails this:

- An LLM (or human) reading `WRAP PENDING` against a `status: done` milestone naturally assumes "I need to wrap this" and may attempt to re-run wrap rituals, write wrap-side sections again, or otherwise duplicate work.
- The actual signal — *"this branch's commits aren't on trunk yet because the epic is still in flight"* — is invisible from the label alone. The operator has to cross-reference the epic's state to disambiguate.

Concrete instance surfaced during M-0160 wrap (2026-06-03): after wrapping M-0160 (status: done, all 4 ACs met, AC body + wrap-side sections written, ready for merge to epic), the operator ran `aiwf status --worktrees` and observed M-0159 still flagged as `WRAP PENDING`. M-0159 had been wrapped the previous day and merged to `epic/E-0030-...` at commit `e1dc6dc6`; the milestone branch was 0 commits ahead of the epic but 286 commits ahead of trunk. The operator's natural question was "did I forget to wrap M-0159?" — and the answer required a manual `git log epic..M-0159` comparison to discover that no, M-0159 was fully wrapped and the label was just describing the trunk-merge gap.

In a future LLM-driven session, the LLM might NOT do the manual comparison and instead re-run wrap work redundantly. That violates the kernel-correctness principle.

## Proposed fix shape

Two angles — at minimum (a), ideally (a)+(b):

**(a) Disambiguate the label.** Recognize at status-render time whether the milestone is `status: done` and merged to its epic branch (when one exists). Three possible labels per state:

| State | Label |
|---|---|
| Milestone `in_progress`, ACs not all met | `WRAP PENDING — driver still in-progress` (current behavior) |
| Milestone `done`, branch ahead of trunk, NOT merged to epic | `WRAP PENDING — driver done but branch not merged to epic` |
| Milestone `done`, merged to epic, branch ahead of trunk (epic-in-flight) | `AWAITING TRUNK MERGE — driver done, merged to epic, epic in flight` (or equivalent) |
| Milestone `done`, merged to epic, epic merged to trunk | (no label; nothing to surface) |

The status renderer would need to know the milestone's parent epic and check the epic branch's state — the parent reference is already in frontmatter (`parent: E-NNNN`), the epic branch name follows the `epic/E-NNNN-<slug>` convention (per `aiwfx-start-epic`).

**(b) Model the epic-integration branch explicitly in `aiwf.yaml`.** Today the kernel only models `allocate.trunk`. Adding an `allocate.epic_pattern` (e.g., `epic/<id>-<slug>`) would let `aiwf status` recognize epic-integration branches structurally and compute the right "pending" state per the epic-vs-trunk distinction. This is a heavier change but eliminates the heuristic embedded in the renderer.

Path (a) alone solves the immediate signal-conflation problem; path (b) makes the model first-class.

## Test surface

Once a fix lands:

- Fixture milestone: `status: done`, branch merged to epic, epic ahead of trunk by N. Render status → assert label is `AWAITING TRUNK MERGE` (or equivalent), NOT `WRAP PENDING`.
- Fixture milestone: `status: done`, branch ahead of trunk but NOT merged to its epic. Render status → assert label specifies the missed epic merge.
- Fixture milestone: `status: in_progress`. Render status → assert the current behavior (the wrap-ritual is genuinely pending).
- Sabotage-verifiable: revert the epic-merge detection → the wrap/trunk states conflate again; the discriminating test fires.

## Workaround

Until the fix lands, the discipline is operator awareness: when `aiwf status --worktrees` says `WRAP PENDING` on a milestone whose status is `done`, manually verify the milestone-to-epic merge state via `git log epic/E-NNNN..milestone/M-NNNN-...`. If the count is 0 (branch fully merged to epic), the label is reporting trunk-merge-pending, not wrap-pending; no action needed beyond eventual epic wrap.

The `aiwfx-wrap-milestone` skill and/or CLAUDE.md should document the disambiguation explicitly until the fix lands.

## Closing this gap

When the impl lands:
- Status renderer in `internal/cli/status/` (or wherever the worktree summary is composed) recognizes the epic-merged state and labels appropriately.
- Tests above land alongside the implementation.
- CLAUDE.md note (or `aiwfx-wrap-milestone` skill text) removed since the discipline becomes mechanical.
- Promote G-0217 to `addressed` with `--by M-NNNN`.

## Discovered in

M-0160 — observed at wrap time when M-0160 was ready for epic merge and M-0159 (wrapped the previous day, already merged to epic) was still flagged as `WRAP PENDING` against trunk. The operator's natural reading was "M-0159 needs wrap"; the actual reading was "M-0159 is wrapped, trunk just hasn't caught up via epic merge." Same kernel-discoverability shape as G-0216 (empty AC body warning conflates two distinct discipline gaps).
